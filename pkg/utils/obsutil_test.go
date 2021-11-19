package utils

import (
	"fmt"
	"testing"
)

func TestSendFileToOBS(t *testing.T) {
	SendNodeIdsFileToOBS("test_clusterId", "D:\\develop\\GolandProjects\\application-auto-scaling-service\\.gitignore")
}

func TestGetStrategiesFileFromOBS(t *testing.T) {
	bytes, err := GetStrategiesFromTianCe("test_clusterId")
	if err != nil {
		t.Errorf("GetStrategiesFromTianCe err: %+v", err)
		return
	}
	fmt.Printf("Read strategies json:\n%s\n", bytes)
}
