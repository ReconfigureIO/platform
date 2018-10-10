// Package logs implements (*logs.Service).Stream(ctx, logStreamName) io.ReadCloser backed by polling CloudWatchLogs.
package fakebatchlogs

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
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		_ = w.CloseWithError(err)
		return r
	}
	req = req.WithContext(ctx)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to run client.Do: %v \n", err)
		_ = w.CloseWithError(err)
		return r
	}

	if response.StatusCode != 200 {
		fmt.Printf("Expected status code 200, got %v \n", response.StatusCode)
		_ = w.CloseWithError(err)
		return r
	}
	go func() {
		defer response.Body.Close()
		fmt.Println("starting io.Copy in Stream function")
		_, err = io.Copy(w, response.Body)
		if err != nil {
			fmt.Printf("Error on io.Copy in Stream function: %v \n", err)
			w.CloseWithError(err)
		}
		w.Close()
	}()

	return r
}
