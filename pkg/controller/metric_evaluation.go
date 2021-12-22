package controller

import "time"

type MetricEvaluation struct {
	fleetId            string
	metricName         string
	comparisonOperator string
	evaluationPeriods  int
	threshold          float32
	action             invokeAction
	satisfiedCount     int
}

func NewMetricEvaluation(fleetId string, metricName string, comparisonOperator string, evaluationPeriods int,
	threshold float32, action invokeAction) *MetricEvaluation {
	return &MetricEvaluation{
		fleetId:            fleetId,
		metricName:         metricName,
		comparisonOperator: comparisonOperator,
		evaluationPeriods:  evaluationPeriods,
		threshold:          threshold,
		action:             action,
		satisfiedCount:     0,
	}
}

func (m *MetricEvaluation) Run(quit <-chan struct{}) {
	for {
		select {
		case <- quit:
			logger.Infof("Quit policy evaluation, policy: %+v", m)
			return
		default:
			m.doEvaluation()
			m.doAction()
			time.Sleep(time.Minute)
		}
	}
}

func (m *MetricEvaluation) doEvaluation() {
	logger.Infof("Evaluation here, fleetId: %s", m.fleetId)


	m.satisfiedCount++
}

func (m *MetricEvaluation) doAction() {
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
