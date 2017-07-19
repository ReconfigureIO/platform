package models

import (
	"github.com/jinzhu/gorm"
)

type BuildRepo interface {
	// Return a list of deployments, with the statuses specified,
	// limited to that number
	GetBuildsWithStatus([]string, int) ([]Build, error)
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