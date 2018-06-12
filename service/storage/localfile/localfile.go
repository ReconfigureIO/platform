package localfile

import (
	"io"
	"os"
	"path/filepath"
)

// Service represents the directory which localfile.Service uses for storage.
type Service string

// Upload writes the contents of `r`` to a file with the given key name.
func (s Service) Upload(key string, r io.Reader) (s3url string, err error) {
	err = os.MkdirAll(string(s), 0777)
	if err != nil {
		return "", err
	}

	fd, err := os.Create(filepath.Join(string(s), key))
	if err != nil {
		return "", err
	}
	defer func() {
		closeErr := fd.Close()
		if err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(fd, r)
	if err != nil {
		return "", err
	}

	return "NOTIMPLEMENTED", nil
}

// Download returns a reader to the contents of the filename `key`.
func (s Service) Download(key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(string(s), key))
}
