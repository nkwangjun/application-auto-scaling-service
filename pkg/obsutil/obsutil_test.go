package obsutil

import (
	"fmt"
	"testing"
)

func TestGetStrategiesFileFromOBS(t *testing.T) {
	bytes, err := GetStrategiesFromTianCe("test_clusterId")
	if err != nil {
		t.Errorf("GetStrategiesFromTianCe err: %+v", err)
		return
	}
	fmt.Printf("Read strategies json:\n%s\n", bytes)
}
