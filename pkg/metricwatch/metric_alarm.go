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
	// 获取最近

	m.satisfiedCount++
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
