package api

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
)

// Queue manages job queue using database.
type Queue struct{}

// Push pushes a job into the queue.
func (q Queue) Push(jobType string, jobID string, weight int) error {
	entry := models.QueueEntry{
		Type:      jobType,
		TypeID:    jobID,
		Weight:    weight,
		Status:    models.StatusQueued,
		CreatedAt: time.Now(),
	}
	return db.Create(&entry).Error
}

// Update updates a job on the queue.
func (q Queue) Update(jobType string, jobID string, status string) error {
	return db.Model(&models.QueueEntry{}).
		Where("type = ? AND type_id = ?", jobType, jobID).
		Update("status", status).Error
}

// Count counts the amount of jobs with status.
func (q Queue) Count(jobType, status string) (int, error) {
	var count int
	err := db.Model(&models.QueueEntry{}).
		Where("status = ?", status).Count(&count).Error
	return count, err
}

// Fetch fetches jobs by priority in the queue.
func (q Queue) Fetch(jobType string, limit int) ([]string, error) {
	var jobs []string
	rows, err := db.Model(&models.QueueEntry{}).
		Select("type_id").
		Where("status = ?", models.StatusQueued).
		Order("weight desc").
		Limit(limit).
		Rows()
	if err != nil {
		return jobs, err
	}
	for rows.Next() {
		var job string
		err := rows.Scan(&job)
		if err != nil {
			return jobs, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
