package metricwatch

import "time"

type MetricData struct {
	metricName string
	value      int32
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
	// TODO(wj): 查询数据库
	return nil
}