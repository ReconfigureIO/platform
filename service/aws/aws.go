package aws

//go:generate mockgen -source=aws.go -package=aws -destination=aws_mock.go

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/abiosoft/errs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
)

// ErrNotFound is not found error.
var ErrNotFound = errors.New("Not Found")

// Service is an AWS service.
type Service interface {
	Upload(key string, r io.Reader, length int64) (string, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	RunBuild(build models.Build, callbackURL string, reportsURL string) (string, error)
	RunGraph(graph models.Graph, callbackURL string) (string, error)
	RunSimulation(inputArtifactURL string, callbackURL string, command string) (string, error)
	HaltJob(batchID string) error
	RunDeployment(command string) (string, error)
	GetJobDetail(id string) (*batch.JobDetail, error)
	DescribeAFIStatus(ctx context.Context, builds []models.Build) (map[string]Status, error)
	GetJobStream(*batch.JobDetail) (*cloudwatchlogs.LogStream, error)
	NewStream(stream cloudwatchlogs.LogStream) *Stream
	Conf() *ServiceConfig
}

type service struct {
	session *session.Session
	conf    ServiceConfig
}

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	LogGroup      string `env:"RECO_AWS_LOG_GROUP" envDefault:"/aws/batch/job"`
	Bucket        string `env:"RECO_AWS_BUCKET" envDefault:"reconfigureio-builds"`
	Queue         string `env:"RECO_AWS_QUEUE" envDefault:"build-jobs"`
	JobDefinition string `env:"RECO_AWS_JOB" envDefault:"sdaccel-builder-build"`
	GenerateAfi   bool   `env:"RECO_FEATURE_DEPLOY"`
}

// New creates a new service with conf.
func New(conf ServiceConfig) Service {
	s := service{conf: conf}
	s.session = session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))
	return &s
}

func (s *service) Upload(key string, r io.Reader, length int64) (string, error) {
	s3Session := s3.New(s.session)

	// s3.PutObjectInput takes in a io.ReadSeeker
	// rather than reading everything into memory
	// let's write it to a temp file instead
	var reader io.ReadSeeker

	// We have multiple lines that are dependent on the
	// previous line returning nil error.
	// Using error group for convenience
	var e errs.Group
	var tmpFile *os.File

	e.Add(func() (err error) {
		tmpFile, err = ioutil.TempFile("", "")
		return
	})
	e.Defer(func() {
		if tmpFile != nil {
			os.Remove(tmpFile.Name())
		}
	})
	e.Add(func() error {
		_, err := io.Copy(tmpFile, r)
		return err
	})
	e.Add(func() (err error) {
		tmpFile.Close()
		tmpFile, err = os.Open(tmpFile.Name())
		return
	})
	e.Add(func() error {
		reader = tmpFile
		return nil
	})
	if err := e.Exec(); err != nil {
		// if writing to temp file fails (which hardly happens)
		// fall back to reading into memory
		// this is bad and buffers the entire body in memory :(
		body := bytes.Buffer{}
		body.ReadFrom(r)
		reader = bytes.NewReader(body.Bytes())
	}

	putParams := &s3.PutObjectInput{
		Bucket:        aws.String(s.conf.Bucket), // Required
		Key:           aws.String(key),           // Required
		Body:          reader,
		ContentLength: aws.Int64(length),
	}

	_, err := s3Session.PutObject(putParams)
	if err != nil {
		return "", err
	}
	return s.s3Url(key), nil
}

func (s *service) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	s3Session := s3.New(s.session)

	getParams := &s3.GetObjectInput{
		Bucket: aws.String(s.conf.Bucket), // Required
		Key:    aws.String(key),           // Required
	}

	object, err := s3Session.GetObjectWithContext(ctx, getParams)
	return object.Body, err
}

func (s *service) s3Url(key string) string {
	return "s3://" + s.conf.Bucket + "/" + key
}

