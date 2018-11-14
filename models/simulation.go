package models

//go:generate mockgen -source=simulation.go -package=models -destination=simulation_mock.go

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
)

type SimulationRepo interface {
	StoreReport(id string, report Report) error
	GetReport(id string) (SimulationReport, error)
	ByID(simulationID string) (Simulation, error)
	ByIDForUser(simulationID, userID string) (Simulation, error)
}

type simulationRepo struct{ db *gorm.DB }

// SimulationDataSource returns the data source for simulations.
func SimulationDataSource(db *gorm.DB) SimulationRepo {
	return &simulationRepo{db: db}
}

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
	if length > 0 {
		return events[length-1].Status
	}
	return StatusSubmitted
}

// PostSimulation is the post request body for new simulation.
type PostSimulation struct {
	ProjectID string `json:"project_id" validate:"nonzero"`
	Command   string `json:"command" validate:"nonzero"`
}

type SimulationReport struct {
	uuidHook
	ID           string     `gorm:"primary_key" json:"-"`
	Simulation   Simulation `json:"-" gorm:"ForeignKey:SimulationID"`
	SimulationID string     `json:"-"`
	Version      string     `json:"-"`
	Report       string     `json:"report" sql:"type:JSONB NOT NULL DEFAULT '{}'::JSONB"`
}

// StoreReport takes a simulation and Report, and attaches the report to the
// simulation in our DB.
func (repo *simulationRepo) StoreReport(id string, report Report) error {
	newBytes, err := json.Marshal(&report)
	if err != nil {
		return err
	}
	simReport := SimulationReport{
		SimulationID: id,
		Version:      "v1",
		Report:       string(newBytes),
	}
	err = repo.db.Create(&simReport).Error
	return err
}

// GetReport gets a simulation report given a simulation
func (repo *simulationRepo) GetReport(id string) (SimulationReport, error) {
	var report SimulationReport
	err := repo.db.Where("simulation_id = ?", id).First(&report).Error
	return report, err
}

func (repo *simulationRepo) preload() {
	repo.db.Preload("Project").
		Preload("BatchJob").
		Preload("BatchJob.Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp ASC")
		})
}

func (repo *simulationRepo) ByID(simID string) (Simulation, error) {
	var sim Simulation
	repo.preload()
	err := repo.db.First(&sim, "simulations.id = ?", simID).Error
	return sim, err
}

func (repo *simulationRepo) ByIDForUser(simID string, userID string) (Simulation, error) {
	repo.db.Joins("join projects on projects.id = simulations.project_id").
		Where("projects.user_id=?", userID)
	return repo.ByID(simID)
}
