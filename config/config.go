package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config 配置结构
type Config struct {
	Server     ServerConfig     `json:"server"`
	Kafka      KafkaConfig      `json:"kafka"`
	Log        LogConfig        `json:"log"`
	Prometheus PrometheusConfig `json:"prometheus"` // 新增 Prometheus 配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `json:"port"`
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers   []string `json:"brokers"`
	Topic     string   `json:"topic"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Mechanism string   `json:"mechanism"`
	Protocol  string   `json:"protocol"`
	Timeout   Duration `json:"timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `json:"level"`
	Filename   string `json:"filename"`
	MaxSize    int    `json:"maxSize"`    // 每个日志文件的最大大小（MB）
	MaxBackups int    `json:"maxBackups"` // 保留的旧日志文件最大数量
	MaxAge     int    `json:"maxAge"`     // 保留的旧日志文件的最大天数
	Compress   bool   `json:"compress"`   // 是否压缩旧日志文件
}

// PrometheusConfig Prometheus 相关配置
type PrometheusConfig struct {
	RemoteURL    string   `json:"remoteUrl"`    // 远程 Prometheus 服务器地址
	RuleFilePath string   `json:"ruleFilePath"` // 本地规则文件路径
	ReloadURL    string   `json:"reloadUrl"`    // Prometheus reload 接口地址
	SyncInterval Duration `json:"syncInterval"` // 同步间隔时间
}

// Duration 是一个自定义的时间持续类型，用于支持 JSON 解析
type Duration struct {
	time.Duration
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		return nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		return nil
	}

	return &config
}
