package sugar

import (
	"fmt"
	"time"

	"github.com/ReconfigureIO/platform/service/events"
	"github.com/gin-gonic/gin"
)

func EnqueueEvent(s events.EventService, c *gin.Context, name string, meta map[string]interface{}) error {
	now := time.Now()
	userID := middleware.GetUser(c).ID

	event := models.Event{
		UserID:    userID,
		EventName: name,
		CreatedAt: now,
		Metadata:  meta,
	}

	return s.EnqueueEvent(event)
}
