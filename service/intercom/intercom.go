package intercom

import (
	intercomOfficial "gopkg.in/intercom/intercom-go.v2"
)

// Service is an intercom service.
type Service interface {
	Save(event Event) error
	Conf() *ServiceConfig
}

type service struct {
	conf ServiceConfig
}

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	AccessToken string `env:"RECO_INTERCOM_ACCESS_TOKEN"`
}

// New creates a new service with conf.
func New(conf ServiceConfig) Service {
	s := service{conf: conf}
	return &s
}

func (s *service) Save(event Event) error {
	ic := intercomOfficial.NewClient("access_token", service.conf.AccessToken)
	err := ic.Events.Save(&event)
}

func (s *service) Conf() *ServiceConfig {
	return &s.conf
}
