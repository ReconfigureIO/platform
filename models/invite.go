package models

import (
	"time"

	"github.com/dchest/uniuri"
)

// InviteToken model.
type InviteToken struct {
	Token      string    `gorm:"type:varchar(128);primary_key" json:"token"`
	IntercomId string    `gorm:"type:varchar(128)" json:"-"`
	Timestamp  time.Time `json:"created_at"`
}

// NewInviteToken creates a new invite token.
func NewInviteToken() InviteToken {
	return InviteToken{Token: uniuri.NewLen(64), Timestamp: time.Now()}
}

func (i *InviteToken) isValid(now time.Time) bool {
	return now.Before(i.Timestamp.Add(7 * 24 * time.Hour))
}
