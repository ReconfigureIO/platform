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
// logStreamName. Under the hood, it polls CloudWatchLogs. It is valid to call
// Stream on a logStreamName which does not yet exist, in that case, Stream will
// wait for it to exist, or for the context to be canceled. Context cancelation
// is treated as the end of the stream, causing the ReadCloser to return io.EOF.
func (s *Service) Stream(ctx context.Context, logStreamName string) io.ReadCloser {
	r, w := io.Pipe()
	defer w.Close()

	URL := fmt.Sprintf("%s/logs/%s", s.Endpoint, logStreamName)
	response, err := http.Get(URL)
	if err != nil {
		fmt.Printf("%s", err)
		return r
	} else {
		defer response.Body.Close()
		_, err = io.Copy(w, response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			return r
		}
	}
	return r
}
