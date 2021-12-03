package cronutil

import (
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

var logger = logutil.GetLogger()

var jobCron *cron.Cron

func InitCron() {
	jobCron = cron.New(cron.WithSeconds())
}

func GetCron() *cron.Cron {
	if jobCron == nil {
		logger.Panic("Obj jobCron not initialized")
	}
	return jobCron
}

func RemoveAllCronEntries() {
	for _, entry := range jobCron.Entries() {
		jobCron.Remove(entry.ID)
	}
}

func FindJobNeedExecNow() (cron.Job, error) {
	maxTime := time.Now()
	var jobExecNow cron.Job
	for _, entry := range jobCron.Entries() {
		if entry.Next.After(maxTime) {
			maxTime = entry.Next
			jobExecNow = entry.Job
		}
	}
	if jobExecNow == nil {
		return nil, errors.New("no job find")
	}
	return jobExecNow, nil
}
