package s3

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
)

type Service struct {
	Bucket string

	S3API       s3iface.S3API
	UploaderAPI s3manageriface.UploaderAPI
}

func (s *Service) Upload(key string, r io.Reader) (string, error) {
	_, err := s.UploaderAPI.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Body:   r,
	})

	return s.s3Url(key), err
}

func (s *Service) Download(key string) (io.ReadCloser, error) {
	object, err := s.S3API.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return object.Body, err
}

func (s *Service) s3Url(key string) string {
	return "s3://" + s.Bucket + "/" + key
}
