package events

import (
	"github.com/ReconfigureIO/platform/models"
)

type Event struct{}

type EventService interface {
	DrainEvents()
	EnqueueEvent(models.Event)
	Close()
}

type IntercomConfig struct {
	AccessToken string `env:"RECO_INTERCOM_ACCESS_TOKEN"`
}
