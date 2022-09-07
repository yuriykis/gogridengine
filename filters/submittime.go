package filters

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yuriykis/gogridengine"
)

//NewBeforeSubmitTimeFilter returns only jobs whose submitted time occurs before the provided time.
func NewBeforeSubmitTimeFilter(t time.Time) func(job gogridengine.Job) bool {
	return func(job gogridengine.Job) bool {
		jobTime, err := time.Parse(ISO8601FMT, job.SubmittedTime)
		if err != nil {
			//If we can't parse the value, discard the job
			log.Error("Failed parsing the time content: ", err)
			return false
		}

		return jobTime.Before(t)
	}
}

//NewAfterSubmitTimeFilter returns only jobs whose submitted time occurs after the provided time.
func NewAfterSubmitTimeFilter(t time.Time) func(job gogridengine.Job) bool {
	return func(job gogridengine.Job) bool {
		jobTime, err := time.Parse(ISO8601FMT, job.SubmittedTime)
		if err != nil {
			//If we can't parse the value, discard the job
			return false
		}

		return jobTime.After(t)
	}
}

//NewBetweenSubmitTimeFilter allows you to provide a start and end time to return jobs whos submit time falls within that range
func NewBetweenSubmitTimeFilter(start time.Time, end time.Time) func(job gogridengine.Job) bool {
	return func(job gogridengine.Job) bool {
		jobTime, err := time.Parse(ISO8601FMT, job.SubmittedTime)
		if err != nil {
			//If we can't parse the value, discard the job
			return false
		}

		return jobTime.After(start) && jobTime.Before(end)
	}
}
