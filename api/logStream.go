package api

import (
	"bytes"
	"context"
	"io"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/stream"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
)

func StreamBatchLogs(awsSession *aws.Service, c *gin.Context, b *models.BatchJob) {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	w := c.Writer

	// set necessary headers to inform client of streaming connection
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// cancel whenever we get a close
	go func() {
		<-w.CloseNotify()
		cancel()
	}()

	refresh := func() error {
		return db.Model(&b).Association("Events").Find(&b.Events).Error
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	refreshTicker := time.NewTicker(10 * time.Second)
	defer refreshTicker.Stop()

	stream.StreamWithContext(ctx, c, func(ctx context.Context, w io.Writer) bool {
		if b.HasStarted() {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			bytes.NewBuffer([]byte{0}).WriteTo(w)
		case <-refreshTicker.C:
			err := refresh()
			if err != nil {
				InternalError(c, err)
				return false
			}
		}
		return true
	})

	logStream, err := awsSession.GetJobStream(b.BatchId)
	if err != nil {
		ErrResponse(c, 500, err)
		return
	}

	log.Printf("opening log stream: %s", *logStream.LogStreamName)

	lstream := awsSession.NewStream(*logStream)

	go func() {
		for !b.HasFinished() {
			select {
			case <-ctx.Done():
				return
			case <-refreshTicker.C:
				err := refresh()
				if err != nil {
					break
				}
			}
		}
		lstream.Ended = true
	}()

	stream.Stream(lstream, c, ctx)

}
