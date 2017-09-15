package events

import (
	"time"
)

type Event struct {
	UserID    string
	EventName string
	CreatedAt time.Time
	Metadata  map[string]interface{}
}

type EventService interface {
	DrainEvents()
	EnqueueEvent(Event)
	Close()
}

type IntercomConfig struct {
	AccessToken string `env:"RECO_INTERCOM_ACCESS_TOKEN"`
}
