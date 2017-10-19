package models

import "time"

// QueueEntry is a queue entry.
type QueueEntry struct {
	uuidHook
	ID           string `gorm:"primary_key"`
	Type         string `gorm:"default:'deployment'"`
	TypeID       string `gorm:"not_null"`
	Weight       int
	Status       string
	CreatedAt    time.Time
	DispatchedAt time.Time
}
