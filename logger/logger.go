package logger

import (
	"fmt"
	"gpu_alert_forward/config"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *zap.SugaredLogger

// InitLogger 初始化日志配置
func InitLogger(cfg config.LogConfig) error {
	// 创建日志目录
	if cfg.Filename != "" {
		logDir := filepath.Dir(cfg.Filename)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %v", err)
		}
	}

	// 设置日志输出
	var core zapcore.Core
	if cfg.Filename != "" {
		// 文件输出
		writer := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,    // MB
			MaxBackups: cfg.MaxBackups, // 文件个数
			MaxAge:     cfg.MaxAge,     // 天数
			Compress:   cfg.Compress,   // 是否压缩
		}
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(writer),
			getLogLevel(cfg.Level),
		)
	} else {
		// 控制台输出
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			getLogLevel(cfg.Level),
		)
	}

	// 创建 logger
	logger := zap.New(core)
	log = logger.Sugar()
	return nil
}

// getLogLevel 获取日志级别
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Debug 输出 Debug 级别日志
func Debug(format string, args ...interface{}) {
	if log != nil {
		log.Debugf(format, args...)
	}
}

// Info 输出 Info 级别日志
func Info(format string, args ...interface{}) {
	if log != nil {
		log.Infof(format, args...)
	}
}

// Warn 输出 Warn 级别日志
func Warn(format string, args ...interface{}) {
	if log != nil {
		log.Warnf(format, args...)
	}
}

// Error 输出 Error 级别日志
func Error(format string, args ...interface{}) {
	if log != nil {
		log.Errorf(format, args...)
	}
}
