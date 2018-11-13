package batchlogs

import (
	"context"
	"io"
)

// batchlogs.Service contains functions for streaming logs in real time from a batch job
type Service interface {
	// Stream takes a batch job's log stream name and returns an io.ReadCloser
	// containing the bytes of the log updated in real time
	Stream(ctx context.Context, logName string) io.ReadCloser
}
