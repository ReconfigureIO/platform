package events

import (
	"log"
	"time"

	intercom "gopkg.in/intercom/intercom-go.v2"
)

func NewIntercomEventService(config IntercomConfig, depth int) EventService {
	return intercomEventService{
		ICClient: intercom.NewClient(config.AccessToken, ""),
		Queue:    make(chan Event, depth),
	}
}

type intercomEventService struct {
	ICClient *intercom.Client
	Queue    chan Event
}

func (s intercomEventService) DrainEvents() {
	ic := s.ICClient
	for event := range s.Queue {
		icEvent := intercom.Event{
			UserID:    event.UserID,
			EventName: event.EventName,
			CreatedAt: int64(time.Time(event.CreatedAt).Unix()),
			Metadata:  event.Metadata,
		}
		err := ic.Events.Save(&icEvent)
		if err != nil {
			log.Printf("Intercom Error: %s", err)
		}
	}
}

func (s intercomEventService) EnqueueEvent(event Event) {
	select {
	case s.Queue <- event:
	default:
		log.Printf("Event queue full. Discarding event: %s", event)
	}
}

func (s intercomEventService) Close() {
	s.DrainEvents()
}
