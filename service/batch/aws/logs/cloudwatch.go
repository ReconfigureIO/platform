// Package logs implements (*logs.Service).Stream(ctx, logStreamName) io.ReadCloser backed by polling CloudWatchLogs.
package logs

import (
	"bytes"
	"context"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

// Service implements Stream(logStreamName string) io.ReadCloser.
// It polls the CloudWatchLogs API at a period defined by defaultPollPeriod.
type Service struct {
	CloudWatchLogsAPI cloudwatchlogsiface.CloudWatchLogsAPI
	LogGroup          string

	_pollPeriod time.Duration // for tests only.
}

const defaultPollPeriod = 10 * time.Second

func (s *Service) pollPeriod() time.Duration {
	if s._pollPeriod == time.Duration(0) {
		return defaultPollPeriod
	}
	return s._pollPeriod
}

// Stream returns an io.ReadCloser containing the logs for the given
// logStreamName. Under the hood, it polls CloudWatchLogs. It is valid to call
// Stream on a logStreamName which does not yet exist, in that case, Stream will
// wait for it to exist, or for the context to be canceled. Context cancelation
// is treated as the end of the stream, causing the ReadCloser to return io.EOF.
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
		SetLogGroupName(s.LogGroup).
		SetLogStreamName(logStreamName).
		SetStartFromHead(true)

	err := getLogEvents(
		ctx,
		s.CloudWatchLogsAPI,
		req,
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

	if isContextCancelation(err) {
		// Treat context cancelation as the end of the stream.
		err = io.EOF
	}

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

// getLogEvents calls cw.GetLogEvents, except that instead of returning
// ResourceNotFound in case of a missing stream, it simply returns an empty
// page.
func getLogEvents(
	ctx context.Context,
	cw cloudwatchlogsiface.CloudWatchLogsAPI,
	input *cloudwatchlogs.GetLogEventsInput,
	fn func(resp *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool,
) error {
	var emptyPage cloudwatchlogs.GetLogEventsOutput

again:
	err := cw.GetLogEventsPagesWithContext(ctx, input, fn)

	if isResourceNotFound(err) {
		// Call fn with an empty page so that it can handle poll logic.
		if keepGoing := fn(&emptyPage, false); keepGoing {
			goto again
		}
		err = nil // Suppress not found error.
	}

	return err
}

func isResourceNotFound(err error) bool {
	aerr, ok := err.(awserr.Error)
	if !ok {
		return false
	}

	const notFound = cloudwatchlogs.ErrCodeResourceNotFoundException
	return aerr.Code() == notFound
}

func isContextCancelation(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasSuffix(err.Error(), context.DeadlineExceeded.Error()) ||
		strings.HasSuffix(err.Error(), context.Canceled.Error())
}
