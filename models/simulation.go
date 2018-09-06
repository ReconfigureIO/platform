package models

//go:generate mockgen -source=simulation.go -package=models -destination=simulation_mock.go

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
)

type SimulationRepo interface {
	// Return a list of simulations, with the statuses specified,
	// limited to that number
	StoreSimulationReport(Simulation, ReportV1) error
	GetSimulationReport(simulation Simulation) (SimulationReport, error)
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

type SimulationReport struct {
	uuidHook
	ID           string     `gorm:"primary_key" json:"-"`
	Simulation   Simulation `json:"-" gorm:"ForeignKey:SimulationID"`
	SimulationID string     `json:"-"`
	Version      string     `json:"-"`
	Report       string     `json:"report" sql:"type:JSONB NOT NULL DEFAULT '{}'::JSONB"`
}

// StoreSimulationReport takes a simulation and reportV1,
// and attaches the report to the simulation
func (repo *simulationRepo) StoreSimulationReport(simulation Simulation, report ReportV1) error {
	db := repo.db
	newBytes, err := json.Marshal(&report)
	if err != nil {
		return err
	}
	simulationReport := SimulationReport{
		SimulationID: simulation.ID,
		Version:      "v1",
		Report:       string(newBytes),
	}
	err = db.Create(&simulationReport).Error
	return err
}

// GetSimulationReport gets a simulation report given a simulation
func (repo *simulationRepo) GetSimulationReport(simulation Simulation) (SimulationReport, error) {
	report := SimulationReport{}
	db := repo.db

	err := db.Model(&simulation).Related(&report).Error
	return report, err
}
