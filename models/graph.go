package models

//go:generate mockgen -source=graph.go -package=models -destination=graph_mock.go

import (
	"fmt"
)

// Graph model.
type Graph struct {
	uuidHook
	ID         string   `gorm:"primary_key" json:"id"`
	Project    Project  `json:"project" gorm:"ForeignKey:ProjectID"`
	ProjectID  string   `json:"-"`
	BatchJob   BatchJob `json:"job" gorm:"ForeignKey:BatchJobId"`
	BatchJobID int64    `json:"-"`
	Token      string   `json:"-"`
	Type       string   `json:"type" gorm:"default:'dataflow'"`
}

// The place to upload graph input to
// should be a tar.gz
func (graph Graph) InputUrl() string {
	return fmt.Sprintf("graphs/%s/input.tar.gz", graph.ID)
}

// The place to graph artifacts will be uploaded to
// Should be a zip file
func (graph Graph) ArtifactUrl() string {
	return fmt.Sprintf("graphs/%s/graph.zip", graph.ID)
}

// Status returns buikld status.
func (b *Graph) Status() string {
	events := b.BatchJob.Events
	length := len(events)
	if len(events) > 0 {
		return events[length-1].Status
	}
	return StatusSubmitted
}

// HasStarted returns if the graph has started.
func (b *Graph) HasStarted() bool {
	return hasStarted(b.Status())
}

// HasFinished returns if graph is finished.
func (b *Graph) HasFinished() bool {
	return hasFinished(b.Status())
}

// PostGraph is post request body for a new graph.
type PostGraph struct {
	ProjectID string `json:"project_id" validate:"nonzero"`
}
