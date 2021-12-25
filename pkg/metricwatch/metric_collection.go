package metricwatch

import (
	"nanto.io/application-auto-scaling-service/pkg/resourceadapter/gamelift"
	"time"
)

type MetricData struct {
	metricName string
	value      float32
	dimensions []Dimension
	unit       string
	timestamp  time.Time
}

type Dimension struct {
	name string
	value string
}

func PutMetricData() error {
	// TODO(wj): 添加到数据库
	return nil
}

func GetMetricData(startTime time.Time, endTime time.Time, metricName string, dimensions []Dimension) []MetricData {
	// TODO(wj): 当前Show仅需要获取gamelift
	if metricName != "PercentAvailableGameSessions" {
		return []MetricData{}
	}

	value, err := gamelift.NewAwsClient().GetPercentAvailableGameSessions("fleet-123")
	if err != nil {
		logger.Errorf("GetMetrcData error:%v", err)
		return []MetricData{}
	}

	return []MetricData {
		{
			metricName: "PercentAvailableGameSessions",
			value: value,
		},
	}
}