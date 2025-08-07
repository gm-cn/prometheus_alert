package handler

import (
	"encoding/json"
	"fmt"
	"gpu_alert_forward/kafka"
	"gpu_alert_forward/logger"
	"gpu_alert_forward/model"
	"io"

	"github.com/kataras/iris/v12"
)

// AlertHandler 处理告警相关的请求
type AlertHandler struct {
	producer *kafka.Producer
}

// NewAlertHandler 创建一个新的告警处理器
func NewAlertHandler(producer *kafka.Producer) *AlertHandler {
	return &AlertHandler{
		producer: producer,
	}
}

// HandleAlert 处理来自 Prometheus Alertmanager 的告警
func (h *AlertHandler) HandleAlert(ctx iris.Context) {
	// 读取请求体
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{
			"error": "Failed to read request body",
		})
		return
	}

	// 记录接收到的数据
	logger.Info("Received alert data: %s", string(body))

	// 首先尝试解析为告警数组
	var alerts []model.Alert
	if err := json.Unmarshal(body, &alerts); err != nil {
		// 如果解析数组失败，尝试解析为AlertGroup
		var alertGroup model.AlertGroup
		if err := json.Unmarshal(body, &alertGroup); err != nil {
			logger.Error("Failed to unmarshal alert data: %v", err)
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.JSON(iris.Map{
				"error": "Invalid alert format",
			})
			return
		}
		// 验证并处理AlertGroup
		if err := h.processAlertGroup(&alertGroup); err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.JSON(iris.Map{
				"error": err.Error(),
			})
			return
		}
		ctx.StatusCode(iris.StatusOK)
		ctx.JSON(iris.Map{
			"status":  "success",
			"message": "Processed alert group successfully",
		})
		return
	}

	// 如果是空数组，返回错误
	if len(alerts) == 0 {
		logger.Error("Empty alert array")
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{
			"error": "Empty alert array",
		})
		return
	}

	logger.Info("Processing %d alerts", len(alerts))

	// 创建AlertGroup
	alertGroup := model.CreateAlertGroupFromAlerts(alerts)
	if alertGroup == nil {
		logger.Error("Failed to create alert group from alerts")
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(iris.Map{
			"error": "Failed to create alert group",
		})
		return
	}

	// 处理AlertGroup
	if err := h.processAlertGroup(alertGroup); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{
			"error": err.Error(),
		})
		return
	}

	// 返回成功响应
	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(iris.Map{
		"status":  "success",
		"message": fmt.Sprintf("Processed %d alerts successfully", len(alerts)),
	})
}

// processAlertGroup 处理单个告警组
func (h *AlertHandler) processAlertGroup(group *model.AlertGroup) error {
	// 验证告警数据
	if err := model.ValidateAlertGroup(group); err != nil {
		logger.Error("Invalid alert data: %v", err)
		return err
	}

	// 发送到 Kafka
	if err := h.producer.SendMessage(*group); err != nil {
		logger.Error("Failed to send alert to Kafka: %v", err)
		return fmt.Errorf("failed to process alert")
	}

	logger.Info("Successfully processed alert group")
	return nil
}

// RegisterHandlers 注册所有处理器
func RegisterHandlers(app *iris.Application, producer *kafka.Producer) error {
	handler := NewAlertHandler(producer)

	// 注册路由
	app.Post("/api/v2/alerts", handler.HandleAlert)

	return nil
}
