package intercom

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	intercomOfficial "gopkg.in/intercom/intercom-go.v2"
)

type NewIntercomEventService(config IntercomConfig, depth int) (EventService, error) {

}

type intercomEventService struct {
	
}

func (s intercomEventService) DrainEvents() {
	ic := intercomOfficial.NewClient("access_token", s.config.AccessToken)
	for event := range s.c {
		icEvent := intercomOfficial.Event{
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



// // Service is an intercom service.
// type Service interface {
// 	Save(event models.Event) error
// 	Conf() *ServiceConfig
// }

// type service struct {
// 	conf ServiceConfig
// }

// // ServiceConfig holds configuration for service.
// type ServiceConfig struct {
// 	AccessToken string `env:"RECO_INTERCOM_ACCESS_TOKEN"`
// }

// // New creates a new service with conf.
// func New(conf ServiceConfig) Service {
// 	s := service{conf: conf}
// 	return &s
// }

// func (s *service) Save(event models.Event) error {


// func (s *service) Conf() *ServiceConfig {
// 	return &s.conf
// }
