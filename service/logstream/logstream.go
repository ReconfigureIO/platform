package logstream

import (
	"context"
	"io"

	"github.com/ReconfigureIO/platform/models"
)

// StreamService contains functions for streaming logs from a batch job
type StreamService interface {
	Stream(ctx context.Context, batchJob models.BatchJob) io.ReadCloser
}
