package aws

import (
	"context"
	"errors"
	"time"

	"github.com/ReconfigureIO/platform/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// ErrNotFound is not found error.
var ErrNotFound = errors.New("Not Found")

type Service struct {
	session *session.Session
	conf    ServiceConfig
}

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	LogGroup      string `env:"RECO_AWS_LOG_GROUP" envDefault:"/aws/batch/job"`
	Bucket        string `env:"RECO_AWS_BUCKET" envDefault:"reconfigureio-builds"`
	Queue         string `env:"RECO_AWS_QUEUE" envDefault:"build-jobs"`
	JobDefinition string `env:"RECO_AWS_JOB" envDefault:"sdaccel-builder-build"`
	EndPoint      string `env:"RECO_AWS_ENDPOINT" envDefault:""` // AWS SDK uses endpoint generated from region when this value is an empty string
}

// New creates a new service with conf.
func New(conf ServiceConfig) *Service {
	s := Service{conf: conf}
	s.session = session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1").WithEndpoint(conf.EndPoint)))
	return &s
}

func (s *Service) s3Url(key string) string {
	return "s3://" + s.conf.Bucket + "/" + key
}

// RunBuild creates an AWS Batch Job that runs our build process
func (s *Service) RunBuild(build models.Build, callbackURL string, reportsURL string) (string, error) {
	batchSession := batch.New(s.session)
	inputArtifactURL := s.s3Url(build.InputUrl())
	debugArtifactURL := s.s3Url(build.DebugUrl())
	outputArtifactURL := s.s3Url(build.ArtifactUrl())
	memory := int64(32000)

	params := &batch.SubmitJobInput{
		JobDefinition: aws.String(s.conf.JobDefinition), // Required
		JobName:       aws.String("example"),            // Required
		JobQueue:      aws.String(s.conf.Queue),         // Required
		ContainerOverrides: &batch.ContainerOverrides{
			Memory: &memory,
			Environment: []*batch.KeyValuePair{
				{
					Name:  aws.String("PART"),
					Value: aws.String("xcvu9p-flgb2104-2-i"),
				},
				{
					Name:  aws.String("PART_FAMILY"),
					Value: aws.String("virtexuplus"),
				},
				{
					Name:  aws.String("INPUT_URL"),
					Value: aws.String(inputArtifactURL),
				},
				{
					Name:  aws.String("CALLBACK_URL"),
					Value: aws.String(callbackURL),
				},
				{
					Name:  aws.String("DEBUG_URL"),
					Value: aws.String(debugArtifactURL),
				},
				{
					Name:  aws.String("DEVICE"),
					Value: aws.String("xilinx_aws-vu9p-f1_4ddr-xpr-2pr_4_0"),
				},
				{
					Name:  aws.String("DEVICE_FULL"),
					Value: aws.String("xilinx:aws-vu9p-f1:4ddr-xpr-2pr:4.0"),
				},
				{
					Name:  aws.String("OUTPUT_URL"),
					Value: aws.String(outputArtifactURL),
				},
				{
					Name:  aws.String("REPORT_URL"),
					Value: aws.String(reportsURL),
				},
				{
					Name:  aws.String("DCP_KEY"),
					Value: aws.String("/dcp/" + build.ID),
				},
				{
					Name:  aws.String("LOG_KEY"),
					Value: aws.String("/dcp-logs/" + build.ID),
				},
				{
					Name:  aws.String("GENERATE_AFI"),
					Value: aws.String("yes"),
				},
			},
		},
	}
	resp, err := batchSession.SubmitJob(params)
	if err != nil {
		return "", err
	}
	return *resp.JobId, nil
}

// RunSimulation creates an AWS Batch Job that runs our simulation process
func (s *Service) RunSimulation(inputArtifactURL string, callbackURL string, command string) (string, error) {
	batchSession := batch.New(s.session)
	params := &batch.SubmitJobInput{
		JobDefinition: aws.String(s.conf.JobDefinition), // Required
		JobName:       aws.String("example"),            // Required
		JobQueue:      aws.String(s.conf.Queue),         // Required
		ContainerOverrides: &batch.ContainerOverrides{
			Command: aws.StringSlice([]string{"/opt/simulate.sh"}),
			Environment: []*batch.KeyValuePair{
				{
					Name:  aws.String("PART"),
					Value: aws.String("xcvu9p-flgb2104-2-i"),
				},
				{
					Name:  aws.String("PART_FAMILY"),
					Value: aws.String("virtexuplus"),
				},
				{
					Name:  aws.String("INPUT_URL"),
					Value: aws.String(inputArtifactURL),
				},
				{
					Name:  aws.String("CALLBACK_URL"),
					Value: aws.String(callbackURL),
				},
				{
					Name:  aws.String("CMD"),
					Value: aws.String(command),
				},
				{
					Name:  aws.String("DEVICE"),
					Value: aws.String("xilinx_aws-vu9p-f1_4ddr-xpr-2pr_4_0"),
				},
				{
					Name:  aws.String("DEVICE_FULL"),
					Value: aws.String("xilinx:aws-vu9p-f1:4ddr-xpr-2pr:4.0"),
				},
			},
		},
	}
	resp, err := batchSession.SubmitJob(params)
	if err != nil {
		return "", err
	}
	return *resp.JobId, nil
}

