package events

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	log "github.com/sirupsen/logrus"
	intercom "gopkg.in/intercom/intercom-go.v2"
)

func NewIntercomEventService(config IntercomConfig, depth int) EventService {
	return intercomEventService{
		ICClient:  intercom.NewClient(config.AccessToken, ""),
		Queue:     make(chan Event, depth),
		usersSeen: make(chan models.User, depth),
	}
}

type intercomEventService struct {
	ICClient  *intercom.Client
	Queue     chan Event
	usersSeen chan models.User
}

func (s intercomEventService) DrainEvents() {
	ic := s.ICClient
	for {
		select {
		case event := <-s.Queue:
			icEvent := intercom.Event{
				UserID:    event.UserID,
				EventName: event.EventName,
				CreatedAt: int64(time.Time(event.CreatedAt).Unix()),
				Metadata:  event.Metadata,
			}
			err := ic.Events.Save(&icEvent)
			if err != nil {
				log.Printf("Intercom Error: %s\n", err)
			}
		case user := <-s.usersSeen:
			icUser := intercom.User{
				UserID:              user.ID,
				Email:               user.Email,
				SignedUpAt:          user.CreatedAt.Unix(),
				UpdateLastRequestAt: intercom.Bool(true),
			}
			_, err := ic.Users.Save(&icUser)
			if err != nil {
				log.Printf("Intercom Error: %s\n", err)
			}
		}
	}
}

func (s intercomEventService) Seen(user models.User) {
	select {
	case s.usersSeen <- user:
	default:
		log.Printf("User seen queue full. Discarding event: %v", user)
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
