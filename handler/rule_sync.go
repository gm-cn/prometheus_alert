package handler

import (
	"encoding/json"
	"fmt"
	"gpu_alert_forward/config"
	"gpu_alert_forward/logger"
	"gpu_alert_forward/model"
	"io"
	"net/http"
	"os"
	"time"
)

// RuleSyncHandler 处理规则同步的处理器
type RuleSyncHandler struct {
	config *config.Config
	client *http.Client
	done   chan struct{}
}

// NewRuleSyncHandler 创建一个新的规则同步处理器
func NewRuleSyncHandler(cfg *config.Config) *RuleSyncHandler {
	return &RuleSyncHandler{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		done: make(chan struct{}),
	}
}

// Start 启动规则同步处理器
func (h *RuleSyncHandler) Start() {
	ticker := time.NewTicker(h.config.Prometheus.SyncInterval.Duration)
	go func() {
		// 启动时立即执行一次同步
		if err := h.syncRules(); err != nil {
			logger.Error("Failed to sync rules: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := h.syncRules(); err != nil {
					logger.Error("Failed to sync rules: %v", err)
				}
			case <-h.done:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop 停止规则同步处理器
func (h *RuleSyncHandler) Stop() {
	close(h.done)
}

// syncRules 同步规则
func (h *RuleSyncHandler) syncRules() error {
	// 获取远程规则
	remoteRules, err := h.fetchRemoteRules()
	if err != nil {
		return fmt.Errorf("failed to fetch remote rules: %v", err)
	}

	// 读取本地规则
	localRules, err := h.readLocalRules()
	if err != nil {
		return fmt.Errorf("failed to read local rules: %v", err)
	}

	// 如果规则相同，不需要更新
	if model.CompareRules(localRules, remoteRules) {
		logger.Info("Rules are identical, no update needed")
		return nil
	}

	// 更新本地规则文件
	if err := h.updateLocalRules(remoteRules); err != nil {
		return fmt.Errorf("failed to update local rules: %v", err)
	}

	// 重新加载 Prometheus
	if err := h.reloadPrometheus(); err != nil {
		return fmt.Errorf("failed to reload Prometheus: %v", err)
	}

	logger.Info("Successfully synced rules and reloaded Prometheus")
	return nil
}

func (h *RuleSyncHandler) fetchRemoteRules() (*model.RuleFile, error) {
	resp, err := h.client.Get(h.config.Prometheus.RemoteURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res model.RuleServerResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if res.Code != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, rid: %s, msg: %s", res.Code, res.RequestId, res.Msg)
	}

	var rules model.RuleFile
	tData, _ := json.Marshal(res.Data)
	if err := json.Unmarshal(tData, &rules); err != nil {
		return nil, err
	}

	return &rules, nil
}

// readLocalRules 读取本地规则文件
func (h *RuleSyncHandler) readLocalRules() (*model.RuleFile, error) {
	data, err := os.ReadFile(h.config.Prometheus.RuleFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果文件不存在，返回空的规则文件
			return &model.RuleFile{Groups: []model.RuleGroup{}}, nil
		}
		return nil, err
	}

	var rules model.RuleFile
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}

	return &rules, nil
}

// updateLocalRules 更新本地规则文件
func (h *RuleSyncHandler) updateLocalRules(rules *model.RuleFile) error {
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(h.config.Prometheus.RuleFilePath, data, 0644)
}

// reloadPrometheus 重新加载 Prometheus
func (h *RuleSyncHandler) reloadPrometheus() error {
	resp, err := h.client.Post(h.config.Prometheus.ReloadURL, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
