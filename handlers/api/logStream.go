package api

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/stream"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// StreamBatchLogs streams batch logs from AWS.
func StreamBatchLogs(awsSession aws.Service, c *gin.Context, b *models.BatchJob) {
	ctx, cancel := WithClose(c)
	defer cancel()

	w := c.Writer

	// set necessary headers to inform client of streaming connection
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	refresh := func() error {
		return db.Model(&b).Association("Events").Find(&b.Events).Error
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	refreshTicker := time.NewTicker(10 * time.Second)
	defer refreshTicker.Stop()

	stream.StartWithContext(ctx, c, func(ctx context.Context, w io.Writer) bool {
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
				sugar.InternalError(c, err)
				return false
			}
		}
		return true
	})

	var logStream *cloudwatchlogs.LogStream
	var err error
	if b.CwLogName != "" {
		logStream, err = awsSession.GetJobStream(b.CwLogName)
		if err != nil {
			sugar.ErrResponse(c, 500, err)
			return
		}
	} else {
		jobDetail, err := awsSession.GetJobDetail(b.BatchID)
		if err != nil {
			sugar.ErrResponse(c, 500, err)
			return
		}
		logStream, err = awsSession.GetJobStream(*jobDetail.Container.LogStreamName)
		if err != nil {
			sugar.ErrResponse(c, 500, err)
			return
		}
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

	stream.Start(ctx, lstream, c, awsSession.Conf().LogGroup)
}

func streamDeploymentLogs(service deployment.Service, c *gin.Context, deployment *models.Deployment) {
	ctx, cancel := WithClose(c)
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
		return db.Model(&deployment).Association("Events").Find(&deployment.Events).Error
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	refreshTicker := time.NewTicker(10 * time.Second)
	defer refreshTicker.Stop()

	stream.StartWithContext(ctx, c, func(ctx context.Context, w io.Writer) bool {
		if deployment.HasStarted() {
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
				sugar.InternalError(c, err)
				return false
			}
		}
		return true
	})

	var logStream *cloudwatchlogs.LogStream
	var err error

	stream.StartWithContext(ctx, c, func(ctx context.Context, w io.Writer) bool {
		logStream, err = service.GetDeploymentStream(ctx, *deployment)
		// No error, or a bad error and we need to exit early
		if err == nil || err != aws.ErrNotFound {
			return false
		}

		// Otherwise, wait

		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			bytes.NewBuffer([]byte{0}).WriteTo(w)
		case <-refreshTicker.C:
		}
		return true
	})

	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	log.Printf("opening log stream: %s", *logStream.LogStreamName)

	lstream := awsSession.NewStream(*logStream)

	go func() {
		for !deployment.HasFinished() {
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
	conf := service.GetServiceConfig()
	stream.Start(ctx, lstream, c, conf.LogGroup)

}
