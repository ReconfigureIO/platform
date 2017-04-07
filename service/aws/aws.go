package aws

import (
	"bytes"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"time"
)

var NOT_FOUND = errors.New("Not Found")

type Service struct {
	session *session.Session
	conf    ServiceConfig
}

type ServiceConfig struct {
	Bucket        string
	Queue         string
	JobDefinition string
}

func New(conf ServiceConfig) *Service {
	s := Service{conf: conf}
	s.session = session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))
	return &s
}

func (s *Service) Upload(key string, r io.Reader, length int64) (string, error) {
	s3Session := s3.New(s.session)

	// This is bad and buffers the entire body in memory :(
	body := bytes.Buffer{}
	body.ReadFrom(r)

	putParams := &s3.PutObjectInput{
		Bucket:        aws.String(s.conf.Bucket), // Required
		Key:           aws.String(key),           // Required
		Body:          bytes.NewReader(body.Bytes()),
		ContentLength: aws.Int64(length),
	}

	_, err := s3Session.PutObject(putParams)
	if err != nil {
		return "", err
	}
	return "s3://" + s.conf.Bucket + "/" + key, nil
}

func (s *Service) RunBuild(inputArtifactUrl string) (string, error) {
	batchSession := batch.New(s.session)
	params := &batch.SubmitJobInput{
		JobDefinition: aws.String(s.conf.JobDefinition), // Required
		JobName:       aws.String("example"),            // Required
		JobQueue:      aws.String(s.conf.Queue),         // Required
		ContainerOverrides: &batch.ContainerOverrides{
			Environment: []*batch.KeyValuePair{
				{
					Name:  aws.String("PART"),
					Value: aws.String("xcvu9p-flgb2104-2-i-es2"),
				},
				{
					Name:  aws.String("PART_FAMILY"),
					Value: aws.String("virtexuplus"),
				},
				{
					Name:  aws.String("INPUT_URL"),
					Value: aws.String(inputArtifactUrl),
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

func (s *Service) GetJobDetail(id string) (*batch.JobDetail, error) {
	batchSession := batch.New(s.session)
	inp := &batch.DescribeJobsInput{Jobs: []*string{&id}}
	resp, err := batchSession.DescribeJobs(inp)
	if err != nil {
		return nil, err
	}
	if len(resp.Jobs) == 0 {
		return nil, NOT_FOUND
	}
	return resp.Jobs[0], nil
}

func (s *Service) GetJobStream(id string) (*cloudwatchlogs.LogStream, error) {
	cwLogs := cloudwatchlogs.New(s.session)

	searchParams := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String("/aws/batch/job"), // Required
		Descending:          aws.Bool(true),
		Limit:               aws.Int64(1),
		LogStreamNamePrefix: aws.String("example/" + id),
	}
	resp, err := cwLogs.DescribeLogStreams(searchParams)
	if err != nil {
		return nil, err
	}

	if len(resp.LogStreams) == 0 {
		return nil, NOT_FOUND
	}
	return resp.LogStreams[0], nil
}

type Stream struct {
	session *Service
	stream  cloudwatchlogs.LogStream
	Events  chan *cloudwatchlogs.GetLogEventsOutput
	stop    chan struct{}
	ended   bool
}

func (s *Service) NewStream(stream cloudwatchlogs.LogStream) *Stream {
	logs := make(chan *cloudwatchlogs.GetLogEventsOutput)
	stop := make(chan struct{}, 1)

	ret := Stream{s, stream, logs, stop, false}
	return &ret
}

func (stream *Stream) Stop() {
	stream.stop <- struct{}{}
}

func (stream *Stream) Run() error {
	cwLogs := cloudwatchlogs.New(stream.session.session)

	params := (&cloudwatchlogs.GetLogEventsInput{}).
		SetLogGroupName("/aws/batch/job").
		SetLogStreamName(*stream.stream.LogStreamName).
		SetStartFromHead(true)

	defer func() {
		close(stream.Events)
	}()
	err := cwLogs.GetLogEventsPages(params, func(page *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
		select {
		case stream.Events <- page:
			if lastPage || (len(page.Events) == 0 && stream.ended) {
				return false
			}
			if len(page.Events) == 0 {
				time.Sleep(10 * time.Second)
			}
			return true
		case <-stream.stop:
			return false
		}
	})
	return err
}