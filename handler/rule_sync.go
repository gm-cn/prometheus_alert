package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"gpu_alert_forward/config"
	"gpu_alert_forward/logger"
	"gpu_alert_forward/model"
	"io"
	"net/http"
	"os"
	"strconv"
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
	for _, prometheusConfig := range h.config.Prometheus {
		ticker := time.NewTicker(prometheusConfig.SyncInterval.Duration)
		go func() {
			// 启动时立即执行一次同步
			if err := h.syncRules(prometheusConfig); err != nil {
				logger.Error("Failed to sync rules: %v", err)
			}

			for {
				select {
				case <-ticker.C:
					if err := h.syncRules(prometheusConfig); err != nil {
						logger.Error("Failed to sync rules: %v", err)
					}
				case <-h.done:
					ticker.Stop()
					return
				}
			}
		}()
	}

	//ticker := time.NewTicker(h.config.Prometheus[0].SyncInterval.Duration)
	//go func() {
	//	// 启动时立即执行一次同步
	//	if err := h.syncRules(); err != nil {
	//		logger.Error("Failed to sync rules: %v", err)
	//	}
	//
	//	for {
	//		select {
	//		case <-ticker.C:
	//			if err := h.syncRules(); err != nil {
	//				logger.Error("Failed to sync rules: %v", err)
	//			}
	//		case <-h.done:
	//			ticker.Stop()
	//			return
	//		}
	//	}
	//}()
}

// Stop 停止规则同步处理器
func (h *RuleSyncHandler) Stop() {
	close(h.done)
}

// syncRules 同步规则
func (h *RuleSyncHandler) syncRules(prometheusConfig config.PrometheusConfig) error {
	// 获取远程规则
	remoteRules, err := h.fetchRemoteRules(prometheusConfig)
	if err != nil {
		return fmt.Errorf("failed to fetch remote rules: %v", err)
	}

	// 读取本地规则
	localRules, err := h.readLocalRules(prometheusConfig)
	if err != nil {
		return fmt.Errorf("failed to read local rules: %v", err)
	}

	// 如果规则相同，不需要更新
	if model.CompareRules(localRules, remoteRules) {
		logger.Info("Rules are identical, no update needed")
		return nil
	}

	// 更新本地规则文件
	if err := h.updateLocalRules(remoteRules, prometheusConfig); err != nil {
		return fmt.Errorf("failed to update local rules: %v", err)
	}

	// 重新加载 Prometheus
	if err := h.reloadPrometheus(prometheusConfig); err != nil {
		return fmt.Errorf("failed to reload Prometheus: %v", err)
	}

	logger.Info("Successfully synced rules and reloaded Prometheus")
	return nil
}

func (h *RuleSyncHandler) fetchRemoteRules(prometheusConfig config.PrometheusConfig) (*model.RuleFile, error) {
	requestBody := map[string]interface{}{
		"strategyType": []int{0, 1},
		"category":     []string{prometheusConfig.RuleTag},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", prometheusConfig.RemoteURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unexpected client.Do: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unexpected io.ReadAll: %v", err.Error())
	}

	var res model.RuleServerResponse
	if err = json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("unexpected json.Unmarshal RuleServerResponse: %v", err.Error())
	}
	if res.Code != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, rid: %s, msg: %s", res.Code, res.RequestId, res.Msg)
	}

	var remoteRules []model.ServerRule
	tData, err := json.Marshal(res.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal res.Data: %v", err.Error())
	}
	if err = json.Unmarshal(tData, &remoteRules); err != nil {
		return nil, fmt.Errorf("unexpected json.Unmarshal remoteRules: %v", err.Error())
	}

	localRuleGroup := model.RuleGroup{Name: "gpu-alerts"}
	for _, remoteRule := range remoteRules {
		tRuleGroup := model.AlertRule{
			Alert: remoteRule.StrategyName,
			Expr:  fmt.Sprintf("%s %s %s", remoteRule.AnalysisRule.AlarmExpressions[0].Path, remoteRule.AnalysisRule.AlarmExpressions[0].Op, remoteRule.AnalysisRule.AlarmExpressions[0].Value),
			Labels: map[string]string{
				"rule_id": strconv.Itoa(remoteRule.ID),
			},
			Annotations: map[string]string{
				"value": `{{ $value }}`,
			},
		}
		localRuleGroup.Rules = append(localRuleGroup.Rules, tRuleGroup)
	}

	localRule := new(model.RuleFile)
	localRule.Groups = []model.RuleGroup{localRuleGroup}
	return localRule, nil
}

