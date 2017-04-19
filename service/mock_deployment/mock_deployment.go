package mock_deployment

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

func (s *Service) HaltDep(id string) error {
	return err
}

func (s *Service) GetDepDetail(id string) (string, error) {
	return "imaginary", nil
}

func (s *Service) GetJobStream(id string) (string, error) {

	return "doing doing deployed", nil
}
