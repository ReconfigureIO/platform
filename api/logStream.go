package api

import (
	"log"
	"time"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/stream"
	"github.com/gin-gonic/gin"
)

type HasBatch interface {
	GetBatchId() string
	Status() string
	HasStarted() bool
	HasFinished() bool
}

func StreamBatchLogs(awsSession *aws.Service, c *gin.Context, b HasBatch, refresh func() error) {
	err := refresh()
	if err != nil {
		internalError(c, err)
		return
	}

	w := c.Writer
	clientGone := w.CloseNotify()

	for !b.HasStarted() {
		select {
		case <-clientGone:
			return
		default:
			time.Sleep(time.Second)
			err = refresh()
			if err != nil {
				internalError(c, err)
				return
			}
		}
	}

	logStream, err := awsSession.GetJobStream(b.GetBatchId())
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	log.Printf("opening log stream: %s", *logStream.LogStreamName)

	lstream := awsSession.NewStream(*logStream)

	go func() {
		for !b.HasFinished() {
			time.Sleep(10 * time.Second)
			err := refresh()
			if err != nil {
				break
			}
		}
		lstream.Ended = true
	}()

	stream.Stream(lstream, c)

}
