package mock_deployment

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ReconfigureIO/platform/models"
	awsService "github.com/ReconfigureIO/platform/service/aws"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ContainerConfig struct {
	Image   string `json:"image"`
	Command string `json:"command"`
}

type LogsConfig struct {
	Group  string `json:"group"`
	Prefix string `json:"prefix"`
}

type Deployment struct {
	Container ContainerConfig `json:"container"`
	Logs      LogsConfig      `json:"logs"`
}

type Service struct {
	session *session.Session
	Conf    ServiceConfig
}

type ServiceConfig struct {
	LogGroup string
	Image    string
	AMI      string
}

func New(conf ServiceConfig) *Service {
	s := Service{Conf: conf}
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
			Group:  s.LogGroup,
			Prefix: fmt.Sprintf("deployment-%d", deployment.ID),
		},
	}
}

func (d Deployment) String() (string, error) {
	buff := bytes.Buffer{}
	b64Encoder := base64.NewEncoder(base64.StdEncoding, &buff)
	enc := json.NewEncoder(b64Encoder)
	err := enc.Encode(d)
	b64Encoder.Close()
	return buff.String(), err
}

func (s *Service) RunDeployment(ctx context.Context, deployment models.Deployment) (string, error) {
	ec2Session := ec2.New(s.session)

	encodedConfig, err := s.Conf.ContainerConfig(deployment).String()
	if err != nil {
		return "", err
	}

	cfg := ec2.RunInstancesInput{
		ImageId: aws.String(s.Conf.AMI),
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

func (s *Service) GetDeploymentStream(ctx context.Context, deployment models.Deployment) (*cloudwatchlogs.LogStream, error) {
	cwLogs := cloudwatchlogs.New(s.session)

	searchParams := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(s.Conf.LogGroup), // Required
		Descending:          aws.Bool(true),
		Limit:               aws.Int64(1),
		LogStreamNamePrefix: aws.String(fmt.Sprintf("deployment-%d", deployment.ID)),
	}
	resp, err := cwLogs.DescribeLogStreams(searchParams)
	if err != nil {
		return nil, err
	}

	if len(resp.LogStreams) == 0 {
		return nil, awsService.NOT_FOUND
	}
	return resp.LogStreams[0], nil

}
