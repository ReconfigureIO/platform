// Package fakebatchlogs implements Stream(...) backed by fakebatch.
package fakebatchlogs

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Service implements Stream()
type Service struct {
	Endpoint string
}

// Stream returns an io.ReadCloser containing the logs for the given
// logStreamName. It is valid to call Stream on a logStreamName which does not
// yet exist, in that case, Stream will wait for it to exist, or for the context
// to be canceled. Context cancelation is treated as the end of the stream,
// causing the ReadCloser to return io.EOF.
func (s *Service) Stream(
	ctx context.Context, logStreamName string,
) io.ReadCloser {
	url := fmt.Sprintf("%s/v1/logs/%s", s.Endpoint, logStreamName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		err := fmt.Errorf("fakebatchlogs.Stream: http.NewRequest: %v", err)
		return errReader(err)
	}

	req = req.WithContext(ctx)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return errReader(fmt.Errorf("fakebatchlogs.Stream: client.Do: %v", err))
	}
	if !(200 <= resp.StatusCode && resp.StatusCode <= 299) {
		err := fmt.Errorf("non-2xx status: %v %v", resp.StatusCode, resp.Status)
		return errReader(fmt.Errorf("fakebatchlogs.Stream: client.Do: %v", err))
	}

	return resp.Body
}

func errReader(err error) io.ReadCloser {
	r, w := io.Pipe()
	_ = w.CloseWithError(err)
	return r
}
