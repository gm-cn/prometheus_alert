package model

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"time"
)

// AlarmExpression 表示告警表达式
type AlarmExpression struct {
	Op          string   `json:"op"`
	Path        string   `json:"path"`
	Type        int      `json:"type"`
	Value       string   `json:"value"`
	GapOp       string   `json:"gap_op"`
	GapValue    string   `json:"gap_value"`
	ValueType   int      `json:"value_type"`
	ValueExcept []string `json:"value_except"`
}

// AnalysisRule 表示分析规则
type AnalysisRule struct {
	EndTime               int               `json:"end_time"`
	AlarmType             int               `json:"alarm_type"`
	StartTime             int               `json:"start_time"`
	AlarmLogic            interface{}       `json:"alarm_logic"`
	AlarmPeriod           int               `json:"alarm_period"`
	RecoverType           int               `json:"recover_type"`
	RecoverLogic          interface{}       `json:"recover_logic"`
	StrategyType          int               `json:"strategy_type"`
	RecoverPeriod         int               `json:"recover_period"`
	AlarmPointList        interface{}       `json:"alarm_point_list"`
	SyslogLevelEnd        int               `json:"syslog_level_end"`
	AlarmExpressions      []AlarmExpression `json:"alarm_expressions"`
	AlarmTouchCount       int               `json:"alarm_touch_count"`
	RecoverPointList      interface{}       `json:"recover_point_list"`
	SyslogLevelStart      int               `json:"syslog_level_start"`
	RecoverExpressions    []AlarmExpression `json:"recover_expressions"`
	RecoverTouchCount     int               `json:"recover_touch_count"`
	AlarmPeriodContinue   int               `json:"alarm_period_continue"`
	RecoverConditionList  interface{}       `json:"recover_condition_list"`
	TriggerConditionList  interface{}       `json:"trigger_condition_list"`
	RecoverPeriodContinue int               `json:"recover_period_continue"`
	SyslogAlarmLogMinute  int               `json:"syslog_alarm_log_minute"`
}

// Application 表示应用配置
type Application struct {
	Feature         string      `json:"feature"`
	Tags            []string    `json:"tags"`
	BasicID         int         `json:"basic_id"`
	RuleID          int         `json:"rule_id"`
	FaultTypeID     int         `json:"fault_type_id"`
	AutoPlanID      string      `json:"auto_plan_id"`
	Extension       interface{} `json:"extension"`
	Priority        int         `json:"priority"`
	KeywordID       int         `json:"keyword_id"`
	EventType       int         `json:"event_type"`
	UserType        int         `json:"user_type"`
	NoticeChannel   int         `json:"notice_channel"`
	IsPushFaultPool int         `json:"is_push_fault_pool"`
}

