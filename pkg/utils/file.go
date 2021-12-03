package utils

import (
	"os"

	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

var logger = logutil.GetLogger()

func CheckFileIsExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		logger.Warningf("CheckFileIsExist os.Stat err: %v", err)
		return !os.IsNotExist(err)
	}
	return true
}