// RunGraph creates an AWS Batch Job that runs our graph process
func (s *Service) RunGraph(graph models.Graph, callbackURL string) (string, error) {
	batchSession := batch.New(s.session)
	inputArtifactURL := s.s3Url(graph.InputUrl())
	outputArtifactURL := s.s3Url(graph.ArtifactUrl())

	params := &batch.SubmitJobInput{
		JobDefinition: aws.String(s.conf.JobDefinition), // Required
		JobName:       aws.String("example"),            // Required
		JobQueue:      aws.String(s.conf.Queue),         // Required
		ContainerOverrides: &batch.ContainerOverrides{
			Command: aws.StringSlice([]string{"/opt/graph.sh"}),
			Environment: []*batch.KeyValuePair{
				{
					Name:  aws.String("INPUT_URL"),
					Value: aws.String(inputArtifactURL),
				},
				{
					Name:  aws.String("CALLBACK_URL"),
					Value: aws.String(callbackURL),
				},
				{
					Name:  aws.String("OUTPUT_URL"),
					Value: aws.String(outputArtifactURL),
				},
			},
		},
	}
	resp, err := batchSession.SubmitJob(params)
	if err != nil {
		return "", err
	}
	return *resp.JobId, nil
}

// HaltJob terminates a running batch job
func (s *Service) HaltJob(batchID string) error {
	batchSession := batch.New(s.session)
	params := &batch.TerminateJobInput{
		JobId:  aws.String(batchID),        // Required
		Reason: aws.String("User request"), // Required
	}
	_, err := batchSession.TerminateJob(params)
	return err
}

// RunDeployment is not implemented
func (s *Service) RunDeployment(command string) (string, error) {
	return "This function does nothing yet", nil
}

// GetJobDetail describes the AWS Batch job.
func (s *Service) GetJobDetail(id string) (*batch.JobDetail, error) {
	batchSession := batch.New(s.session)
	inp := &batch.DescribeJobsInput{
		Jobs: aws.StringSlice([]string{id}),
	}
	resp, err := batchSession.DescribeJobs(inp)
	if err != nil {
		return nil, err
	}
	if len(resp.Jobs) == 0 {
		return nil, ErrNotFound
	}
	return resp.Jobs[0], nil
}

// GetJobStream takes a cloudwatch logstream name and returns the actual logstream
func (s *Service) GetJobStream(logStreamName string) (*cloudwatchlogs.LogStream, error) {
	cwLogs := cloudwatchlogs.New(s.session)

	searchParams := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(s.conf.LogGroup), // Required
		Descending:          aws.Bool(true),
		Limit:               aws.Int64(1),
		LogStreamNamePrefix: aws.String(logStreamName),
	}
	resp, err := cwLogs.DescribeLogStreams(searchParams)
	if err != nil {
		return nil, err
	}

	if len(resp.LogStreams) == 0 {
		return nil, ErrNotFound
	}
	return resp.LogStreams[0], nil
}

// Conf is used to retrieve the service's config
func (s *Service) Conf() *ServiceConfig {
	return &s.conf
}

// Stream is log stream.
type Stream struct {
	session *Service
	stream  cloudwatchlogs.LogStream
	Events  chan *cloudwatchlogs.GetLogEventsOutput
	Ended   bool
}

// NewStream TODO campgareth: write proper comment
func (s *Service) NewStream(stream cloudwatchlogs.LogStream) *Stream {
	logs := make(chan *cloudwatchlogs.GetLogEventsOutput)

	return &Stream{
		session: s,
		stream:  stream,
		Events:  logs,
	}
}

// Run starts the stream using context.
func (stream *Stream) Run(ctx context.Context, logGroup string) error {
	cwLogs := cloudwatchlogs.New(stream.session.session)

	params := (&cloudwatchlogs.GetLogEventsInput{}).
		SetLogGroupName(logGroup).
		SetLogStreamName(*stream.stream.LogStreamName).
		SetStartFromHead(true)

	defer func() {
		close(stream.Events)
	}()
	err := cwLogs.GetLogEventsPagesWithContext(ctx, params, func(page *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
		select {
		case <-ctx.Done():
			return false
		case stream.Events <- page:
			if lastPage || (len(page.Events) == 0 && stream.Ended) {
				return false
			}
			if len(page.Events) == 0 {
				time.Sleep(10 * time.Second)
			}
			return true
		}
	})
	return err
}
