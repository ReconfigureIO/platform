package mock_deployment

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ReconfigureIO/platform/models"
	awsservice "github.com/ReconfigureIO/platform/service/aws"
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
	Container   ContainerConfig `json:"container"`
	Logs        LogsConfig      `json:"logs"`
	CallbackUrl string          `json:"callback_url"`
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

func (s *ServiceConfig) ContainerConfig(deployment models.Deployment, callbackUrl string) Deployment {
	return Deployment{
		CallbackUrl: callbackUrl,
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

func (s *Service) RunDeployment(ctx context.Context, deployment models.Deployment, callbackUrl string) (string, error) {
	ec2Session := ec2.New(s.session)

	encodedConfig, err := s.Conf.ContainerConfig(deployment, callbackUrl).String()
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

	resp, err := ec2Session.RunInstancesWithContext(ctx, &cfg)
	if err != nil {
		return "", err
	}

	InstanceId := *resp.Instances[0].InstanceId

	return InstanceId, nil
}

func (s *Service) StopDeployment(ctx context.Context, deployment models.Deployment) error {
	InstanceId := deployment.InstanceID
	ec2Session := ec2.New(s.session)

	cfg := ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(InstanceId),
		},
	}

	_, err := ec2Session.TerminateInstancesWithContext(ctx, &cfg)

	return err
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
		return nil, awsservice.ErrNotFound
	}
	return resp.LogStreams[0], nil

}

func (s *Service) DescribeInstanceStatus(ctx context.Context, deployments []models.Deployment) ([]models.InstanceStatus, error) {
	var instanceids []string
	for _, deployment := range deployments {
		instanceids = append(instanceids, deployment.InstanceID)
	}
	ec2Session := ec2.New(s.session)

	cfg := ec2.DescribeInstancesInput{
		InstanceIds: instanceids,
	}

	results, err := ec2Session.DescribeInstancesWithContext(ctx, &cfg)
	if err != nil {
		return nil, err
	}

	var instancestatuses []models.InstanceStatus
	for _, reservation := range results.Reservations {
		for _, instance := range reservation.Instances {
			instancestatus := models.InstanceStatus{
				ID:     instance.InstanceId,
				Status: instance.InstanceState.Name,
			}
			instancestatuses = append(instancestatuses, instancestatus)
		}
	}

	return instancestatuses, nil
}
