package storage

import "io"

// A Service provides a content store.
// It is implemented by service/storage/s3.Service.
type Service interface {
	Upload(key string, r io.Reader) (string, error)
	Download(key string) (io.ReadCloser, error)
}