func (s *service) RunBuild(build models.Build, callbackURL string, reportsURL string) (string, error) {
	batchSession := batch.New(s.session)
	inputArtifactURL := s.s3Url(build.InputUrl())
	outputArtifactURL := s.s3Url(build.ArtifactUrl())

	genAfi := "no"
	if s.conf.GenerateAfi {
		genAfi = "yes"
	}

	params := &batch.SubmitJobInput{
		JobDefinition: aws.String(s.conf.JobDefinition), // Required
		JobName:       aws.String("example"),            // Required
		JobQueue:      aws.String(s.conf.Queue),         // Required
		ContainerOverrides: &batch.ContainerOverrides{
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
					Value: aws.String(genAfi),
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

func (s *service) RunSimulation(inputArtifactURL string, callbackURL string, command string) (string, error) {
	batchSession := batch.New(s.session)
	params := &batch.SubmitJobInput{
		JobDefinition: aws.String(s.conf.JobDefinition), // Required
		JobName:       aws.String("example"),            // Required
		JobQueue:      aws.String(s.conf.Queue),         // Required
		ContainerOverrides: &batch.ContainerOverrides{
			Command: []*string{
				aws.String("/opt/simulate.sh"),
			},
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

func (s *service) RunGraph(graph models.Graph, callbackURL string) (string, error) {
	batchSession := batch.New(s.session)
	inputArtifactURL := s.s3Url(graph.InputUrl())
	outputArtifactURL := s.s3Url(graph.ArtifactUrl())

	params := &batch.SubmitJobInput{
		JobDefinition: aws.String(s.conf.JobDefinition), // Required
		JobName:       aws.String("example"),            // Required
		JobQueue:      aws.String(s.conf.Queue),         // Required
		ContainerOverrides: &batch.ContainerOverrides{
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

func (s *service) HaltJob(batchID string) error {
	batchSession := batch.New(s.session)
	params := &batch.TerminateJobInput{
		JobId:  aws.String(batchID),        // Required
		Reason: aws.String("User request"), // Required
	}
	_, err := batchSession.TerminateJob(params)
	return err
}

func (s *service) RunDeployment(command string) (string, error) {

	return "This function does nothing yet", nil
}

func (s *service) GetJobDetail(id string) (*batch.JobDetail, error) {
	batchSession := batch.New(s.session)
	inp := &batch.DescribeJobsInput{Jobs: []*string{&id}}
	resp, err := batchSession.DescribeJobs(inp)
	if err != nil {
		return nil, err
	}
	if len(resp.Jobs) == 0 {
		return nil, ErrNotFound
	}
	return resp.Jobs[0], nil
}

func (s *service) GetJobStream(job *batch.JobDetail) (*cloudwatchlogs.LogStream, error) {
	cwLogs := cloudwatchlogs.New(s.session)

	searchParams := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(s.conf.LogGroup), // Required
		Descending:          aws.Bool(true),
		Limit:               aws.Int64(1),
		LogStreamNamePrefix: job.Container.LogStreamName,
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

func (s *service) Conf() *ServiceConfig {
	return &s.conf
}

// Stream is log stream.
type Stream struct {
	session *service
	stream  cloudwatchlogs.LogStream
	Events  chan *cloudwatchlogs.GetLogEventsOutput
	Ended   bool
}

func (s *service) NewStream(stream cloudwatchlogs.LogStream) *Stream {
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

type Status struct {
	Status    string
	UpdatedAt time.Time
}

func (s *service) DescribeAFIStatus(ctx context.Context, builds []models.Build) (map[string]Status, error) {
	ret := make(map[string]Status)

	var afiids []*string
	for _, build := range builds {
		afiids = append(afiids, &build.FPGAImage)
	}
	ec2Session := ec2.New(s.session)

	cfg := ec2.DescribeFpgaImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("fpga-image-global-id"),
				Values: afiids,
			},
		},
	}

	results, err := ec2Session.DescribeFpgaImagesWithContext(ctx, &cfg)
	if err != nil {
		return ret, err
	}

	for _, image := range results.FpgaImages {
		ret[*image.FpgaImageGlobalId] = Status{*image.State.Code, *image.UpdateTime}
	}

	return ret, nil
}
