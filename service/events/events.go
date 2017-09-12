package events

import (
	// "bytes"
	// "context"
	// "errors"
	// "io"
	// "io/ioutil"
	// "os"
	"time"
	// "github.com/ReconfigureIO/platform/models"
	// "github.com/abiosoft/errs"
)

// Service is an AWS service.
type Service interface {
	QueueEvent()
	PostEvents()
	Conf() *ServiceConfig
}

type service struct {
	conf ServiceConfig
}

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	//TODO
}

// New creates a new service with conf.
func New(conf ServiceConfig) Service {
	s := service{conf: conf}
	return &s
}

func (s *service) QueueEvent() (string, error) {
	//TODO
}

func (s *service) PostEvents() error {
	//TODO
}

func (s *service) Conf() *ServiceConfig {
	return &s.conf
}

type Event struct {
	EventName string
	CreatedAt time.Time
	UserID    string
	Metadata  []map[string]string{}
}
