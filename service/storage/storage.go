package storage

//go:generate mockgen -source=storage.go -package=storage -destination=storage_mock.go

import "io"

// A Service provides a content store.
// It is implemented by service/storage/s3.Service.
type Service interface {
	Upload(key string, r io.Reader) (string, error)
	Download(key string) (io.ReadCloser, error)
}
