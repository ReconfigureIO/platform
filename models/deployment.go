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
	// DeploymentHoursBtw returns the total time used for deployments between
	// startTime and endTime.
	DeploymentHoursBetween(userID string, startTime, endTime time.Time) (time.Duration, error)
	// HoursUsedSince returns the total time used for deployments between startTime and now
	HoursUsedSince(userID string, startTime time.Time) (int, error)
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
ON j.id = e.deployment_id
	AND e.timestamp = (
		SELECT max(timestamp)
		FROM deployment_events e1
		WHERE j.id = e1.deployment_id
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
	err = db.Preload("Events").Where("id in (?)", ids).Find(&deps).Error
	if err != nil {
		return nil, err
	}

	return deps, nil
}

//Wrapper function for DeploymentHoursBetween that uses time.Now as the end time
func (repo *deploymentRepo) HoursUsedSince(userID string, startTime time.Time) (duration int, err error) {
	t, err := repo.DeploymentHoursBetween(userID, startTime, time.Now())
	duration = int(t / time.Hour)
	return
}

//Finds used deployment hours between two times for one user.
func (repo *deploymentRepo) DeploymentHoursBetween(userID string, startTime, endTime time.Time) (t time.Duration, err error) {
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
			Where("timestamp<=?", timeToSQLStr(endTime)).
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

// monthEnd changes t to the end of the month in UTC.
func monthEnd(t time.Time) time.Time {
	return monthStart(t).AddDate(0, 1, 0).Add(-1 * time.Second)
}

// timeToSQLStr formats t in sql format YYYY-MM-DD HH:MM:SS.
func timeToSQLStr(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
