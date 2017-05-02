package models

import (
	"github.com/dchest/uniuri"
	"time"
)

const (
	SUBMITTED  = "SUBMITTED"
	QUEUED     = "QUEUED"
	STARTED    = "STARTED"
	TERMINATED = "TERMINATED"
	COMPLETED  = "COMPLETED"
	ERRORED    = "ERRORED"
)

type User struct {
	ID                int    `gorm:"primary_key" json:"id"`
	GithubID          int    `gorm:"unique_index" json:"-"`
	GithubName        string `json:"github_name"`
	Name              string `json:"name"`
	Email             string `gorm:"type:varchar(100);unique_index" json:"email"`
	GithubAccessToken string `json:"-"`
	Token             string `json:"-"`
}

func NewUser() User {
	return User{Token: uniuri.NewLen(64)}
}

type Project struct {
	ID          int     `gorm:"primary_key" json:"id"`
	User        User    `json:"-" gorm:"ForeignKey:UserID"` //Project belongs to User
	UserID      int     `json:"-"`
	Name        string  `json:"name"`
	Builds      []Build `json:"builds,omitempty" gorm:"ForeignKey:ProjectID"`
	Simulations []Build `json:"simulations,omitempty" gorm:"ForeignKey:ProjectID"`
}

type Build struct {
	ID          int          `gorm:"primary_key" json:"id"`
	Project     Project      `json:"project" gorm:"ForeignKey:ProjectID"`
	ProjectID   int          `json:"-"`
	BatchJob    BatchJob     `json:"job" gorm:"ForeignKey:BatchJobId"`
	BatchJobId  int64        `json:"-"`
	Token       string       `json:"-"`
	Deployments []Deployment `json:"deployments,omitempty" gorm:"ForeignKey:BuildID"`
}

type PostBatchEvent struct {
	Status  string `json:"status" validate:"nonzero"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type PostDepEvent struct {
	Status  string `json:"status" validate:"nonzero"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (b *Build) Status() string {
	events := b.BatchJob.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return SUBMITTED
}

func (b *Build) HasStarted() bool {
	return hasStarted(b.Status())
}

func (b *Build) HasFinished() bool {
	return hasFinished(b.Status())
}

type PostBuild struct {
	ProjectID int `json:"project_id" validate:"nonzero"`
}

type Simulation struct {
	ID         int      `gorm:"primary_key" json:"id"`
	User       User     `json:"-" gorm:"ForeignKey:UserID"`
	UserID     int      `json:"-"`
	Project    Project  `json:"project,omitempty" gorm:"ForeignKey:ProjectID"`
	ProjectID  int      `json:"-"`
	BatchJobId int64    `json:"-"`
	BatchJob   BatchJob `json:"job" gorm:"ForeignKey:BatchJobId"`
	Token      string   `json:"-"`
	Command    string   `json:"command"`
}

func (s *Simulation) Status() string {
	events := s.BatchJob.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return SUBMITTED
}

type PostSimulation struct {
	ProjectID int    `json:"project_id" validate:"nonzero"`
	Command   string `json:"command" validate:"nonzero"`
}

type Deployment struct {
	ID       int    `gorm:"primary_key" json:"id"`
	Build    Build  `json:"build" gorm:"ForeignKey:BuildID"`
	BuildID  int    `json:"-"`
	Command  string `json:"command"`
	Token    string `json:"-"`
	DepJobId int    `json:"-"`
	DepJob   DepJob `json:"job,omitempty" gorm:"ForeignKey:DepJobId"`
}

type PostDeployment struct {
	BuildID int    `json:"build_id" validate:"nonzero"`
	Command string `json:"command" validate:"nonzero"`
}

func (d *Deployment) Status() string {
	events := d.DepJob.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return SUBMITTED
}

var statuses = struct {
	started  []string
	finished []string
}{
	started:  []string{STARTED, COMPLETED, ERRORED},
	finished: []string{COMPLETED, ERRORED, TERMINATED},
}

type BatchJob struct {
	ID      int64           `gorm:"primary_key" json:"-"`
	BatchId string          `json:"-"`
	Events  []BatchJobEvent `json:"events" gorm:"ForeignKey:BatchJobId"`
}

type DepJob struct {
	ID     int64         `gorm:"primary_key" json:"-"`
	DepId  string        `json:"-" validate:"nonzero"`
	Events []DepJobEvent `json:"events" gorm:"ForeignKey:BatchJobId"`
}

func (b *BatchJob) Status() string {
	events := b.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return SUBMITTED
}

func (b *BatchJob) HasStarted() bool {
	return hasStarted(b.Status())
}

func (b *BatchJob) HasFinished() bool {
	return hasFinished(b.Status())
}

func (d *DepJob) Status() string {
	events := d.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return SUBMITTED
}

type BatchJobEvent struct {
	ID         int64     `gorm:"primary_key" json:"-"`
	BatchJobId int64     `json:"-"`
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	Code       int       `json:"code"`
}

func (d *DepJob) HasStarted() bool {
	return hasStarted(d.Status())
}

func (d *DepJob) HasFinished() bool {
	return hasFinished(d.Status())
}

type DepJobEvent struct {

	ID        int64     `gorm:"primary_key" json:"-"`
	DepJobId  int64     `json:"-" validate:"nonzero"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Code      int       `json:"code"`
}

func hasStarted(status string) bool {
	return inSlice(statuses.started, status)
}

func hasFinished(status string) bool {
	return inSlice(statuses.finished, status)
}

func CanTransition(current string, next string) bool {
	switch current {
	case SUBMITTED:
		return inSlice([]string{QUEUED, TERMINATED}, next)
	case QUEUED:
		return inSlice([]string{STARTED, TERMINATED}, next)
	case STARTED:
		return inSlice([]string{TERMINATED, COMPLETED, ERRORED}, next)
	default:
		return false
	}
}

func inSlice(slice []string, val string) bool {
	for _, v := range slice {
		if val == v {
			return true
		}
	}
	return false
}
