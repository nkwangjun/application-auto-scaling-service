package k8s

import (
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

var jobCron *cron.Cron

func InitCron() {
	jobCron = cron.New(cron.WithSeconds())
}

func GetCron() *cron.Cron {
	if jobCron == nil {
		klog.Fatal("Obj jobCron not initialized")
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
	fmt.Println("===jobCron.Entries", jobCron.Entries())
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
