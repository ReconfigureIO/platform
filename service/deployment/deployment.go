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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	//log "github.com/sirupsen/logrus"
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
	LogGroup      string `env:"RECO_DEPLOY_LOG_GROUP" envDefault:"/reconfigureio/deployments"`
	Image         string `env:"RECO_DEPLOY_IMAGE" envDefault:"reconfigureio/docker-aws-fpga-runtime:latest"`
	AMI           string `env:"RECO_DEPLOY_AMI"`
	Bucket        string `env:"RECO_DEPLOY_BUCKET" envDefault:"reconfigureio-builds"`
	Subnet        string `env:"RECO_DEPLOY_SUBNET" envDefault:"subnet-fa2a9c9e"`
	SecurityGroup string `env:"RECO_DEPLOY_SG" envDefault:"sg-7fbfbe0c"`
}

func newService(conf ServiceConfig) *service {
	s := service{Conf: conf}
	s.session = session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))
	return &s
}

func New(conf ServiceConfig) Service {
	return newService(conf)
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
	DescribeInstanceIPs(ctx context.Context, deployments []models.Deployment) (map[string]string, error)
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

func (s *service) runSpotInstance(ctx context.Context, encodedConfig string, dryRun bool) (string, error) {
	ec2Session := ec2.New(s.session)

	launch := ec2.RequestSpotLaunchSpecification{
		ImageId:      aws.String(s.Conf.AMI),
		InstanceType: aws.String("f1.2xlarge"),
		UserData:     aws.String(encodedConfig),
		Placement: &ec2.SpotPlacement{
			AvailabilityZone: aws.String("us-east-1d"),
		},
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			&ec2.InstanceNetworkInterfaceSpecification{
				DeviceIndex:              aws.Int64(0),
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
				SubnetId:                 aws.String(s.Conf.Subnet),
				Groups:                   []*string{aws.String(s.Conf.SecurityGroup)},
			},
		},
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Arn: aws.String("arn:aws:iam::398048034572:instance-profile/deployment-worker"),
		},
	}

	cfg := ec2.RequestSpotInstancesInput{
		DryRun:              aws.Bool(dryRun),
		InstanceCount:       aws.Int64(1),
		LaunchSpecification: &launch,
		SpotPrice:           aws.String("0.60"),
		Type:                aws.String("one-time"),
	}

	resp, err := ec2Session.RequestSpotInstancesWithContext(ctx, &cfg)
	if err != nil {
		return "", err
	}

	InstanceId := *resp.SpotInstanceRequests[0].SpotInstanceRequestId

	return InstanceId, nil
}

func (s *service) runInstance(ctx context.Context, encodedConfig string, dryRun bool) (string, error) {
	ec2Session := ec2.New(s.session)

	cfg := ec2.RunInstancesInput{
		DryRun:  aws.Bool(dryRun),
		ImageId: aws.String(s.Conf.AMI),
		InstanceInitiatedShutdownBehavior: aws.String("terminate"),
		InstanceType:                      aws.String("f1.2xlarge"),
		MaxCount:                          aws.Int64(1),
		MinCount:                          aws.Int64(1),
		UserData:                          aws.String(encodedConfig),
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			&ec2.InstanceNetworkInterfaceSpecification{
				DeviceIndex:              aws.Int64(0),
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
				SubnetId:                 aws.String(s.Conf.Subnet),
				Groups:                   []*string{aws.String(s.Conf.SecurityGroup)},
			},
		},
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

func (s *service) RunDeployment(ctx context.Context, deployment models.Deployment, callbackUrl string) (string, error) {
	encodedConfig, err := s.Conf.ContainerConfig(deployment, callbackUrl).String()
	if err != nil {
		return "", err
	}

	if deployment.SpotInstance {
		instanceId, err := s.runSpotInstance(ctx, encodedConfig, false)
		return instanceId, err
	}

	instanceId, err := s.runInstance(ctx, encodedConfig, false)
	return instanceId, err
}

func (s *service) stopInstance(ctx context.Context, InstanceId string) error {
	ec2Session := ec2.New(s.session)

	cfg := ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(InstanceId),
		},
	}

	_, err := ec2Session.TerminateInstancesWithContext(ctx, &cfg)

	return err
}

func (s *service) stopSpotInstance(ctx context.Context, InstanceId string) error {
	ec2Session := ec2.New(s.session)

	input := &ec2.CancelSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{
			aws.String(InstanceId),
		},
	}

	_, err := ec2Session.CancelSpotInstanceRequestsWithContext(ctx, input)
	if err != nil {
		return err
	}

	descInput := &ec2.DescribeSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{
			aws.String(InstanceId),
		},
	}

	result, err := ec2Session.DescribeSpotInstanceRequestsWithContext(ctx, descInput)
	if err != nil {
		return err
	}

	instanceIds := []*string{}

	for _, req := range result.SpotInstanceRequests {
		if req.InstanceId != nil {
			instanceIds = append(instanceIds, req.InstanceId)
		}
	}

	if len(instanceIds) > 0 {
		cfg := ec2.TerminateInstancesInput{
			InstanceIds: instanceIds,
		}

		_, err = ec2Session.TerminateInstancesWithContext(ctx, &cfg)
	}

	return err
}

