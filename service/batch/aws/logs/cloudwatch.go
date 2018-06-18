// Package logs implements (*logs.Service).Stream(ctx, logStreamName) io.ReadCloser backed by polling CloudWatchLogs.
package logs

import (
	"bytes"
	"context"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

// Service implements Stream(logStreamName string) io.ReadCloser.
// It polls the CloudWatchLogs API at a period defined by defaultPollPeriod.
type Service struct {
	cw          cloudwatchlogsiface.CloudWatchLogsAPI
	_pollPeriod time.Duration // for tests only.
	logGroup    string
}

const defaultPollPeriod = 10 * time.Second

func (s *Service) pollPeriod() time.Duration {
	if s._pollPeriod == time.Duration(0) {
		return defaultPollPeriod
	}
	return s._pollPeriod
}

// Stream returns an io.ReadCloser containing the logs for the given logStreamName.
// Under the hood, it polls CloudWatchLogs.
func (s *Service) Stream(ctx context.Context, logStreamName string) io.ReadCloser {
	r, w := io.Pipe()
	go s.pollCloudWatch(ctx, w, logStreamName)
	return r
}

func (s *Service) pollCloudWatch(ctx context.Context, w *io.PipeWriter, logStreamName string) {
	pollTimer := time.NewTimer(1 * time.Hour)
	if !pollTimer.Stop() {
		// Unlikely, due to the 1h duration chosen above but correct in spirit.
		<-pollTimer.C
	}

	var (
		scratchBuf bytes.Buffer
		err2       error // For tracking write errors.
	)

	req := (&cloudwatchlogs.GetLogEventsInput{}).
		SetLogGroupName(s.logGroup).
		SetLogStreamName(logStreamName).
		SetStartFromHead(true)

	err := s.cw.GetLogEventsPagesWithContext(ctx, req,
		func(resp *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
			err2 = writeEvents(&scratchBuf, w, resp)
			if err2 != nil {
				return false // Stop.
			}

			// Start the timer now.
			pollTimer.Reset(s.pollPeriod())

			select {
			case <-ctx.Done(): // Cancelled.
				return false // Stop.
			case <-pollTimer.C: // Wait on timer.
			}

			return true // Continue.
		})

	if err == nil {
		err = err2
	}

	err = w.CloseWithError(err)
	if err != nil {
		log.Printf("aws/logs/Service.Stream: w.CloseWithError: %v", err)
	}
}

func writeEvents(
	scratchBuf *bytes.Buffer,
	w io.Writer,
	resp *cloudwatchlogs.GetLogEventsOutput,
) error {
	for _, ev := range resp.Events {
		scratchBuf.Reset()
		scratchBuf.WriteString(*ev.Message)
		scratchBuf.WriteRune('\n')

		_, err := io.Copy(w, scratchBuf)
		if err != nil {
			return err
		}
	}

	return nil
}
