package migration201807191024

import (
	"time"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// uuidHook hooks new uuid as primary key for models before creation.
type uuidHook struct{}

func (u uuidHook) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("id", uuid.NewV4().String())
}

// InviteToken model.
type InviteToken struct {
	Token      string    `gorm:"type:varchar(128);primary_key" json:"token"`
	IntercomId string    `gorm:"type:varchar(128)" json:"-"`
	Timestamp  time.Time `json:"created_at"`
}

type User struct {
	uuidHook
	ID                string    `gorm:"primary_key" json:"id"`
	GithubID          int       `gorm:"unique_index" json:"-"`
	GithubName        string    `json:"github_name"`
	Name              string    `json:"name"`
	Email             string    `gorm:"type:varchar(100);unique_index" json:"email"`
	CreatedAt         time.Time `json:"created_at"`
	PhoneNumber       string    `json:"phone_number"`
	Company           string    `json:"company"`
	Landing           string    `json:"-"`
	MainGoal          string    `json:"-"`
	Employees         string    `json:"-"`
	MarketVerticals   string    `json:"-"`
	JobTitle          string    `json:"-"`
	GithubAccessToken string    `json:"-"`
	Token             string    `json:"-"`
	StripeToken       string    `json:"-"`
	// We'll ignore this in the db for now, to provide mock data
	BillingPlan string `gorm:"-" json:"billing_plan"`
}

// Project model.
type Project struct {
	uuidHook
	ID          string  `gorm:"primary_key" json:"id"`
	User        User    `json:"-" gorm:"ForeignKey:UserID"` // Project belongs to User
	UserID      string  `json:"-"`
	Name        string  `json:"name"`
	Builds      []Build `json:"builds,omitempty" gorm:"ForeignKey:ProjectID"`
	Simulations []Build `json:"simulations,omitempty" gorm:"ForeignKey:ProjectID"`
}

// Simulation model.
type Simulation struct {
	uuidHook
	ID         string   `gorm:"primary_key" json:"id"`
	User       User     `json:"-" gorm:"ForeignKey:UserID"`
	UserID     int      `json:"-"`
	Project    Project  `json:"project,omitempty" gorm:"ForeignKey:ProjectID"`
	ProjectID  string   `json:"-"`
	BatchJobID int64    `json:"-"`
	BatchJob   BatchJob `json:"job" gorm:"ForeignKey:BatchJobId"`
	Token      string   `json:"-"`
	Command    string   `json:"command"`
}

// Deployment model.
type Deployment struct {
	uuidHook
	ID           string            `gorm:"primary_key" json:"id"`
	Build        Build             `json:"build" gorm:"ForeignKey:BuildID"`
	BuildID      string            `json:"-"`
	Command      string            `json:"command"`
	Token        string            `json:"-"`
	InstanceID   string            `json:"-"`
	UserID       string            `json:"-"`
	IPAddress    string            `json:"ip_address"`
	SpotInstance bool              `json:"-" sql:"NOT NULL;DEFAULT:false"`
	Events       []DeploymentEvent `json:"events" gorm:"ForeignKey:DeploymentID"`
}

// BatchJob model.
type BatchJob struct {
	ID      int64           `gorm:"primary_key" json:"-"`
	BatchID string          `json:"-"`
	LogName string          `json:"-"`
	Events  []BatchJobEvent `json:"events" gorm:"ForeignKey:BatchJobId"`
}

// BatchJobEvent model.
type BatchJobEvent struct {
	uuidHook
	ID         string    `gorm:"primary_key" json:"-"`
	BatchJobID int64     `json:"-"`
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	Code       int       `json:"code"`
}

// DeploymentEvent model.
type DeploymentEvent struct {
	uuidHook
	ID           string    `gorm:"primary_key" json:"-"`
	DeploymentID string    `json:"-" validate:"nonzero"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"`
	Message      string    `json:"message,omitempty"`
	Code         int       `json:"code"`
}

// Build model.
type Build struct {
	uuidHook
	ID          string       `gorm:"primary_key" json:"id"`
	Project     Project      `json:"project" gorm:"ForeignKey:ProjectID"`
	ProjectID   string       `json:"-"`
	BatchJob    BatchJob     `json:"job" gorm:"ForeignKey:BatchJobId"`
	BatchJobID  int64        `json:"-"`
	FPGAImage   string       `json:"-"`
	Token       string       `json:"-"`
	Comment     string       `json:"comment"`
	Deployments []Deployment `json:"deployments,omitempty" gorm:"ForeignKey:BuildID"`
}

type BuildReport struct {
	uuidHook
	ID      string `gorm:"primary_key" json:"-"`
	Build   Build  `json:"-" gorm:"ForeignKey:BuildID"`
	BuildID string `json:"-"`
	Version string `json:"-"`
	Report  string `json:"report" sql:"type:JSONB NOT NULL DEFAULT '{}'::JSONB"`
}

// Graph model.
type Graph struct {
	uuidHook
	ID         string   `gorm:"primary_key" json:"id"`
	Project    Project  `json:"project" gorm:"ForeignKey:ProjectID"`
	ProjectID  string   `json:"-"`
	BatchJob   BatchJob `json:"job" gorm:"ForeignKey:BatchJobId"`
	BatchJobID int64    `json:"-"`
	Token      string   `json:"-"`
	Type       string   `json:"type" gorm:"default:'dataflow'"`
}

// QueueEntry is a queue entry.
type QueueEntry struct {
	uuidHook
	ID           string `gorm:"primary_key"`
	Type         string `gorm:"default:'deployment'"`
	TypeID       string `gorm:"not_null"`
	User         User   `json:"-" gorm:"ForeignKey:UserID"`
	UserID       string `json:"-"`
	Weight       int
	Status       string
	CreatedAt    time.Time
	DispatchedAt time.Time
}
