package models

//go:generate mockgen -source=deployment.go -package=models -destination=deployment_mock.go

import (
	"fmt"
	"math"
	"time"

	"github.com/jinzhu/gorm"
)

// DeploymentRepo handles deployment details.
type DeploymentRepo interface {
	// Return a list of deployments, with the statuses specified,
	// limited to that number
	GetWithStatus([]string, int) ([]Deployment, error)

	// Return a list of all deployment for a user, with the statuses specified
	GetWithStatusForUser(string, []string) ([]Deployment, error)

	// DeploymentHoursBtw returns the total time used for deployments between
	// startTime and endTime.
	DeploymentHours(userID string, startTime, endTime time.Time) ([]DeploymentHours, error)

	// ActiveDeployments returns basic information about running deployments.
	ActiveDeployments(userID string) ([]DeploymentHours, error)

	AddEvent(Deployment, DeploymentEvent) error
	SetIP(Deployment, string) error

	GetWithoutIP() ([]Deployment, error)
}

type DeploymentHours struct {
	Id         string
	Started    time.Time
	Terminated time.Time
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

	sqlDeploymentStatusForUser = `SELECT j.id
FROM deployments j
JOIN builds on builds.id = j.build_id
JOIN projects on builds.project_id = projects.id
LEFT join deployment_events e
ON j.id = e.deployment_id
	AND e.timestamp = (
		SELECT max(timestamp)
		FROM deployment_events e1
		WHERE j.id = e1.deployment_id
	)
WHERE (projects.user_id = ? and e.status in (?))
`

	sqlDeploymentHours = `
select j.id as id, started.timestamp as started, coalesce(terminated.timestamp, now()) as terminated
from deployments j
join builds on builds.id = j.build_id
join projects on builds.project_id = projects.id
left join deployment_events started
on j.id = started.deployment_id
    and started.id = (
        select e1.id
        from deployment_events e1
        where j.id = e1.deployment_id and e1.status = 'STARTED'
        limit 1
    )
left outer join deployment_events terminated
on j.id = terminated.deployment_id
    and terminated.id = (
        select e2.id
        from deployment_events e2
        where j.id = e2.deployment_id and e2.status = 'TERMINATED'
        limit 1
    )
where (projects.user_id = ? and coalesce(terminated.timestamp, now()) > ? and coalesce(terminated.timestamp, now()) < ?)
`
	sqlDeploymentInstances = `
select j.id as id, started.timestamp as started, terminated.timestamp as terminated
from deployments j
join builds on builds.id = j.build_id
join projects on builds.project_id = projects.id
left join deployment_events started
on j.id = started.deployment_id
    and started.id = (
        select e1.id
        from deployment_events e1
        where j.id = e1.deployment_id and e1.status = 'STARTED'
        limit 1
    )
left outer join deployment_events terminated
on j.id = terminated.deployment_id
    and terminated.id = (
        select e2.id
        from deployment_events e2
        where j.id = e2.deployment_id and e2.status = 'TERMINATED'
        limit 1
    )
where projects.user_id = ? and terminated IS NULL
`
)

func (repo *deploymentRepo) AddEvent(dep Deployment, event DeploymentEvent) error {
	event.DeploymentID = dep.ID
	err := repo.db.Create(&event).Error
	return err
}

func (repo *deploymentRepo) SetIP(dep Deployment, ip string) error {
	err := repo.db.Model(&dep).Update("ip_address", ip).Error
	return err
}

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
	err = db.Preload("Events", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp ASC")
	}).Where("id in (?)", ids).Find(&deps).Error

	if err != nil {
		return nil, err
	}

	return deps, nil
}

func (repo *deploymentRepo) GetWithStatusForUser(userID string, statuses []string) ([]Deployment, error) {
	db := repo.db
	rows, err := db.Raw(sqlDeploymentStatusForUser, userID, statuses).Rows()
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

func (repo *deploymentRepo) GetWithoutIP() ([]Deployment, error) {
	db := repo.db

	// Find deployments where ip_address field is null
	var deps []Deployment
	err := db.Preload("Events", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp ASC")
	}).Where("ip_address = ?", "").Find(&deps).Error

	if err != nil {
		return nil, err
	}

	return deps, nil
}

func AggregateHoursBetween(deps []DeploymentHours, startTime, endTime time.Time) int {
	t := 0

	for _, dep := range deps {
		s := dep.Started
		// Bound calculated start time to this start time
		if s.Before(startTime) {
			s = startTime
		}

		// Bound calculated end time to this end time
		e := dep.Terminated
		if e.After(endTime) {
			e = endTime
		}
		// Round up and convert to an int
		t += int(math.Ceil(e.Sub(s).Hours()))
	}

	return t
}

func DeploymentHoursBtw(repo DeploymentRepo, userID string, startTime, endTime time.Time) (int, error) {
	deps, err := repo.DeploymentHours(userID, startTime, endTime)
	if err != nil {
		return 0, err
	}
	return AggregateHoursBetween(deps, startTime, endTime), nil
}

func (repo *deploymentRepo) DeploymentHours(userID string, startTime, endTime time.Time) (deps []DeploymentHours, err error) {
	db := repo.db

	rows, err := db.Raw(sqlDeploymentHours, userID, startTime, endTime).Rows()
	if err != nil {
		return nil, err
	}

	deps = []DeploymentHours{}
	for rows.Next() {
		var dep DeploymentHours
		err = db.ScanRows(rows, &dep)
		if err != nil {
			return
		}
		deps = append(deps, dep)
	}
	rows.Close()

	return
}

func (repo *deploymentRepo) ActiveDeployments(userID string) (deps []DeploymentHours, err error) {
	db := repo.db

	rows, err := db.Raw(sqlDeploymentInstances, userID).Rows()
	if err != nil {
		return nil, err
	}

	deps = []DeploymentHours{}
	for rows.Next() {
		var dep DeploymentHours
		err = db.ScanRows(rows, &dep)
		if err != nil {
			return
		}
		deps = append(deps, dep)
	}
	rows.Close()

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
