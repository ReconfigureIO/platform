package models

import (
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
join projects on simulations.project_id = projects.id
join batchjobs on simulations.batchjob_id = batchjobs.id
left join batchjob_events started
on batchjobs.id = started.batchjob_id
    and started.id = (
        select e1.id
        from batchjob_events e1
        where j.id = e1.batchjob_id and e1.status = 'STARTED'
    )
left outer join batchjob_events terminated
on batchjobs.id = terminated.batchjob_id
    and terminated.id = (
        select e2.id
        from batchjob_events e2
        where batchjobs.id = e2.batchjob_id and e2.status = 'TERMINATED'
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

	sims := []Simulation{}
	for rows.Next() {
		var sim Simulation
		err = db.ScanRows(rows, &sim)
		if err != nil {
			return nil, err
		}
		sims = append(sims, sim)
	}
	rows.Close()

	return sims, nil
}
