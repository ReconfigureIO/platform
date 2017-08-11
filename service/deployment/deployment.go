package deployment

//go:generate mockgen -source=deployment.go -package=deployment -destination=deployment_mock.go

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

type BuildConfig struct {
	ArtifactUrl string `json:"artifact_url"`
	Agfi        string `json:"agfi"`
}

type Deployment struct {
	Container   ContainerConfig `json:"container"`
	Logs        LogsConfig      `json:"logs"`
	CallbackUrl string          `json:"callback_url"`
	Build       BuildConfig     `json:"build"`
}

type service struct {
	session *session.Session
	Conf    ServiceConfig
}

type ServiceConfig struct {
	LogGroup string `env:"RECO_DEPLOY_LOG_GROUP" envDefault:"/reconfigureio/deployments"`
	Image    string `env:"RECO_DEPLOY_IMAGE" envDefault:"reconfigureio/docker-aws-fpga-runtime:latest"`
	AMI      string `env:"RECO_DEPLOY_AMI"`
	Bucket   string `env:"RECO_DEPLOY_BUCET" envDefault:"reconfigureio-builds"`
}

func New(conf ServiceConfig) Service {
	s := service{Conf: conf}
	s.session = session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))
	return &s
}

// DeploymentRepo handles deployment details.
type Service interface {
	// RunDeployment creates an EC2 instance (a deployment)
	RunDeployment(ctx context.Context, deployment models.Deployment, callbackUrl string) (string, error)
	// StopDeployment stops the EC2 instance associated with a deployment
	StopDeployment(ctx context.Context, deployment models.Deployment) error
	// GetDepDetail does nothing at the moment
	GetDepDetail(id int) (string, error)
	// GetDeploymentStream gets the cloudwatch logstream for a given deployment
	GetDeploymentStream(ctx context.Context, deployment models.Deployment) (*cloudwatchlogs.LogStream, error)
	// DescribeInstanceStatus gets the statuses of the instances associated with
	// a list of deployments
	DescribeInstanceStatus(ctx context.Context, deployments []models.Deployment) (map[string]string, error)
	// GetServiceConfig outputs the configuration of the service
	GetServiceConfig() ServiceConfig
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
			Prefix: fmt.Sprintf("deployment-%s", deployment.ID),
		},
		Build: BuildConfig{
			ArtifactUrl: fmt.Sprintf("s3://%s/%s", s.Bucket, deployment.Build.ArtifactUrl()),
			Agfi:        deployment.Build.FPGAImage,
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

func (s *service) RunDeployment(ctx context.Context, deployment models.Deployment, callbackUrl string) (string, error) {
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

func (s *service) StopDeployment(ctx context.Context, deployment models.Deployment) error {
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

func (s *service) GetDepDetail(id int) (string, error) {
	return "imaginary", nil
}

func (s *service) GetServiceConfig() ServiceConfig {
	return s.Conf
}

func (s *service) GetDeploymentStream(ctx context.Context, deployment models.Deployment) (*cloudwatchlogs.LogStream, error) {
	cwLogs := cloudwatchlogs.New(s.session)

	searchParams := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(s.Conf.LogGroup), // Required
		Descending:          aws.Bool(true),
		Limit:               aws.Int64(1),
		LogStreamNamePrefix: aws.String(fmt.Sprintf("deployment-%s", deployment.ID)),
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

func (s *service) DescribeInstanceStatus(ctx context.Context, deployments []models.Deployment) (map[string]string, error) {
	ret := make(map[string]string)

	var instanceids []*string
	for _, deployment := range deployments {
		instanceids = append(instanceids, &deployment.InstanceID)
	}
	ec2Session := ec2.New(s.session)

	cfg := ec2.DescribeInstancesInput{
		InstanceIds: instanceids,
	}

	results, err := ec2Session.DescribeInstancesWithContext(ctx, &cfg)
	if err != nil {
		return ret, err
	}

	for _, reservation := range results.Reservations {
		for _, instance := range reservation.Instances {
			ret[*instance.InstanceId] = *instance.State.Name
		}
	}

	return ret, nil
}