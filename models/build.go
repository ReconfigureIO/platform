package models

//go:generate mockgen -source=build.go -package=models -destination=build_mock.go

import (
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"
)

type BuildRepo interface {
	// Return a list of deployments, with the statuses specified,
	// limited to that number
	GetBuildsWithStatus([]string, int) ([]Build, error)
	StoreBuildReport(Build, ReportV1) error
	GetBuildReport(build Build) (BuildReport, error)
}

type buildRepo struct{ db *gorm.DB }

// DeploymentDataSource returns the data source for deployments.
func BuildDataSource(db *gorm.DB) BuildRepo {
	return &buildRepo{db: db}
}

const (
	SQL_BUILD_STATUS = `SELECT j.id
FROM builds j
LEFT join batch_job_events e
ON j.batch_job_id = e.batch_job_id
    AND e.timestamp = (
        SELECT max(timestamp)
        FROM batch_job_events e1
        WHERE j.batch_job_id = e1.batch_job_id
    )
WHERE (e.status in (?))
LIMIT ?
`
)

func (repo *buildRepo) GetBuildsWithStatus(statuses []string, limit int) ([]Build, error) {
	db := repo.db
	rows, err := db.Raw(SQL_BUILD_STATUS, statuses, limit).Rows()
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	var builds []Build
	err = db.Preload("BatchJob").Preload("BatchJob.Events").Where("id in (?)", ids).Find(&builds).Error
	if err != nil {
		return nil, err
	}

	return builds, nil
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

// The place to upload build input to
// should be a tar.gz
func (build Build) InputUrl() string {
	return fmt.Sprintf("builds/%s/build.tar.gz", build.ID)
}

// The place to build artifacts will be uploaded to
// Should be a zip file
func (build Build) ArtifactUrl() string {
	return fmt.Sprintf("builds/%s/artifacts.zip", build.ID)
}

// The place the debug artifacts including reports, logs, etc
// Should be a zip file
func (build Build) DebugUrl() string {
	return fmt.Sprintf("builds/%s/debug.zip", build.ID)
}

// The place build reports will be uploaded to
func (build Build) ReportUrl() string {
	return fmt.Sprintf("builds/%s/reports", build.ID)
}

// Status returns buikld status.
func (b *Build) Status() string {
	events := b.BatchJob.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return StatusSubmitted
}

// HasStarted returns if the build has started.
func (b *Build) HasStarted() bool {
	return hasStarted(b.Status())
}

// HasFinished returns if build is finished.
func (b *Build) HasFinished() bool {
	return hasFinished(b.Status())
}

type BuildReport struct {
	uuidHook
	ID      string `gorm:"primary_key" json:"-"`
	Build   Build  `json:"-" gorm:"ForeignKey:BuildID"`
	BuildID string `json:"-"`
	Version string `json:"-"`
	Report  string `json:"report" sql:"type:JSONB NOT NULL DEFAULT '{}'::JSONB"`
}

// StoreBuildReport takes a build and reportV1,
// and attaches the report to the build
func (repo *buildRepo) StoreBuildReport(build Build, report ReportV1) error {
	db := repo.db
	newBytes, err := json.Marshal(&report)
	if err != nil {
		return err
	}
	buildReport := BuildReport{
		BuildID: build.ID,
		Version: "v1",
		Report:  string(newBytes),
	}
	err = db.Create(&buildReport).Error
	return err
}

// GetBuildReport gets a build report given a build
func (repo *buildRepo) GetBuildReport(build Build) (BuildReport, error) {
	report := BuildReport{}
	db := repo.db

	err := db.Model(&build).Related(&report).Error
	return report, err
}

// PostBuild is post request body for a new build.
type PostBuild struct {
	ProjectID string `json:"project_id" validate:"nonzero"`
	Comment   string `json:"comment"`
}

type ReportV1 struct {
	ModuleName      string       `json:"moduleName"`
	PartName        string       `json:"partName"`
	LutSummary      GroupSummary `json:"lutSummary"`
	RegSummary      GroupSummary `json:"regSummary"`
	BlockRamSummary GroupSummary `json:"blockRamSummary"`
	UltraRamSummary PartDetail   `json:"ultraRamSummary"`
	DspBlockSummary PartDetail   `json:"dspBlockSummary"`
	WeightedAverage PartDetail   `json:"weightedAverage"`
}

type GroupSummary struct {
	Description string      `json:"description"`
	Used        int         `json:"used"`
	Available   int         `json:"available"`
	Utilisation float32     `json:"utilisation"`
	Detail      PartDetails `json:"detail"`
}

type PartDetails map[string]PartDetail

type PartDetail struct {
	Description string  `json:"description"`
	Used        int     `json:"used"`
	Available   int     `json:"available"`
	Utilisation float32 `json:"utilisation"`
}
