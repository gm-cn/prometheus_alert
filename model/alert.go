package model

import (
	"fmt"
	"time"
)

// AlertGroup 表示一组告警
type AlertGroup struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
	Timestamp         string            `json:"timestamp"`
}

// Alert 表示单个告警
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// CreateAlertGroupFromAlerts 从告警数组创建告警组
func CreateAlertGroupFromAlerts(alerts []Alert) *AlertGroup {
	if len(alerts) == 0 {
		return nil
	}

	// 使用第一个告警的信息作为公共标签
	firstAlert := alerts[0]
	return &AlertGroup{
		Version:     "4", // Prometheus 2.53.4
		Status:      "firing",
		GroupLabels: firstAlert.Labels,
		Alerts:      alerts,
		Timestamp:   time.Now().Format(time.DateTime),
	}
}

// ValidateAlertGroup 验证告警组数据
func ValidateAlertGroup(group *AlertGroup) error {
	if group == nil {
		return fmt.Errorf("alert group is nil")
	}

	// 验证所有告警
	for i, alert := range group.Alerts {
		if err := validateAlert(&alert); err != nil {
			return fmt.Errorf("invalid alert at index %d: %v", i, err)
		}
	}

	return nil
}

// validateAlert 验证单个告警数据
func validateAlert(alert *Alert) error {
	if alert == nil {
		return fmt.Errorf("alert is nil")
	}

	// 不再强制要求status字段
	if alert.Labels == nil {
		alert.Labels = make(map[string]string)
	}

	return nil
}
