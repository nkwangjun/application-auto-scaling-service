package utils

import "testing"

func TestSendFileToOBS(t *testing.T) {
	SendFileToOBS("test_clusterId", "D:\\develop\\GolandProjects\\application-auto-scaling-service\\.gitignore")
}
