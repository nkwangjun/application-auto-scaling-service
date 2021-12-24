package policy

import (
	"github.com/pkg/errors"
	"nanto.io/application-auto-scaling-service/pkg/metricwatch"
	"nanto.io/application-auto-scaling-service/pkg/resourceadapter/scalinggroup"
	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

var logger = logutil.GetLogger()

type Policy struct {
	name                  string
	fleetId               string
	fleetArn              string
	location              string
	metricName            string
	policyType            string
	scalingAdjustmentType string
	threshold             float32
	scalingAdjustment     int
	comparisonOperator    string
	evaluationPeriods     int
	quitChan              chan struct{}
	groupInterface        scalinggroup.ScalingGroup
}

func NewPolicy(name string, fleetId string, metricName string, policyType string, scalingAdjustmentType string,
	threshold float32, scalingAdjustment int, comparisonOperator string, evaluationPeriods int,
	groupInterface scalinggroup.ScalingGroup) *Policy {
	return &Policy{
		name:                  name,
		fleetId:               fleetId,
		metricName:            metricName,
		policyType:            policyType,
		scalingAdjustmentType: scalingAdjustmentType,
		threshold:             threshold,
		scalingAdjustment:     scalingAdjustment,
		comparisonOperator:    comparisonOperator,
		// 评估周期, 指标需要持续满足触发条件才会执行扩缩容
		evaluationPeriods: evaluationPeriods,
		quitChan:          make(chan struct{}),
		groupInterface:    groupInterface,
	}
}

func (p *Policy) Start() {
	go metricwatch.NewMetricAlarm(p.fleetId, p.metricName, p.comparisonOperator, p.evaluationPeriods, p.threshold,
		p.DoAction).Run(p.quitChan)
}

func (p *Policy) Stop() {
	// stop policy
	p.quitChan <- struct{}{}
}

func (p *Policy) Update(metricName string, policyType string, scalingAdjustmentType string,
	threshold float32, scalingAdjustment int, comparisonOperator string, evaluationPeriods int) {
	// 停止策略
	p.Stop()

	// 更新策略
	p.metricName = metricName
	p.policyType = policyType
	p.scalingAdjustmentType = scalingAdjustmentType
	p.scalingAdjustment = scalingAdjustment
	p.threshold = threshold
	p.comparisonOperator = comparisonOperator
	p.evaluationPeriods = evaluationPeriods

	// 启动策略
	p.Start()
}

func (p *Policy) DoAction() error {
	// Do scaling
	switch p.scalingAdjustmentType {
	case "ChangeInCapacity":
		// Scale out
		if p.scalingAdjustment > 0 {
			if err := p.scaleOut(p.fleetId, p.scalingAdjustment); err != nil {
				logger.Errorf("Scaling error: %+v", err)
			}
		}
		if p.scalingAdjustment < 0 {
			if err := p.scaleIn(p.fleetId, p.scalingAdjustment); err != nil {
				logger.Errorf("Scaling error: %+v", err)
			}
		}
		return nil
	case "ExactCapacity":
	case "PercentChangeInCapacity":
	default:
		return errors.New("Unsupported scalingAdjustmentType!")
	}

	return nil
}

func (p *Policy) scaleOut(fleetId string, instanceNum int) error {
	logger.Infof("Try to scale out %d instances in scalinggroup: %s", instanceNum, fleetId)
	return p.groupInterface.ScaleOut(fleetId, instanceNum)
}

func (p *Policy) scaleIn(fleetId string, instanceNum int) error {
	panic("implement me")
}
