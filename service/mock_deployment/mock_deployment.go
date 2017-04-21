package mock_deployment

import (
//"github.com/ReconfigureIO/platform/models"
)

type Service struct {
	session string
	conf    ServiceConfig
}

type ServiceConfig struct {
	Foo string
	Bar string
	Baz string
}

func New(conf ServiceConfig) *Service {
	s := Service{conf: conf}
	s.session = "something"
	return &s
}

func (s *Service) RunDeployment(command string, buildID int) (string, error) {

	return "This function does nothing yet", nil
}

func (s *Service) HaltDep(id int) error {
	return nil
}

func (s *Service) GetDepDetail(id int) (string, error) {
	return "imaginary", nil
}

func (s *Service) GetJobStream(id int) (string, error) {

	return "doing doing deployed", nil
}
