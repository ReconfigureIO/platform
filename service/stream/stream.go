package stream

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/gin-gonic/gin"
)

// StartWithContext starts a stream using a context.
func StartWithContext(ctx context.Context, c *gin.Context, step func(ctx context.Context, w io.Writer) bool) {
	for {
		keepGoing := step(ctx, c.Writer)
		c.Writer.Flush()
		if !keepGoing {
			return
		}
	}
}

// Start starts a stream of cloudwatch log events, and stream the messages to
// the client until it finishes.
func Start(ctx context.Context, stream *aws.Stream, c *gin.Context) {
	go func() {
		err := stream.Run(ctx)
		if err != nil {
			c.Error(err)
		}
	}()

	StartWithContext(ctx, c, func(ctx context.Context, w io.Writer) bool {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		select {
		case <-ctx.Done():
			return false
		case log, ok := <-stream.Events:
			if ok {
				for _, e := range log.Events {
					_, err := bytes.NewBufferString((*e.Message) + "\n").WriteTo(w)
					if err != nil {
						c.Error(err)
						return false
					}
				}
			}
			return ok
		case <-ticker.C:
			bytes.NewBuffer([]byte{0}).WriteTo(w)
			return true
		}
	})
}
