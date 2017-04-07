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

type PostBuild struct {
	UserID         int    `json:"user_id"`
	ProjectID      int    `json:"project_id"`
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
	Status        string  `gorm:"default:'SUBMITTED'" json:"status"`
}

type PostSimulation struct {
	UserID        int    `json:"user_id"`
	ProjectID     int    `json:"project_id"`
	InputArtifact string `json:"input_artifact"`
	Command       string `json:"command"`
	OutputStream  string `json:"output_stream"`
	Status        string `gorm:"default:'SUBMITTED'" json:"status"`
}
