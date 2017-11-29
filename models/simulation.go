package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type SimulationRepo interface {
	ActiveSimulations(user User) ([]Simulation, error)
}

type simulationRepo struct{ db *gorm.DB }

func SimulationDataSource(db *gorm.DB) SimulationRepo {
	return &simulationRepo{db: db}
}

const (
	SQL_ACTIVE_SIMULATIONS = `
select j.id as id, started.timestamp as started, terminated.timestamp as terminated
from simulations j
join projects on projects.id = j.project_id
left join batch_jobs on batch_jobs.id = j.batch_job_id
left join batch_job_events started
on batch_jobs.id = started.batch_job_id
    and started.id = (
        select e1.id
        from batch_job_events e1
        where j.batch_job_id = e1.batch_job_id and e1.status = 'STARTED'
    )
left outer join batch_job_events terminated
on batch_jobs.id = terminated.batch_job_id
    and terminated.id = (
        select e2.id
        from batch_job_events e2
        where j.batch_job_id = e2.batch_job_id and e2.status = 'TERMINATED'
    )
where projects.user_id = ? and terminated IS NULL
`
)

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

// Status returns simulation status.
func (s *Simulation) Status() string {
	events := s.BatchJob.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return StatusSubmitted
}

// PostSimulation is the post request body for new simulation.
type PostSimulation struct {
	ProjectID string `json:"project_id" validate:"nonzero"`
	Command   string `json:"command" validate:"nonzero"`
}

func (repo *simulationRepo) ActiveSimulations(user User) ([]Simulation, error) {
	db := repo.db

	rows, err := db.Raw(SQL_ACTIVE_SIMULATIONS, user.ID).Rows()
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for rows.Next() {
		var sst SimStartedTerminated
		err = db.ScanRows(rows, &sst)
		if err != nil {
			return []Simulation{}, err
		}
		ids = append(ids, sst.Id)
	}
	rows.Close()

	var sims []Simulation
	err = db.Preload("BatchJob").Preload("BatchJob.Events").Where("id in (?)", ids).Find(&sims).Error
	if err != nil {
		return nil, err
	}

	return sims, nil
}

//scanrows needs an exact match to tie a row to an object.
//this object has an ID, started, terminated time
type SimStartedTerminated struct {
	Id         string
	Started    time.Time
	Terminated time.Time
}
