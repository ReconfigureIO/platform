package cloudwatch

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/fortytw2/leaktest"
)

func TestCloudWatchStream(t *testing.T) {
	// TestCloudWatchStream tests that the stream returns each message in a
	// separate read call, and that we don't probe CloudWatch for pages more
	// frequently than allowed by the poll period.

	// Each string here is sent as a message in a separate page in CloudWatch.
	pagesSimple := [][]string{{"foo"}, {"bar"}, {"baz"}, {"qux"}}
	t.Run("pagesSimple", testCasePages(pagesSimple).run)

	pagesEmpty := [][]string{{}, {}, {}, {"hello", "world"}, {}}
	t.Run("pagesEmpty", testCasePages(pagesEmpty).run)

	pagesMultiLine := [][]string{{"hello", "world"}, {}, {"foo", "bar", "baz"}}
	t.Run("pagesMultiLine", testCasePages(pagesMultiLine).run)
}

type testCasePages [][]string

func (testCase testCasePages) run(t *testing.T) {
	// Leaky goroutines check.
	defer leaktest.Check(t)()

	s := Service{
		CloudWatchLogsAPI: &fakeCloudWatchLogsPages{
			pageToOutputLogEvents: stringsToPageToOutputLogEvents([][]string(testCase)),
		},
		_pollPeriod: 250 * time.Microsecond,
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rc := s.Stream(ctx, "testLogStreamName")
	defer func() {
		err2 := rc.Close()
		if err2 != nil {
			t.Fatal(err2)
		}
	}()

	// Do a series of reads, and assert that we see each line printed into the
	// log separately, with a delay of at least the poll period.

	// T=Zero, a long time ago. First wait duration will be huge, and we make
	// use of this..
	var lastMsg time.Time

	var buf [64]byte
	for i, expectedLines := range testCase {
		// Check that we get the expected content with each Read().
		expected := linesToString(expectedLines)
		if len(expected) == 0 {
			continue // nothing to read, rc.Read will not block.
		}

		nextStart := time.Now()
		n, err := io.ReadFull(rc, buf[:len(expected)])
		if string(buf[:n]) != expected {
			t.Errorf(
				"Read did not return as expected: "+
					"n, err, buf = %d, %v, %q (expected %q)",
				n, err, string(buf[:n]), expected,
			)
		}

		// Timing check: is the poll period working? We shouldn't get here any
		// faster than the poll period.
		waitDuration := time.Since(lastMsg)
		if waitDuration < s._pollPeriod {
			t.Fatalf("CloudWatchStream not rate limited, "+
				"only waited %v for case %d",
				waitDuration, i)
		}
		lastMsg = nextStart
	}

	// Check that we get EOF.
	n, err := rc.Read(buf[:])
	if n != 0 || err != io.EOF {
		t.Errorf("Expected io.EOF, got: n, err := %d, %v", n, err)
	}
}

func TestCloudWatchError(t *testing.T) {
	// TestCloudWatchError tests the behaviour of the CloudWatchLogs service
	// when CloudWatch returns an error.

	// Leaky goroutines check.
	defer leaktest.Check(t)()

	errTest := errors.New("errTest")

	s := Service{
		CloudWatchLogsAPI: &fakeCloudWatchLogsError{
			// GetLogEventsPagesWithContext returns errTest.
			err: errTest,
		},
		_pollPeriod: 1 * time.Microsecond,
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rc := s.Stream(ctx, "testLogStreamName")

	_, err := io.Copy(ioutil.Discard, rc)
	if err != errTest {
		t.Fatalf("err != errTest: (err is %v)", err)
	}

	err = rc.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloudWatchTermination(t *testing.T) {
	// TestCloudWatchTermination tests that the Stream function comes to
	// an end if the context is cancelled.

	// Leaky goroutines check.
	defer leaktest.Check(t)()

	s := Service{
		// Infinite stream of empty pages.
		CloudWatchLogsAPI: &fakeCloudWatchLogInfinite{},
		// Poll at an extremely high frequency.
		_pollPeriod: 1 * time.Microsecond,
	}

	// Set up a context which will time out in the very near future.
	ctx := context.Background()
	const timeout = 5000 * time.Microsecond
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rc := s.Stream(ctx, "testLogStreamName")

	_, err := io.Copy(ioutil.Discard, rc)
	if err != nil {
		t.Fatalf("io.Copy: %v ", err)
	}

	err = rc.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// fakeCloudWatchLogsPages implements GetLogEventsPagesWithContext(), returning a
// page per log message.
type fakeCloudWatchLogsPages struct {
	// Embedded so that it satisfies the interface.
	cloudwatchlogsiface.CloudWatchLogsAPI

	// List of pages to return.
	pageToOutputLogEvents [][]*cloudwatchlogs.OutputLogEvent
}

func (cw *fakeCloudWatchLogsPages) GetLogEventsPagesWithContext(
	ctx aws.Context,
	req *cloudwatchlogs.GetLogEventsInput,
	fn func(*cloudwatchlogs.GetLogEventsOutput, bool) bool,
	opts ...request.Option,
) error {
	for _, outputLogEvents := range cw.pageToOutputLogEvents {
		keepGoing := fn(
			&cloudwatchlogs.GetLogEventsOutput{
				Events: outputLogEvents,
			},
			false,
		)
		if !keepGoing {
			break
		}
	}
	return nil
}

// fakeCloudWatchLogsError implements GetLogEventsPagesWithContext(), returning only
// the given err.
type fakeCloudWatchLogsError struct {
	// Embedded so that it satisfies the interface.
	cloudwatchlogsiface.CloudWatchLogsAPI

	err error
}

func (cw *fakeCloudWatchLogsError) GetLogEventsPagesWithContext(
	ctx aws.Context,
	req *cloudwatchlogs.GetLogEventsInput,
	fn func(*cloudwatchlogs.GetLogEventsOutput, bool) bool,
	opts ...request.Option,
) error {
	return cw.err
}

// fakeCloudWatchLogInfinite implements GetLogEventsPagesWithContext(),
// returning empty pages forever.
type fakeCloudWatchLogInfinite struct {
	// Embedded so that it satisfies the interface.
	cloudwatchlogsiface.CloudWatchLogsAPI
}

func (cw *fakeCloudWatchLogInfinite) GetLogEventsPagesWithContext(
	ctx aws.Context,
	req *cloudwatchlogs.GetLogEventsInput,
	fn func(*cloudwatchlogs.GetLogEventsOutput, bool) bool,
	opts ...request.Option,
) error {
	for {
		keepGoing := fn(
			&cloudwatchlogs.GetLogEventsOutput{},
			false,
		)
		if !keepGoing {
			break
		}
	}
	return nil
}

func stringsToPageToOutputLogEvents(
	strings [][]string,
) (pageToOutputLogEvents [][]*cloudwatchlogs.OutputLogEvent) {
	for _, lines := range strings {
		var outputLogEvents []*cloudwatchlogs.OutputLogEvent
		for _, line := range lines {
			outputLogEvents = append(
				outputLogEvents,
				&cloudwatchlogs.OutputLogEvent{
					Message: aws.String(line),
				},
			)
		}

		pageToOutputLogEvents = append(
			pageToOutputLogEvents, outputLogEvents)
	}
	return pageToOutputLogEvents
}

// linesToString puts a newline at the end of each line, and joins them together
// as one string. If the input is empty, the output is empty.
func linesToString(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestGetLogEvents(t *testing.T) {
	// TestGetLogEvents checks that getLogEvents() behaves correctly in the case
	// that the log stream doesn't exist until some time after streaming has
	// started.

	cw := &fakeCloudWatchLogsDelayedLogStreamExistence{
		nCallsUntilExisting: 5,
		onceExisting: &fakeCloudWatchLogsPages{
			pageToOutputLogEvents: stringsToPageToOutputLogEvents(
				[][]string{{"Hello"}, {"World"}},
			),
		},
	}
	ctx := context.Background()

	var gotPages [][]string
	err := getLogEvents(
		ctx,
		cw,
		&cloudwatchlogs.GetLogEventsInput{},
		func(resp *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
			var page []string
			for _, ev := range resp.Events {
				page = append(page, *ev.Message)
			}
			gotPages = append(gotPages, page)
			return true
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	expectedPages := [][]string{
		// Number of nils (empty pages) corresponding to nCallsUntilExisting.
		nil, nil, nil, nil, nil,
		// Output corresponding to onceExisting.
		{"Hello"}, {"World"},
	}

	if !reflect.DeepEqual(expectedPages, gotPages) {
		t.Fatalf("expectedPages != gotPages (%q != %q)", expectedPages, gotPages)
	}
}

// fakeCloudWatchLogsDelayedLogStreamExistence returns a ResourceNotFound
// exception for the first nCallsUntilExisting calls to GetLogEventsPagesWithContext.
// Then it fowards calls to the underlying `onceExisting` API.
type fakeCloudWatchLogsDelayedLogStreamExistence struct {
	cloudwatchlogsiface.CloudWatchLogsAPI

	onceExisting                cloudwatchlogsiface.CloudWatchLogsAPI
	nCalls, nCallsUntilExisting int
}

func (
	cw *fakeCloudWatchLogsDelayedLogStreamExistence,
) GetLogEventsPagesWithContext(
	ctx aws.Context,
	req *cloudwatchlogs.GetLogEventsInput,
	fn func(*cloudwatchlogs.GetLogEventsOutput, bool) bool,
	opts ...request.Option,
) error {
	defer func() { cw.nCalls++ }()

	// First nCallsUntilExisting pages don't exist.
	if cw.nCalls < cw.nCallsUntilExisting {
		return awserr.New(
			cloudwatchlogs.ErrCodeResourceNotFoundException,
			"test error",
			nil,
		)
	}

	return cw.onceExisting.
		GetLogEventsPagesWithContext(ctx, req, fn, opts...)
}
