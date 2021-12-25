package metricwatch

import (
	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
	"time"
)

var logger = logutil.GetLogger()

type invokeAction func() error

type MetricAlarm struct {
	fleetId            string
	metricName         string
	dimensions         []Dimension
	comparisonOperator string
	evaluationPeriods  int
	threshold          float32
	action             invokeAction
	satisfiedCount     int
}

func NewMetricAlarm(fleetId string, metricName string, comparisonOperator string, evaluationPeriods int,
	threshold float32, action invokeAction) *MetricAlarm {
	return &MetricAlarm{
		fleetId:            fleetId,
		metricName:         metricName,
		comparisonOperator: comparisonOperator,
		evaluationPeriods:  evaluationPeriods,
		threshold:          threshold,
		action:             action,
		satisfiedCount:     0,
	}
}

func (m *MetricAlarm) Run(quit <-chan struct{}) {
	for {
		select {
		case <-quit:
			logger.Infof("Quit policy evaluation, policy: %+v", m)
			return
		default:
			m.doEvaluation()
			m.doAction()
			time.Sleep(time.Minute)
		}
	}
}

func (m *MetricAlarm) doEvaluation() {
	logger.Debugf("Evaluation here, MetricAlarm: %+v", m)
	// 获取最近的一次指标
	dataList := GetMetricData(time.Now().Add(-time.Minute), time.Now(), m.metricName, []Dimension{
		{"fleetId", m.fleetId}})

	if dataList == nil || len(dataList) == 0 {
		logger.Warnf("Invalid metric evaluation, metricName:%s, fleetId:%s", m.metricName, m.fleetId)
		m.satisfiedCount = 0
		return
	}

	// 获取最近一个值
	value := dataList[0].value
	isSatisfied := false
	switch m.comparisonOperator {
	case "GreaterThanOrEqualToThreshold":
		if value >= m.threshold {
			isSatisfied = true
		}
	case "GreaterThanThreshold":
		if value > m.threshold {
			isSatisfied = true
		}
	case "LessThanThreshold":
		if value < m.threshold {
			isSatisfied = true
		}
	case "LessThanOrEqualToThreshold":
		if value <= m.threshold {
			isSatisfied = true
		}
	default:
	}

	if isSatisfied {
		logger.Infof("Metric evaluation satisfied, metric data:%v, metric alarm:%v", dataList[0], m)
		m.satisfiedCount++
		return
	}

	logger.Infof("Metric evaluation not satisfied, metric data:%v, metric alarm:%v", dataList[0], m)
	m.satisfiedCount = 0
}

func (m *MetricAlarm) doAction() {
	if m.satisfiedCount < m.evaluationPeriods {
		logger.Debugf("No need to do action, satisfiedCount:%d, evaluationPeriods:%d, fleetId: %s",
			m.satisfiedCount, m.evaluationPeriods, m.fleetId)
		return
	}

	// execute action
	if err := m.action(); err != nil {
		// retry next runPeriods
		logger.Errorf("Do action error: %+v", err)
		return
	}

	logger.Infof("Do action successful, fleetId: %s", m.fleetId)

	// update satisfiedCount
	m.satisfiedCount = 0
}
