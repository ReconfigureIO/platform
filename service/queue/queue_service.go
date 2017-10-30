package queue

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
)

// QueueService manages job queue using database.
type QueueService struct {
	db *gorm.DB
}

// Push pushes a job into the queue.
func (q *QueueService) Push(jobType string, job Job) error {
	entry := models.QueueEntry{
		Type:      jobType,
		TypeID:    job.ID,
		User:      job.User,
		UserID:    job.User.ID,
		Weight:    job.Weight,
		Status:    models.StatusQueued,
		CreatedAt: time.Now(),
	}
	return q.db.Create(&entry).Error
}

// Update updates a job on the queue.
func (q *QueueService) Update(jobType string, jobID string, status string) error {
	return q.db.Model(&models.QueueEntry{}).
		Where("type = ? AND type_id = ?", jobType, jobID).
		Update("status", status).Error
}

// Count counts the amount of jobs with status.
func (q *QueueService) Count(jobType, status string) (int, error) {
	var count int
	err := q.db.Model(&models.QueueEntry{}).
		Where("status = ? AND type = ?", status, jobType).
		Count(&count).Error
	return count, err
}

// Count counts the amount of jobs with status.
func (q *QueueService) CountUserJobsInStatus(jobType string, user models.User, status string) (int, error) {
	var count int
	err := q.db.Model(&models.QueueEntry{}).
		Where("status = ?", status).
		Where("type = ?", jobType).
		Where("user_id = ?", user.ID).
		Count(&count).Error
	return count, err
}

// Fetch fetches jobs by priority in the queue.
func (q *QueueService) Fetch(jobType string, limit int) ([]string, error) {
	var jobs []string
	rows, err := q.db.Model(&models.QueueEntry{}).
		Select("type_id").
		Where("status = ? AND type = ?", models.StatusQueued, jobType).
		Order("weight desc, created_at").
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

// Fetch fetches jobs with status.
func (q *QueueService) FetchWithStatus(jobType string, status string) ([]string, error) {
	var jobs []string
	rows, err := q.db.Model(&models.QueueEntry{}).
		Select("type_id").
		Where("status = ? AND type = ?", status, jobType).
		Order("weight desc, created_at").
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
