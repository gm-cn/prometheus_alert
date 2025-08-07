package main

import (
	"context"
	"flag"
	"fmt"
	"gpu_alert_forward/config"
	"gpu_alert_forward/handler"
	"gpu_alert_forward/kafka"
	"gpu_alert_forward/logger"
	"os"
	"os/signal"
	"syscall"

	"github.com/kataras/iris/v12"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// 加载配置
	cfg := config.LoadConfig(*configPath)
	if cfg == nil {
		fmt.Printf("Failed to load config from file: %s\n", *configPath)
		os.Exit(1)
	}
	fmt.Printf("Loaded config from file: %s\n", *configPath)

	// 初始化日志
	if err := logger.InitLogger(cfg.Log); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// 创建 Kafka 生产者
	producer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		logger.Error("Failed to create Kafka producer: %v", err)
		os.Exit(1)
	}
	defer producer.Close()

	// 创建 Iris 应用
	app := iris.New()

	// 注册路由
	if err := handler.RegisterHandlers(app, producer); err != nil {
		logger.Error("Failed to register handlers: %v", err)
		os.Exit(1)
	}

	// 设置优雅关闭
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info("Server is shutting down...")
		if err := app.Shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown server: %v", err)
		}
		close(done)
	}()

	// 启动服务器
	logger.Info("Starting server on :%d", cfg.Server.Port)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		logger.Error("Error starting server: %v", err)
		os.Exit(1)
	}

	<-done
	logger.Info("Server stopped")
}
