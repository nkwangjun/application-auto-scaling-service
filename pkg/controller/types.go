package controller

import (
	"nanto.io/application-auto-scaling-service/pkg/apis/autoscaling/v1alpha1"
)

const (
	// 策略 执行动作（actions） 相关
	operationTypeScaleUp   = "ScaleUp"
	operationTypeScaleDown = "ScaleDown"
	operationUnitTask      = "Task"

	// 策略 触发条件（metricTrigger） 相关
	MetricOptScaleUp            = ">"
	MetricOptScaleDown          = "<"
	metricNameCPURatioToRequest = "CPURatioToRequest"
	statisticInstantaneous      = "instantaneous"

	/// 策略 规则（Rule） 相关
	ruleTypeMetric = "Metric"
)

var (
	defaultHitThreshold     int32 = 1
	defaultPeriodSeconds    int32 = 60
	defaultRuleDisableFalse       = false
)

type StrategiesInfo struct {
	ClusterId string `yaml:"clusterId"`
	// 策略创建时间，时间戳（毫秒）
	CreateTime int64 `yaml:"createTime"`
	// 目标HPA
	TargetHPA  string     `yaml:"targetHPA"`
	Strategies []Strategy `yaml:"strategies"`
}

type Strategy struct {
	// 生效时间段，eg："0:00-09:30"
	ValidTime string                                       `yaml:"validTime"`
	Spec      v1alpha1.CustomedHorizontalPodAutoscalerSpec `yaml:"spec"`
}

// todo 后面将yaml解析 和 k8s api server 请求结构体解耦
// CheckAndCompleteInfo 校验用户输入的 strategies 信息是否合法，并补全信息
func (s *StrategiesInfo) CheckAndCompleteInfo() error {
	// todo 参数校验
	for i := 0; i < len(s.Strategies); i++ {
		completeRules(&s.Strategies[i])
	}
	return nil
}

// completeRules 补全策略的规则（Rules）信息
func completeRules(strategy *Strategy) {
	for i := 0; i < len(strategy.Spec.Rules); i++ {
		// 执行动作
		completeRuleActions(&strategy.Spec.Rules[i])

		// 是否启用 Disable 默认 false 即可
		strategy.Spec.Rules[i].Disable = &defaultRuleDisableFalse

		// 规则触发条件
		// hitThreshold: 1
		strategy.Spec.Rules[i].MetricTrigger.HitThreshold = &defaultHitThreshold
		// metricName: CPURatioToRequest
		strategy.Spec.Rules[i].MetricTrigger.MetricName = metricNameCPURatioToRequest
		// periodSeconds: 60
		strategy.Spec.Rules[i].MetricTrigger.PeriodSeconds = &defaultPeriodSeconds
		// statistic: instantaneous
		strategy.Spec.Rules[i].MetricTrigger.Statistic = statisticInstantaneous

		// 规则类型
		// ruleType: Metric
		strategy.Spec.Rules[i].RuleType = ruleTypeMetric
	}
}

// 补全规则中执行动作（actions）信息
func completeRuleActions(rule *v1alpha1.Rule) {
	optType := ""
	if rule.MetricTrigger.MetricOperation == MetricOptScaleUp { // metricOperation: ">"
		optType = operationTypeScaleUp
	} else if rule.MetricTrigger.MetricOperation == MetricOptScaleDown { // metricOperation: "<"
		optType = operationTypeScaleDown
	}
	// todo rule.MetricTrigger.MetricOperation 不合法情况，在上一个步骤，参数检查时做

	for i := 0; i < len(rule.Actions); i++ {
		// operationType: ScaleUp / ScaleDown
		rule.Actions[i].OperationType = optType
		// operationUnit: Task
		rule.Actions[i].OperationUnit = operationUnitTask
	}
}