// readLocalRules 读取本地规则文件
func (h *RuleSyncHandler) readLocalRules(prometheusConfig config.PrometheusConfig) (*model.RuleFile, error) {
	logger.Info("Reading rules from %s", prometheusConfig.RuleFilePath)
	data, err := os.ReadFile(prometheusConfig.RuleFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果文件不存在，返回空的规则文件
			return &model.RuleFile{Groups: []model.RuleGroup{}}, nil
		}
		return nil, err
	}

	var rules model.RuleFile
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	return &rules, nil
}

// updateLocalRules 更新本地规则文件
func (h *RuleSyncHandler) updateLocalRules(rules *model.RuleFile, prometheusConfig config.PrometheusConfig) error {
	// 创建一个新的 yaml.Node 来控制输出格式
	node := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Value: "groups",
					},
					{
						Kind: yaml.SequenceNode,
						Content: func() []*yaml.Node {
							var nodes []*yaml.Node
							for _, group := range rules.Groups {
								groupNode := &yaml.Node{
									Kind: yaml.MappingNode,
									Content: []*yaml.Node{
										{
											Kind:  yaml.ScalarNode,
											Value: "name",
										},
										{
											Kind:  yaml.ScalarNode,
											Value: group.Name,
										},
										{
											Kind:  yaml.ScalarNode,
											Value: "rules",
										},
										{
											Kind: yaml.SequenceNode,
											Content: func() []*yaml.Node {
												var ruleNodes []*yaml.Node
												for _, rule := range group.Rules {
													ruleNode := &yaml.Node{
														Kind: yaml.MappingNode,
														Content: []*yaml.Node{
															{
																Kind:  yaml.ScalarNode,
																Value: "alert",
															},
															{
																Kind:  yaml.ScalarNode,
																Value: rule.Alert,
															},
															{
																Kind:  yaml.ScalarNode,
																Value: "expr",
															},
															{
																Kind:  yaml.ScalarNode,
																Value: rule.Expr,
															},
														},
													}

													// 添加 labels
													if len(rule.Labels) > 0 {
														labelNodes := []*yaml.Node{
															{
																Kind:  yaml.ScalarNode,
																Value: "labels",
															},
															{
																Kind: yaml.MappingNode,
															},
														}
														for k, v := range rule.Labels {
															labelNodes[1].Content = append(labelNodes[1].Content,
																&yaml.Node{
																	Kind:  yaml.ScalarNode,
																	Value: k,
																},
																&yaml.Node{
																	Kind:  yaml.ScalarNode,
																	Value: v,
																})
														}
														ruleNode.Content = append(ruleNode.Content, labelNodes...)
													}

													// 添加 annotations
													if len(rule.Annotations) > 0 {
														annotationNodes := []*yaml.Node{
															{
																Kind:  yaml.ScalarNode,
																Value: "annotations",
															},
															{
																Kind: yaml.MappingNode,
															},
														}
														for k, v := range rule.Annotations {
															annotationNodes[1].Content = append(annotationNodes[1].Content,
																&yaml.Node{
																	Kind:  yaml.ScalarNode,
																	Value: k,
																},
																&yaml.Node{
																	Kind:  yaml.ScalarNode,
																	Value: v,
																})
														}
														ruleNode.Content = append(ruleNode.Content, annotationNodes...)
													}

													ruleNodes = append(ruleNodes, ruleNode)
												}
												return ruleNodes
											}(),
										},
									},
								}
								nodes = append(nodes, groupNode)
							}
							return nodes
						}(),
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(node); err != nil {
		return fmt.Errorf("failed to encode rules to YAML: %v", err)
	}

	encoder.Close()
	return os.WriteFile(prometheusConfig.RuleFilePath, buf.Bytes(), 0644)
}

// reloadPrometheus 重新加载 Prometheus
func (h *RuleSyncHandler) reloadPrometheus(prometheusConfig config.PrometheusConfig) error {
	resp, err := h.client.Post(prometheusConfig.ReloadURL, "application/json", nil)
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
