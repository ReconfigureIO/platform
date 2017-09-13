package events

import (
	"time"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/intercom"
	"github.com/gin-gonic/gin"
)

type Event struct{}

type EventService interface {
	DrainEvents()
	EnqueueEvent(models.Event) error
	Close()
}

type IntercomConfig struct {
	AccessToken string `env:"RECO_INTERCOM_ACCESS_TOKEN"`
}
