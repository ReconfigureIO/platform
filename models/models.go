package models

import (
	"fmt"
	"sort"
	"time"

	"github.com/dchest/uniuri"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

const (
	// StatusSubmitted is submitted job state.
	StatusSubmitted = "SUBMITTED"
	// StatusQueued is queued job state.
	StatusQueued = "QUEUED"
	// StatusCreatingImage is creating image job state.
	StatusCreatingImage = "CREATING_IMAGE"
	// StatusStarted is started job state.
	StatusStarted = "STARTED"
	// StatusTerminating is terminating job state.
	StatusTerminating = "TERMINATING"
	// StatusTerminated is terminated job state.
	StatusTerminated = "TERMINATED"
	// StatusCompleted is completed job state.
	StatusCompleted = "COMPLETED"
	// StatusErrored is errored job state.
	StatusErrored = "ERRORED"

	// PlanOpenSource is open source plan.
	PlanOpenSource = "open-source"
	// PlanSingleUser is single user plan.
	PlanSingleUser = "single-user"

	// DefaultHours is the amount of hours a new user gets.
	DefaultHours = 20
)

// uuidHook hooks new uuid as primary key for models before creation.
type uuidHook struct{}

func (u uuidHook) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("id", uuid.NewV4().String())
}

// User model.
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

// LoginToken return the user's login token.
func (u User) LoginToken() string {
	return fmt.Sprintf("gh_%d_%s", u.GithubID, u.Token)
}

// NewUser creates a new User.
func NewUser() User {
	return User{Token: uniuri.NewLen(64), BillingPlan: PlanOpenSource}
}

// CreateOrUpdateUser creates the given u in the database if it doesn't exist, and otherwise returns the existing one.
func CreateOrUpdateUser(db *gorm.DB, u User, createNew bool) (User, error) {
	q := db.Where(User{GithubID: u.GithubID})

	var user User
	if err := q.First(&user).Error; err != nil {
		// not found
		user = NewUser()
		if err != gorm.ErrRecordNotFound {
			return user, err
		}
		if !createNew {
			return user, err
		}
	}

	err := q.Attrs(user).FirstOrInit(&u).Error
	if err != nil {
		return u, err
	}
	db.Save(&u)
	return u, err
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

// PostBatchEvent is post request body for batch events.
type PostBatchEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status" validate:"nonzero"`
	Message   string    `json:"message"`
	Code      int       `json:"code"`
}

// PostDepEvent is post request body for deployment events.
type PostDepEvent struct {
	Status  string `json:"status" validate:"nonzero"`
	Message string `json:"message"`
	Code    int    `json:"code"`
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
	IPAddress    string            `json:"ip_address"`
	UserID       string            `gorm:"not_null"`
	SpotInstance bool              `json:"-" sql:"NOT NULL;DEFAULT:false"`
	Events       []DeploymentEvent `json:"events" gorm:"ForeignKey:DeploymentID"`
}

// PostDeployment is post request body for new deployment.
type PostDeployment struct {
	BuildID string `json:"build_id" validate:"nonzero"`
	Command string `json:"command" validate:"nonzero"`
}

// Status returns deployment status.
func (d *Deployment) Status() string {
	events := d.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return StatusSubmitted
}

// StartTime returns the time that the deployment
// begins.
func (d Deployment) StartTime() (t time.Time) {
	for _, e := range d.Events {
		if e.Status == StatusStarted {
			return e.Timestamp
		}
	}
	return
}

var statuses = struct {
	started  []string
	finished []string
}{
	started:  []string{StatusStarted, StatusCompleted, StatusErrored, StatusTerminating, StatusCreatingImage, StatusTerminated},
	finished: []string{StatusCompleted, StatusErrored, StatusTerminated},
}

// BatchJob model.
type BatchJob struct {
	ID      int64           `gorm:"primary_key" json:"-"`
	BatchID string          `json:"-"`
	LogName string          `json:"-"`
	Events  []BatchJobEvent `json:"events" gorm:"ForeignKey:BatchJobId"`
}

// Status returns the status of the job.
func (b *BatchJob) Status() string {
	events := b.Events
	length := len(events)
	switch length {
	case 0:
		return StatusSubmitted
	case 1:
		return events[length-1].Status
	default:
		sort.Slice(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })
		return events[length-1].Status
	}
}

// HasStarted returns if batch job has started.
func (b *BatchJob) HasStarted() bool {
	return hasStarted(b.Status())
}

// HasFinished returns if batch job is finished.
func (b *BatchJob) HasFinished() bool {
	return hasFinished(b.Status())
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

// HasStarted returns if the job has started.
func (d *Deployment) HasStarted() bool {
	return hasStarted(d.Status())
}

// HasFinished returns if the job has finished.
func (d *Deployment) HasFinished() bool {
	return hasFinished(d.Status())
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

func hasStarted(status string) bool {
	return inSlice(statuses.started, status)
}

func hasFinished(status string) bool {
	return inSlice(statuses.finished, status)
}

// CanTransition returns if the status can move to the next stage.
func CanTransition(current string, next string) bool {
	switch current {
	case StatusSubmitted:
		return inSlice([]string{StatusQueued, StatusTerminated, StatusTerminating}, next)
	case StatusQueued:
		return inSlice([]string{StatusStarted, StatusTerminated, StatusTerminating}, next)
	case StatusStarted:
		return inSlice([]string{StatusTerminated, StatusCreatingImage, StatusCompleted, StatusErrored, StatusTerminating}, next)
	case StatusCreatingImage:
		return inSlice([]string{StatusTerminated, StatusCompleted, StatusErrored, StatusTerminating}, next)
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
