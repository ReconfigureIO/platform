package models

import (
	"github.com/jinzhu/gorm"
)

type User struct {
	ID         int         `gorm:"primary_key" json:"id"`
	GithubID   string      `json:"github_id"`
	Email      string      `gorm:"type:varchar(100);unique_index" json:"email"`
	AuthTokens []AuthToken `json:"auth_token"` //User has many AuthTokens
}

type Project struct {
	ID     int     `gorm:"primary_key" json:"id"`
	User   User    `json:"user"` //Project belongs to User
	UserID int     `json:"user_id"`
	Name   string  `json:"name"`
	Builds []Build `json:"builds"`
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
	ID             int     `gorm:"primary_key" json:"id"`
	User           User    `json:"user"` //Build belongs to User, UserID is foreign key
	UserID         int     `json:"user_id"`
	Project        Project `json:"project"`
	ProjectID      int     `json:"project_id"`
	InputArtifact  string  `json:"input_artifact"`
	OutputArtifact string  `json:"output_artifact"`
	OutputStream   string  `json:"output_stream"`
	BatchId        string  `json:"-"`
	Status         string  `gorm:"default:'SUBMITTED'" json:"status"`
}

func (b *Build) HasStarted() bool {
	return hasStarted(b.Status)
}

func (b *Build) HasFinished() bool {
	return hasFinished(b.Status)
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
	ID            int     `gorm:"primary_key" json:"id"`
	User          User    `json:"user"` //Build belongs to User, UserID is foreign key
	UserID        int     `json:"user_id"`
	Project       Project `json:"project"`
	ProjectID     int     `json:"project_id"`
	InputArtifact string  `json:"input_artifact"`
	Command       string  `json:"command"`
	OutputStream  string  `json:"output_stream"`
	BatchId       string  `json:"-"`
	Status        string  `gorm:"default:'SUBMITTED'" json:"status"`
}

func (s *Simulation) HasStarted() bool {
	return hasStarted(s.Status)
}

func (s *Simulation) HasFinished() bool {
	return hasFinished(s.Status)
}

type PostSimulation struct {
	UserID        int    `json:"user_id" validate:"nonzero"`
	ProjectID     int    `json:"project_id" validate:"nonzero"`
	InputArtifact string `json:"input_artifact"`
	Command       string `json:"command" validate:"nonzero"`
	OutputStream  string `json:"output_stream"`
	Status        string `gorm:"default:'SUBMITTED'" json:"status"`
}

var statuses = struct {
	started  []string
	finished []string
}{
	started:  []string{"STARTED", "COMPLETED", "ERRORED"},
	finished: []string{"COMPLETED", "ERRORED"},
}

func hasStarted(status string) bool {
	return inSlice(statuses.started, status)
}

func hasFinished(status string) bool {
	return inSlice(statuses.finished, status)
}

func inSlice(slice []string, val string) bool {
	for _, v := range slice {
		if val == v {
			return true
		}
	}
	return false
}
