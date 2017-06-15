package models

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

// DeploymentRepo handles deployment details.
type DeploymentRepo interface {
	// Return a list of deployments, with the statuses specified,
	// limited to that number
	GetWithStatus([]string, int) ([]Deployment, error)
	// DeploymentHoursSince returns the total time used for deployments since
	// startTime.
	DeploymentHoursSince(userID string, startTime time.Time) (time.Duration, error)
}

type deploymentRepo struct{ db *gorm.DB }

// DeploymentDataSource returns the data source for deployments.
func DeploymentDataSource(db *gorm.DB) DeploymentRepo {
	return &deploymentRepo{db: db}
}

const (
	sqlDeploymentStatus = `SELECT j.id
FROM deployments j
LEFT join deployment_events e
ON j.id = e.dep_id
    AND e.timestamp = (
        SELECT max(timestamp)
        FROM deployment_events e1
        WHERE j.id = e1.dep_id
    )
WHERE (e.status in (?))
LIMIT ?
`
)

func (repo *deploymentRepo) GetWithStatus(statuses []string, limit int) ([]Deployment, error) {
	db := repo.db
	rows, err := db.Raw(sqlDeploymentStatus, statuses, limit).Rows()
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

	var deps []Deployment
	err = db.Preload("DepJob").Where("id in (?)", ids).Find(&deps).Error
	if err != nil {
		return nil, err
	}

	return deps, nil
}

func (repo *deploymentRepo) DeploymentHoursSince(userID string, startTime time.Time) (t time.Duration, err error) {
	db := repo.db
	var deployments []Deployment
	err = db.Model(&Deployment{}).
		Joins("left join builds on builds.id = deployments.build_id").
		Joins("left join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", userID).
		Find(&deployments).Error
	if err != nil {
		return
	}
	for _, deployment := range deployments {
		// TODO this queries the db for each deployment.
		// there should be a better way of lazy loading
		// and filtering outside of database.
		err := db.Model(&DeploymentEvent{}).
			Where("dep_id=?", deployment.ID).
			Where("timestamp>=?", timeToSQLStr(startTime)).
			Order("timestamp").
			Find(&deployment.Events).Error
		if err != nil {
			// TODO decide if this should stop this calculation
			// or it should be ignored and calculation should continue
			// as it currently is.
			fmt.Println(err)
			continue
		}
		if deployment.HasFinished() {
			stopTime := deployment.Events[len(deployment.Events)-1].Timestamp
			duration := stopTime.Sub(deployment.StartTime())
			t += duration
		} else if deployment.HasStarted() {
			duration := time.Now().Sub(deployment.StartTime())
			t += duration
		}
	}
	return
}

// monthStart changes t to the beginning of the month in UTC.
func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// timeToSQLStr formats t in sql format YYYY-MM-DD HH:MM:SS.
func timeToSQLStr(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
