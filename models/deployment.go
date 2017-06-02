package models

import (
	"github.com/jinzhu/gorm"
)

type DB gorm.DB

type DeploymentRepo interface {
	// Return a list of deployments, with the statuses specified,
	// limited to that number
	GetWithStatus([]string, int) ([]Deployment, error)
}

const (
	SQL_DEPLOYMENT_STATUS = `SELECT j.dep_id
FROM dep_jobs j
LEFT join dep_job_events e
ON j.id = e.dep_job_id
    AND e.timestamp = (
        SELECT max(timestamp)
        FROM dep_job_events e1
        WHERE j.id = e1.dep_job_id
    )
WHERE (e.status in (?))
LIMIT ?
`
)

func (d *DB) GetWithStatus(statuses []string, limit int) ([]Deployment, error) {
	db := (*gorm.DB)(d)
	rows, err := db.Raw(SQL_DEPLOYMENT_STATUS, statuses, limit).Rows()
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
	err = db.Where("id in (?)", ids).Find(&deps).Error
	if err != nil {
		return nil, err
	}

	return deps, nil
}
