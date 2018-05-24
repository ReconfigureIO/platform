package storage

import "io"

// Interface for anything storagey
type Service interface {
	Upload(key string, r io.Reader, length int64) (string, error)
	Download(key string) (io.ReadCloser, error)
}
