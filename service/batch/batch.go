package batch

import (
	"context"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar/noidle"
)

// Service supplies the ability to run and monitor batch jobs.
type Service interface {
	RunBuild(build models.Build, callbackURL, reportsURL string) (string, error)
	RunGraph(graph models.Graph, callbackURL string) (string, error)
	RunSimulation(inputArtifactURL, callbackURL, command string) (string, error)

	HaltJob(batchID string) error

	Logs(ctx context.Context, batchJob *models.BatchJob) (io.ReadCloser, error)
}

// CopyLogs live-streams the logs from batchJob to http.ResponseWriter.
//
// Under the hood, it uses pingproto if possible, or otherwise regularly puts
// nul-bytes on the wire to connection avoid idle timeouts.
func CopyLogs(
	ctx context.Context,
	svc Service,
	w http.ResponseWriter,
	r *http.Request,
	batchJob interface {
		HasStarted() bool
		HasFinished() bool
		GetLogName() string
	},
) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Need this cancel in case closeNotify not supported.

	closeNotify(w, cancel) // Cancel if downstream goes away.

	started, finished := await(ctx, batchJob, func() {
		// this function needs to update the batch job 
	}) // TODO campgareth: figure out what's going on here, b should be a models BatchJob but there's also an update function required
	<-started
	go func() {
		<-finished
		cancel()
	}()

	rc := svc.Logs(ctx, batchJob)
	defer func() {
		err2 := rc.Close()
		if err == nil {
			err = err2
		}
	}()

	// noidle also causes every write to flush, negating the effects of
	// buffering.
	appW := noidle.Upgrade(w, r)
	defer func() {
		err2 := appW.Close()
		if err2 != nil {
			log.WithError(err2).Warnln("CopyLogs: appW.Close")
		}
	}()

	_, err = io.Copy(appW, rc)
	return err
}

func closeNotify(w http.ResponseWriter, cancel func()) {
	closeNotifier, ok := w.(http.CloseNotifier)
	if !ok {
		return
	}
	go func() {
		<-closeNotifier.CloseNotify()
		cancel()
	}()
}

func await(
	ctx context.Context,
	hasStartFinisher interface {
		HasStarted() bool
		HasFinished() bool
	},
	update func(),
) (
	started, finished <-chan struct{},
) {
	started = make(chan struct{})
	finished = make(chan struct{})

	const pollPeriod = 10 * time.Second

	go func(started, finished chan<- struct{}) {
		ticker := time.NewTicker(pollPeriod)
		defer ticker.Stop()
		tick := ticker.C

		for {
			select {
			case <-ctx.Done():
				if started != nil {
					close(started)
				}
				if finished != nil {
					close(finished)
				}
				return
			case <-tick:
			}

			update()

			if started != nil && hasStartFinisher.HasStarted() {
				close(started)
				started = nil
			}
			if finished != nil && hasStartFinisher.HasFinished() {
				close(finished)
				finished = nil
			}

			if started == nil && finished == nil {
				return
			}
		}
	}(started, finished)

	return started, finished
}
