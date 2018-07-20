// Package logs implements (*logs.Service).Stream(ctx, logStreamName) io.ReadCloser backed by polling CloudWatchLogs.
package fakebatch

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Service implements Stream(logStreamName string) io.ReadCloser.
// It polls the CloudWatchLogs API at a period defined by defaultPollPeriod.
type Service struct {
	Endpoint string
}

// Stream returns an io.ReadCloser containing the logs for the given
// logStreamName. It is valid to call
// Stream on a logStreamName which does not yet exist, in that case, Stream will
// wait for it to exist, or for the context to be canceled. Context cancelation
// is treated as the end of the stream, causing the ReadCloser to return io.EOF.
func (s *Service) Stream(ctx context.Context, logStreamName string) io.ReadCloser {
	r, w := io.Pipe()

	URL := fmt.Sprintf("%s/v1/logs/%s", s.Endpoint, logStreamName)
	response, err := http.Get(URL)
	if err != nil {
		fmt.Printf("%s", err)
		w.Close()
		return r
	}
	if response.StatusCode != 200 {
		fmt.Printf("Expected status code 200, got %v \n", response.StatusCode)
		w.Close()
		return r
	}
	go watchForContextCancel(ctx, w)
	go copy(response.Body, w)
	return r
}

func watchForContextCancel(ctx context.Context, writer io.WriteCloser) {
	select {
	case <-ctx.Done():
		writer.Close()
	}
}

func copy(body io.ReadCloser, writer io.WriteCloser) {
	defer writer.Close()
	defer body.Close()
	_, err := io.Copy(writer, body)
	if err != nil {
		fmt.Printf("%s", err)
	}
}
