package mock_deployment

import (
	"context"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Service struct {
	session *session.Session
	conf    ServiceConfig
}

type ServiceConfig struct {
	Foo string
	Bar string
	Baz string
}

func New(conf ServiceConfig) *Service {
	s := Service{conf: conf}
	s.session = session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))
	return &s
}

func (s *Service) RunDeployment(ctx context.Context, deployment models.Deployment) (string, error) {
	ec2Session := ec2.New(s.session)

	cfg := ec2.RunInstancesInput{
		ImageId: aws.String("ami-7427bb62"),
		InstanceInitiatedShutdownBehavior: aws.String("terminate"),
		InstanceType:                      aws.String("f1.2xlarge"),
		MaxCount:                          aws.Int64(1),
		MinCount:                          aws.Int64(1),
		UserData:                          aws.String("base 64 encode some config"),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String("deployment-worker"),
		},
	}

	_, err := ec2Session.RunInstancesWithContext(ctx, &cfg)
	if err != nil {
		return "", err
	}

	return "Hello", nil
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
