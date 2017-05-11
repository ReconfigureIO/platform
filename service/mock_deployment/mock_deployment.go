package mock_deployment

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ContainerConfig struct {
	Image   string `json:"image"`
	Command string `json:"command"`
}

type LogsConfig struct {
	Group string `json:"group"`
}

type Deployment struct {
	Container ContainerConfig `json:"container"`
	Logs      LogsConfig      `json:"logs"`
}

type Service struct {
	session *session.Session
	conf    ServiceConfig
}

type ServiceConfig struct {
	LogGroup string
	Image    string
	AMI      string
}

func New(conf ServiceConfig) *Service {
	s := Service{conf: conf}
	s.session = session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))
	return &s
}

func (s *ServiceConfig) ContainerConfig(deployment models.Deployment) Deployment {
	return Deployment{
		Container: ContainerConfig{
			Image:   s.Image,
			Command: deployment.Command,
		},
		Logs: LogsConfig{
			Group: s.LogGroup,
		},
	}
}

func (d Deployment) String() (string, error) {
	buff := bytes.Buffer{}
	enc := json.NewEncoder(base64.NewEncoder(base64.StdEncoding, &buff))
	err := enc.Encode(d)
	return buff.String(), err
}

func (s *Service) RunDeployment(ctx context.Context, deployment models.Deployment) (string, error) {
	ec2Session := ec2.New(s.session)

	encodedConfig, err := s.conf.ContainerConfig(deployment).String()
	if err != nil {
		return "", err
	}

	cfg := ec2.RunInstancesInput{
		ImageId: aws.String(s.conf.AMI),
		InstanceInitiatedShutdownBehavior: aws.String("terminate"),
		InstanceType:                      aws.String("f1.2xlarge"),
		MaxCount:                          aws.Int64(1),
		MinCount:                          aws.Int64(1),
		UserData:                          aws.String(encodedConfig),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Arn: aws.String("arn:aws:iam::398048034572:instance-profile/deployment-worker"),
		},
	}

	_, err = ec2Session.RunInstancesWithContext(ctx, &cfg)
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
