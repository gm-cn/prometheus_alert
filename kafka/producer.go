package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"gpu_alert_forward/config"
	"gpu_alert_forward/logger"
	"gpu_alert_forward/model"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// SHA256 实现 SCRAM-SHA-256 认证
type SHA256 struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

// SHA512 实现 SCRAM-SHA-512 认证
type SHA512 struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	// 创建 Kafka 配置
	config := sarama.NewConfig()

	// 设置生产者配置
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	// 设置认证信息
	if cfg.Username != "" && cfg.Password != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = cfg.Username
		config.Net.SASL.Password = cfg.Password

		// 设置认证机制
		switch cfg.Mechanism {
		case "PLAIN":
			config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		case "SCRAM-SHA-256":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &SHA256{HashGeneratorFcn: sha256.New}
			}
		case "SCRAM-SHA-512":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &SHA512{HashGeneratorFcn: sha512.New}
			}
		default:
			return nil, fmt.Errorf("unsupported SASL mechanism: %s", cfg.Mechanism)
		}

		// 设置协议
		switch cfg.Protocol {
		case "PLAINTEXT":
			config.Net.TLS.Enable = false
		case "SSL":
			config.Net.TLS.Enable = true
		default:
			return nil, fmt.Errorf("unsupported security protocol: %s", cfg.Protocol)
		}
	}

	// 设置超时时间
	if cfg.Timeout.Duration > 0 {
		config.Net.DialTimeout = cfg.Timeout.Duration
		config.Net.ReadTimeout = cfg.Timeout.Duration
		config.Net.WriteTimeout = cfg.Timeout.Duration
	}

	// 创建生产者
	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		if len(cfg.Brokers) == 1 && cfg.Brokers[0] == "localhost:9092" {
			logger.Warn("Failed to connect to local Kafka, messages will be logged only: %v", err)
			return &Producer{
				producer: nil,
				topic:    cfg.Topic,
			}, nil
		}
		return nil, fmt.Errorf("failed to create kafka producer: %v", err)
	}

	return &Producer{
		producer: producer,
		topic:    cfg.Topic,
	}, nil
}

func (p *Producer) Close() error {
	if p.producer != nil {
		if err := p.producer.Close(); err != nil {
			return fmt.Errorf("failed to close Kafka producer: %v", err)
		}
	}
	return nil
}

func (p *Producer) SendMessage(alert model.AlertGroup) error {
	// 序列化告警消息
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert message: %v", err)
	}

	// 如果没有可用的 Kafka 生产者，只记录日志
	if p.producer == nil {
		logger.Info("[DRY RUN] Would send message to topic %s: %s", p.topic, string(data))
		return nil
	}

	// 创建消息
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.ByteEncoder(data),
	}

	// 发送消息
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	logger.Info("Message sent successfully to partition %d at offset %d", partition, offset)
	return nil
}

// SCRAM 认证相关方法实现
func (x *SHA256) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

func (x *SHA256) Step(challenge string) (response string, err error) {
	response, err = x.ClientConversation.Step(challenge)
	return
}

func (x *SHA256) Done() bool {
	return x.ClientConversation.Done()
}

func (x *SHA512) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

func (x *SHA512) Step(challenge string) (response string, err error) {
	response, err = x.ClientConversation.Step(challenge)
	return
}

func (x *SHA512) Done() bool {
	return x.ClientConversation.Done()
}
