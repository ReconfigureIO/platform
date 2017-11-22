package sugar

import (
	"time"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/gin-gonic/gin"
)

func EnqueueEvent(s events.EventService, c *gin.Context, name string, userID string, meta map[string]interface{}) {
	now := time.Now()

	event := events.Event{
		UserID:    userID,
		EventName: name,
		CreatedAt: now,
		Metadata:  meta,
	}

	s.EnqueueEvent(event)
}
