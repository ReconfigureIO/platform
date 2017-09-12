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
	EventName string
	CreatedAt time.Time
	UserID    string
	Metadata  []map[string]string{}
}