func (s *service) StopDeployment(ctx context.Context, deployment models.Deployment) error {
	InstanceId := deployment.InstanceID

	if deployment.SpotInstance {
		return s.stopSpotInstance(ctx, InstanceId)
	}
	return s.stopInstance(ctx, InstanceId)

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

func isNotFound(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidSpotInstanceRequestID.NotFound":
				return true
			case "InvalidInstanceID.Malformed":
				return true
			default:
				return false
			}
		}
	}
	return false
}

func (s *service) DescribeInstanceStatus(ctx context.Context, deployments []models.Deployment) (map[string]string, error) {
	ret := make(map[string]string)

	var instanceids []*string
	var spotInstanceIDs []*string
	for _, deployment := range deployments {
		if deployment.SpotInstance {
			spotInstanceIDs = append(spotInstanceIDs, &deployment.InstanceID)
		} else {
			instanceids = append(instanceids, &deployment.InstanceID)
		}

	}
	ec2Session := ec2.New(s.session)

	if len(instanceids) > 0 {
		//regular instances
		cfg := ec2.DescribeInstancesInput{
			InstanceIds: instanceids,
		}

		results, err := ec2Session.DescribeInstancesWithContext(ctx, &cfg)
		if err != nil {
			if !isNotFound(err) {
				return ret, err
			}
		}

		for _, reservation := range results.Reservations {
			for _, instance := range reservation.Instances {
				ret[*instance.InstanceId] = *instance.State.Name
			}
		}
	}

	if len(spotInstanceIDs) > 0 {
		//spot instance
		cfgSpot := ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: spotInstanceIDs,
		}

		spotResults, err := ec2Session.DescribeSpotInstanceRequestsWithContext(ctx, &cfgSpot)
		if err != nil {
			if isNotFound(err) {
				return ret, nil
			}
			return ret, err
		}

		// A map for the spotinstance ec2 instances
		spotInstanceMap := make(map[string]string)
		spotInstanceIds := []*string{}

		for _, spotInstanceRequest := range spotResults.SpotInstanceRequests {
			instanceId := (*string)(spotInstanceRequest.InstanceId)
			if instanceId != nil {
				spotInstanceIds = append(spotInstanceIds, instanceId)
				spotId := (*string)(spotInstanceRequest.SpotInstanceRequestId)
				spotInstanceMap[*instanceId] = *spotId
			}
		}

		spotInstanceResults, err := ec2Session.DescribeInstancesWithContext(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: spotInstanceIds,
		})

		if err != nil {
			return ret, err
		}

		for _, reservation := range spotInstanceResults.Reservations {
			for _, instance := range reservation.Instances {
				spotId, ok := spotInstanceMap[*instance.InstanceId]
				if ok {
					ret[spotId] = *instance.State.Name
				}
			}
		}
	}

	return ret, nil
}

func (s *service) DescribeInstanceIPs(ctx context.Context, deployments []models.Deployment) (map[string]string, error) {
	ret := make(map[string]string)

	var instanceids []*string
	var spotInstanceIDs []*string
	for _, deployment := range deployments {
		if deployment.SpotInstance {
			spotInstanceIDs = append(spotInstanceIDs, &deployment.InstanceID)
		} else {
			instanceids = append(instanceids, &deployment.InstanceID)
		}

	}
	ec2Session := ec2.New(s.session)

	if len(instanceids) > 0 {
		//regular instances
		cfg := ec2.DescribeInstancesInput{
			InstanceIds: instanceids,
		}

		results, err := ec2Session.DescribeInstancesWithContext(ctx, &cfg)
		if err != nil {
			if !isNotFound(err) {
				return ret, err
			}
		}

		for _, reservation := range results.Reservations {
			for _, instance := range reservation.Instances {
				ret[*instance.InstanceId] = *instance.PublicIpAddress
			}
		}
	}

	if len(spotInstanceIDs) > 0 {
		//spot instance
		cfgSpot := ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: spotInstanceIDs,
		}

		spotResults, err := ec2Session.DescribeSpotInstanceRequestsWithContext(ctx, &cfgSpot)
		if err != nil {
			if isNotFound(err) {
				return ret, nil
			}
			return ret, err
		}

		// A map for the spotinstance ec2 instances
		spotInstanceMap := make(map[string]string)
		spotInstanceIds := []*string{}

		for _, spotInstanceRequest := range spotResults.SpotInstanceRequests {
			instanceId := (*string)(spotInstanceRequest.InstanceId)
			if instanceId != nil {
				spotInstanceIds = append(spotInstanceIds, instanceId)
				spotId := (*string)(spotInstanceRequest.SpotInstanceRequestId)
				spotInstanceMap[*instanceId] = *spotId
			}
		}

		spotInstanceResults, err := ec2Session.DescribeInstancesWithContext(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: spotInstanceIds,
		})

		if err != nil {
			return ret, err
		}

		for _, reservation := range spotInstanceResults.Reservations {
			for _, instance := range reservation.Instances {
				spotId, ok := spotInstanceMap[*instance.InstanceId]
				if ok {
					ret[spotId] = *instance.PublicIpAddress
				}
			}
		}
	}

	return ret, nil
}
