package events

import (
	"time"

	"github.com/ReconfigureIO/platform/service/intercom"
)

var eventQueue []Event

func QueueEvent(event Event) error {
	eventQueue = append(eventQueue, event)
}

func PostEvents() error {
	for e := range eventQueue {
		intercom.Save(e)
	}
}

type Event struct {
	UserID    string
	EventName string
	CreatedAt time.Time
	Metadata  []map[string]interface{}
}
