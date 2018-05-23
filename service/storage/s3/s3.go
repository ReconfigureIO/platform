package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/ReconfigureIO/platform/service/storage"
	"github.com/abiosoft/errs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// ErrNotFound is not found error.
var ErrNotFound = errors.New("Not Found")

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
}

// New creates a new service with conf.
func New(conf ServiceConfig) storage.Service {
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

func (s *service) Download(ctx context.Context, key string) ([]byte, error) {
	s3Session := s3.New(s.session)

	getParams := &s3.GetObjectInput{
		Bucket: aws.String(s.conf.Bucket), // Required
		Key:    aws.String(key),           // Required
	}

	object, err := s3Session.GetObjectWithContext(ctx, getParams)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(object.Body)
	object.Body.Close()
	return data, err
}

func (s *service) s3Url(key string) string {
	return "s3://" + s.conf.Bucket + "/" + key
}

func (s *service) Conf() *ServiceConfig {
	return &s.conf
}
