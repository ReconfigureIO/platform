package models

type DeploymentRepo interface {
	// Return a list of deployments, with the statuses specified,
	// limited to that number
	GetWithStatus([]string, int) ([]Deployment, error)
}

const (
	SQL_DEPLOYMENT_STATUS = `SELECT j.id
FROM deployments j
LEFT join dep_job_events e
ON j.dep_job_id = e.dep_job_id
    AND e.timestamp = (
        SELECT max(timestamp)
        FROM dep_job_events e1
        WHERE j.dep_job_id = e1.dep_job_id
    )
WHERE (e.status in (?))
LIMIT ?
`
)

func (repo *PostgresRepo) GetWithStatus(statuses []string, limit int) ([]Deployment, error) {
	db := repo.DB
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
	err = db.Preload("DepJob").Where("id in (?)", ids).Find(&deps).Error
	if err != nil {
		return nil, err
	}

	return deps, nil
}
