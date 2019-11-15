package gogridengine

import (
	"encoding/xml"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	//TASKRANGEIDENTIFIERREGEX is a regex string used for identifying whether <tasks> objects indicate a range of tasks (normally only expressed on pending tasks)
	TASKRANGEIDENTIFIERREGEX string = `[0-9]{1,}-[0-9]{1,}:[0-9]`
)

//Task is an element used for handling task arrays from the grid engine. Here we'll store the raw value (Source) and the TaskID if an individual identifier.
type Task struct {
	//Mixed type. Can be either a string representation of an int64 OR a string range identifier, eg: 40-55:1 (Jobs 40-55 incremented by 1)
	Source string
	//Typed representation of the Source if mapped to an integer
	TaskID int64
}

//UnmarshalXML is a custom marshaller for handling complex logic surrounding task data.
func (t *Task) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {

	var v string
	d.DecodeElement(&v, &start)

	t.Source = v

	if !strings.Contains(t.Source, ":") {
		//Only process TaskIDs when not presented with a ":"
		parsed, err := strconv.ParseInt(t.Source, 10, 64)

		if err != nil {
			log.Error("Attempting to parse Task identifier failed: ", err)
			return err
		}

		t.TaskID = parsed
	}

	return nil
}

//MarshalXML renders the value back down to the XML structure
func (t *Task) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if strings.Contains(t.Source, ":") {
		e.EncodeElement(string(t.Source), start)
	} else {
		if t.TaskID == 0 {
			e.EncodeElement(nil, start)
			return nil
		}
		e.EncodeElement(strconv.Itoa(int(t.TaskID)), start)
	}

	return nil
}

//TODO: Marshaller implementation for moving back to XML -- Just represent the JOB ID. Won't work on de-duplicating back to string only

//JobList is a slice of Jobs that is filterable and otherwise actionable via receiver.
type JobList []Job

//Job is the Sun Grid Engine XML Definition for a job running on a specific host, its details and current status
type Job struct {
	//Because this is a node, we still need the XMLName identifier
	XMLName        xml.Name `xml:"job_list" json:"-"`
	StateAttribute string   `xml:"state,attr" json:"state_attribute_text"`
	State          string   `xml:"state" json:"state"`
	JBJobNumber    int64    `xml:"JB_job_number" json:"jb_job_number"`
	JATPriority    float64  `xml:"JAT_prio" json:"jat_prio"`
	JobName        string   `xml:"JB_name" json:"jb_name"`
	JobOwner       string   `xml:"JB_owner" json:"jb_owner"`
	StartTime      string   `xml:"JAT_start_time,omitempty" json:"start_time"`
	SubmittedTime  string   `xml:"JB_submission_time,omitempty" json:"submitted_time"`
	Slots          int32    `xml:"slots" json:"slots"`
	Tasks          Task     `xml:"tasks,omitempty" json:"tasks,omitempty"`
}

//IsJobRunning returns a int (1 - running) (0 - not)
func IsJobRunning(job Job) int {

	if job.State == "r" {
		return 1
	}

	return 0
}

//GetJobs returns a slice of only jobs from both scheduled and unscheduled queues
func GetJobs() (JobList, error) {
	var jobs []Job

	ji, err := GetJobInfo()

	if err != nil {
		return []Job{}, err
	}

	//Add running jobs to the slice first
	for _, q := range ji.QueueInfo.Queues {
		jobs = append(jobs, q.JobList...)
	}

	//Add pending jobs
	jobs = append(jobs, ji.PendingJobs.JobList...)

	return jobs, nil
}

//GetJobsWithFilter allows you to specify a filter at the time of retrieving the JobList
func GetJobsWithFilter(filterfunc func(j Job) bool) (JobList, error) {
	jobs, err := GetJobs()
	if err != nil {
		return JobList{}, err
	}

	return jobs.Filter(filterfunc), nil
}

//FilterJobs is a function allowing you to manually provide a JobList and a filter function to limit the content down.
func FilterJobs(jobs JobList, filter func(j Job) bool) JobList {
	var jl JobList

	for _, v := range jobs {
		if filter(v) {
			jl = append(jl, v)
		}
	}

	return jl
}

//Filter allows for the passage of any function taking a JobList and Filtering its contents down.
//Should be usable in fluent fashion as long as JobList is being returned
func (jl JobList) Filter(filter func(j Job) bool) JobList {
	var jobs JobList

	for _, v := range jl {
		if filter(v) {
			jobs = append(jobs, v)
		}
	}

	return jobs
}

//Sort allows you to provide your own Less function to handle sorting the list directly
func (jl JobList) Sort(sorter func(i, j int) bool) JobList {
	sort.Slice(jl[:], sorter)

	return jl
}

//DoesJobContainTaskRange evaluates whether the XML marshalled tasks value contains the regex indicating a sequence of tasks.
func DoesJobContainTaskRange(j Job) (bool, error) {
	regex, err := regexp.Compile(TASKRANGEIDENTIFIERREGEX)

	if err != nil {
		return false, err
	}

	return regex.MatchString(j.Tasks.Source), nil
}

//ExtrapolateTasksToJobs takes the role of finding the range identifier and returning a job list from it (Extrapolated from the task list)
func ExtrapolateTasksToJobs(original Job) (JobList, error) {
	var jl JobList

	ok, err := DoesJobContainTaskRange(original)

	if err != nil {
		return JobList{}, err
	}

	if !ok {
		return JobList{}, errors.New("The provided job does not actually indicate a range of tasks")
	}

	r, err := regexp.Compile(TASKRANGEIDENTIFIERREGEX)

	if err != nil {
		return JobList{}, err
	}

	identifier := r.FindString(original.Tasks.Source)

	pieces := strings.Split(identifier, ":")
	rangeComponent := pieces[0]
	incrementor := pieces[1]

	rangePieces := strings.Split(rangeComponent, "-")
	begin := rangePieces[0]
	end := rangePieces[1]

	//Get a mathematically usable value
	intrementor, err := strconv.ParseInt(incrementor, 10, 64)

	if err != nil {
		return JobList{}, err
	}

	beginInt, err := strconv.ParseInt(begin, 10, 64)

	if err != nil {
		return JobList{}, err
	}

	endInt, err := strconv.ParseInt(end, 10, 64)

	if err != nil {
		return JobList{}, err
	}

	for i := beginInt; i <= endInt; i = i + intrementor {
		lj := original

		lj.Tasks.TaskID = i

		jl = append(jl, lj)
	}

	return jl, nil
}