// ServerRule 表示服务器规则
type ServerRule struct {
	ID             int          `json:"id"`
	BasicID        int          `json:"basicId"`
	Category       string       `json:"category"`
	FaultTypeID    int          `json:"faultTypeId"`
	StrategyName   string       `json:"strategyName"`
	Status         int          `json:"status"`
	AnalysisRule   AnalysisRule `json:"analysisRule"`
	Label          string       `json:"label"`
	AutoPlanID     string       `json:"autoPlanId"`
	Priority       int          `json:"priority"`
	CreateBy       string       `json:"createBy"`
	UpdateBy       string       `json:"updateBy"`
	Notices        interface{}  `json:"notices"`
	RepeatUpgrade  interface{}  `json:"repeatUpgrade"`
	DeepUpgrade    interface{}  `json:"deepUpgrade"`
	DragUpgrade    interface{}  `json:"dragUpgrade"`
	Event          interface{}  `json:"event"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
	LevelName      string       `json:"levelName"`
	EventLevelName string       `json:"event_level_name"`
	Application    Application  `json:"application"`
	Extract        interface{}  `json:"extract"`
}

// RuleGroup 表示一组告警规则
type RuleGroup struct {
	Name     string      `json:"name" yaml:"name"`
	Rules    []AlertRule `json:"rules" yaml:"rules"`
	Interval Duration    `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// MarshalYAML 自定义 RuleGroup 的 YAML 序列化
func (g RuleGroup) MarshalYAML() (interface{}, error) {
	type Alias RuleGroup
	return struct {
		Alias
		Rules yaml.Node `yaml:"rules"`
	}{
		Alias: Alias(g),
		Rules: yaml.Node{
			Kind:  yaml.SequenceNode,
			Style: yaml.FlowStyle,
			Tag:   "!!seq",
			Content: func() []*yaml.Node {
				var nodes []*yaml.Node
				for _, rule := range g.Rules {
					ruleNode := &yaml.Node{
						Kind:  yaml.MappingNode,
						Style: yaml.FlowStyle,
						Tag:   "!!map",
					}
					if err := ruleNode.Encode(rule); err != nil {
						continue
					}
					nodes = append(nodes, ruleNode)
				}
				return nodes
			}(),
		},
	}, nil
}

// AlertRule 表示单个告警规则
type AlertRule struct {
	Alert       string            `json:"alert" yaml:"alert"`
	Expr        string            `json:"expr" yaml:"expr"`
	For         Duration          `json:"for,omitempty" yaml:"for,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// MarshalYAML 自定义 AlertRule 的 YAML 序列化
func (r AlertRule) MarshalYAML() (interface{}, error) {
	type Alias AlertRule
	return struct {
		Alias
		Labels      yaml.Node `yaml:"labels,omitempty"`
		Annotations yaml.Node `yaml:"annotations,omitempty"`
	}{
		Alias: Alias(r),
		Labels: yaml.Node{
			Kind:  yaml.MappingNode,
			Style: yaml.FlowStyle,
			Value: "",
			Tag:   "!!map",
			Content: func() []*yaml.Node {
				var nodes []*yaml.Node
				for k, v := range r.Labels {
					nodes = append(nodes, &yaml.Node{
						Kind:  yaml.ScalarNode,
						Style: 0,
						Value: k,
						Tag:   "!!str",
					}, &yaml.Node{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: v,
						Tag:   "!!str",
					})
				}
				return nodes
			}(),
		},
		Annotations: yaml.Node{
			Kind:  yaml.MappingNode,
			Style: yaml.FlowStyle,
			Value: "",
			Tag:   "!!map",
			Content: func() []*yaml.Node {
				var nodes []*yaml.Node
				for k, v := range r.Annotations {
					nodes = append(nodes, &yaml.Node{
						Kind:  yaml.ScalarNode,
						Style: 0,
						Value: k,
						Tag:   "!!str",
					}, &yaml.Node{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: v,
						Tag:   "!!str",
					})
				}
				return nodes
			}(),
		},
	}, nil
}

// Duration 是一个自定义的时间持续类型，用于支持 JSON 和 YAML 解析
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

// UnmarshalYAML 实现 yaml.Unmarshaler 接口
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v string
	if err := unmarshal(&v); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(v)
	if err != nil {
		return err
	}

	d.Duration = parsed
	return nil
}

// MarshalYAML 实现 yaml.Marshaler 接口
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

type RuleServerResponse struct {
	RequestId string      `json:"requestId"`
	Code      int         `json:"code"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data"`
}

// RuleFile 表示完整的规则文件结构
type RuleFile struct {
	Groups []RuleGroup `json:"groups" yaml:"groups"`
}

// MarshalYAML 自定义 RuleFile 的 YAML 序列化
func (f RuleFile) MarshalYAML() (interface{}, error) {
	type Alias RuleFile
	return struct {
		Alias
		Groups yaml.Node `yaml:"groups"`
	}{
		Alias: Alias(f),
		Groups: yaml.Node{
			Kind:  yaml.SequenceNode,
			Style: yaml.FlowStyle,
			Tag:   "!!seq",
			Content: func() []*yaml.Node {
				var nodes []*yaml.Node
				for _, group := range f.Groups {
					groupNode := &yaml.Node{
						Kind:  yaml.MappingNode,
						Style: yaml.FlowStyle,
						Tag:   "!!map",
					}
					if err := groupNode.Encode(group); err != nil {
						continue
					}
					nodes = append(nodes, groupNode)
				}
				return nodes
			}(),
		},
	}, nil
}

type RemoteRuleFile struct {
	Groups []ServerRule
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

	if !compareStringMap(map[string]string(local.Annotations), map[string]string(remote.Annotations)) {
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
