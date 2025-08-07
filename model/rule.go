package model

import (
	"encoding/json"
	"time"
)

// RuleGroup 表示一组告警规则
type RuleGroup struct {
	Name     string      `json:"name"`
	Rules    []AlertRule `json:"rules"`
	Interval Duration    `json:"interval,omitempty"`
}

// AlertRule 表示单个告警规则
type AlertRule struct {
	Alert       string            `json:"alert"`
	Expr        string            `json:"expr"`
	For         Duration          `json:"for,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Duration 是一个自定义的时间持续类型，用于支持 JSON 解析
type Duration struct {
	time.Duration
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(v)
	if err != nil {
		return err
	}

	d.Duration = parsed
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

type RuleServerResponse struct {
	RequestId string      `json:"requestId"`
	Code      int         `json:"code"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data"`
}

// RuleFile 表示完整的规则文件结构
type RuleFile struct {
	Groups []RuleGroup `json:"groups"`
}

type RemoteRuleFile struct {
	Groups []RuleGroup `json:"groups"`
}

// CompareRules 比较两个规则文件是否相同
func CompareRules(local, remote *RuleFile) bool {
	if local == nil || remote == nil {
		return false
	}

	if len(local.Groups) != len(remote.Groups) {
		return false
	}

	// 创建本地规则组的映射，用于快速查找
	localGroups := make(map[string]RuleGroup)
	for _, group := range local.Groups {
		localGroups[group.Name] = group
	}

	// 比较每个远程规则组
	for _, remoteGroup := range remote.Groups {
		localGroup, exists := localGroups[remoteGroup.Name]
		if !exists {
			return false
		}

		if !compareRuleGroup(localGroup, remoteGroup) {
			return false
		}
	}

	return true
}

// compareRuleGroup 比较两个规则组是否相同
func compareRuleGroup(local, remote RuleGroup) bool {
	if len(local.Rules) != len(remote.Rules) {
		return false
	}

	// 创建本地规则的映射，用于快速查找
	localRules := make(map[string]AlertRule)
	for _, rule := range local.Rules {
		localRules[rule.Alert] = rule
	}

	// 比较每个远程规则
	for _, remoteRule := range remote.Rules {
		localRule, exists := localRules[remoteRule.Alert]
		if !exists {
			return false
		}

		if !compareRule(localRule, remoteRule) {
			return false
		}
	}

	return true
}

// compareRule 比较两个规则是否相同
func compareRule(local, remote AlertRule) bool {
	if local.Alert != remote.Alert || local.Expr != remote.Expr {
		return false
	}

	if local.For != remote.For {
		return false
	}

	if !compareStringMap(local.Labels, remote.Labels) {
		return false
	}

	if !compareStringMap(local.Annotations, remote.Annotations) {
		return false
	}

	return true
}

// compareStringMap 比较两个字符串映射是否相同
func compareStringMap(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v1 := range m1 {
		if v2, ok := m2[k]; !ok || v1 != v2 {
			return false
		}
	}

	return true
}
