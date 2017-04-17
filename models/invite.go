package models

import (
	"github.com/dchest/uniuri"
	"time"
)

type InviteToken struct {
	Token     string `gorm:"type:varchar(128);primary_key"`
	Timestamp time.Time
}

func NewInviteToken() InviteToken {
	return InviteToken{Token: uniuri.NewLen(64), Timestamp: time.Now()}
}

// Check if the token is less than a week old
func (i *InviteToken) IsValid(now time.Time) bool {
	return now.Before(i.Timestamp.Add(7 * 24 * time.Hour))
}
