package logstream

import (
	"context"
	"io"
)

// StreamService contains functions for streaming logs from a batch job
type StreamService interface {
	Stream(ctx context.Context, logName string) io.ReadCloser
}
