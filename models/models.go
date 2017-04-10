package models

import (
	"github.com/jinzhu/gorm"
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
	ID         int         `gorm:"primary_key" json:"id"`
	GithubID   string      `json:"github_id"`
	Email      string      `gorm:"type:varchar(100);unique_index" json:"email"`
	AuthTokens []AuthToken `json:"auth_token"` //User has many AuthTokens
}

type Project struct {
	ID          int     `gorm:"primary_key" json:"id"`
	User        User    `json:"-" gorm:"ForeignKey:UserID"` //Project belongs to User
	UserID      int     `json:"-"`
	Name        string  `json:"name"`
	Builds      []Build `json:"builds" gorm:"ForeignKey:ProjectID"`
	Simulations []Build `json:"simulations" gorm:"ForeignKey:ProjectID"`
}

type PostProject struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
}

type AuthToken struct {
	gorm.Model
	Token  string `json:"token"`
	UserID int    `json:"user_id"`
}

type Build struct {
	ID        int     `gorm:"primary_key" json:"id"`
	Project   Project `json:"project" gorm:"ForeignKey:ProjectID"`
	ProjectID int     `json:"project_id"`
	BatchId   string  `json:"-"`
}

func (b *Build) HasStarted() bool {
	return false
	//	return hasStarted(b.Status)
}

func (b *Build) HasFinished() bool {
	return false
	//	return hasFinished(b.Status)
}

type PostBuild struct {
	UserID         int    `json:"user_id" validate:"nonzero"`
	ProjectID      int    `json:"project_id" validate:"nonzero"`
	InputArtifact  string `json:"input_artifact"`
	OutputArtifact string `json:"output_artifact"`
	OutputStream   string `json:"output_stream"`
	Status         string `gorm:"default:'SUBMITTED'" json:"status"`
}

type Simulation struct {
	ID        int               `gorm:"primary_key" json:"id"`
	Token     string            `json:"-"` // Internal Authentication token for service updates
	User      User              `json:"-" gorm:"ForeignKey:UserID"`
	UserID    int               `json:"-"`
	Project   *Project          `json:"project,omitempty" gorm:"ForeignKey:ProjectID"`
	ProjectID int               `json:"-"`
	Command   string            `json:"command"`
	BatchId   string            `json:"-"`
	Events    []SimulationEvent `json:"events" gorm:"ForeignKey:SimulationID"`
}

func (s *Simulation) Status() string {
	events := s.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return SUBMITTED
}

type SimulationEvent struct {
	ID           int       `gorm:"primary_key" json:"-"`
	SimulationID int       `json:"-"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"`
	Message      string    `json:"message,omitempty"`
	Code         int       `json:"code"`
}

type PostSimulationEvent struct {
	Status  string `json:"status" validate:"nonzero"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (s *Simulation) HasStarted() bool {
	return false
	//	return hasStarted(s.Status)
}

func (s *Simulation) HasFinished() bool {
	return false
	//return hasFinished(s.Status)
}

type PostSimulation struct {
	ProjectID int    `json:"project_id" validate:"nonzero"`
	Command   string `json:"command" validate:"nonzero"`
}

var statuses = struct {
	started  []string
	finished []string
}{
	started:  []string{STARTED, COMPLETED, ERRORED},
	finished: []string{COMPLETED, ERRORED, TERMINATED},
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
