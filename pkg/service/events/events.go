package events

import (
	"time"

	"github.com/ReconfigureIO/platform/pkg/models"
)

type Event struct {
	UserID    string
	EventName string
	CreatedAt time.Time
	Metadata  map[string]interface{}
}

type EventService interface {
	// Mark that a user has been seen
	Seen(models.User)
	DrainEvents()
	EnqueueEvent(Event)
	Close()
}

type IntercomConfig struct {
	AccessToken string `env:"RECO_INTERCOM_ACCESS_TOKEN"`
}
