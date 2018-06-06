package s3

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/abiosoft/errs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Service struct {
	Bucket         string
	ConfigProvider client.ConfigProvider

	once sync.Once
	s3   *s3.S3
}

func (s *Service) ensureInit() {
	s.once.Do(func() {
		s.s3 = s3.New(s.ConfigProvider)
	})
}

func (s *Service) Upload(key string, r io.Reader, length int64) (string, error) {
	s.ensureInit()

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
		Bucket:        aws.String(s.Bucket), // Required
		Key:           aws.String(key),      // Required
		Body:          reader,
		ContentLength: aws.Int64(length),
	}

	_, err := s.s3.PutObject(putParams)
	if err != nil {
		return "", err
	}
	return s.s3Url(key), nil
}

func (s *Service) Download(key string) (io.ReadCloser, error) {
	s.ensureInit()

	getParams := &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket), // Required
		Key:    aws.String(key),      // Required
	}

	object, err := s.s3.GetObject(getParams)
	if err != nil {
		return nil, err
	}
	return object.Body, err
}

func (s *Service) s3Url(key string) string {
	return "s3://" + s.Bucket + "/" + key
}
