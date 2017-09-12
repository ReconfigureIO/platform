package events

import (
	"time"

	"github.com/ReconfigureIO/platform/service/intercom"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/gin-gonic/gin"
)

var eventQueue = make(chan models.Event, 100)

func QueueEvent(event models.Event) {
	eventQueue <- event
}

func PostEvents() {
	go func(){
		for e := range eventQueue {
			err := intercom.Save(e)
			if err != nil {
				log.Printf("Intercom Error: %s", err)
			}
		}
	}
}

func CreateEvent(c *gin.Context, postEvent PostEvent) {
	eventTime := time.Now()
	if postEvent.CreatedAt != nil{
		eventTime = postEvent.CreatedAt
	}
	event := models.Event{
		UserID: middleware.GetUser(c).ID,
		EventName: postEvent.EventName,
		CreatedAt: eventTime,
		Metadata: postEvent.Metadata,
	}
	QueueEvent(event)
}

type PostEvent struct {
	EventName string
	CreatedAt time.Time
	Metadata  map[string]interface{}
}
