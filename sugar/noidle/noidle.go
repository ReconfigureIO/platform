// Package noidle prevents a http connection from remaining idle, by putting nul
// bytes on the wire or by using pingproto if the client supports it.
package noidle

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ReconfigureIO/pingproto"
)

// Upgrade returns a new writer which is used to send content to the client. The
// underlying protocol will speak `ReconfigureIO/pingproto` if the client
// supports it, or otherwise nul bytes will be inserted into the stream at a
// regular interval. Failing to close the resulting WriteCloser will result in a
// goroutine leak. 'appW' stands for 'application Writer', to disambiguate from 'w'.
//
// This is done to defeat the Elastic Load Balancer's idle timeout, as our logs
// may be idle for many hours at a time before continuing. TCP Keep-Alives (not
// to be confused with HTTP's "Connection: Keep-Alive") are not used because the
// ELB does not respect those for keeping a connection alive, only data sent on
// the wire keeps a connection from being terminated as idle.
//
// Additionally, writes to the resulting writer are propagated immediately to
// the client by flushing the ResponseWriter, if supported.
func Upgrade(w http.ResponseWriter, r *http.Request) (appW io.WriteCloser) {
	w = newFlushWriter(w) // Flush for every write.

	appW, ok := pingproto.HTTPTryContentEncoding(w, r)
	if ok {
		return appW
	}

	// Client does not support pingproto. Fall back to writing nul bytes in-stream.
	return newFallback(w)
}

type fallback struct {
	closed  chan<- struct{}
	written chan<- struct{}
	done    <-chan struct{}

	mu sync.Mutex
	io.Writer
}

// pingPeriod may be modified in tests.
var (
	pingPeriodMu sync.Mutex
	pingPeriod   = 10 * time.Second
)

func newFallback(w io.Writer) *fallback {
	closed := make(chan struct{})
	written := make(chan struct{}, 16)
	done := make(chan struct{})
	f := &fallback{
		closed:  closed,
		written: written,
		done:    done,
		Writer:  w,
	}
	go func() {
		defer close(done)

		timer := time.NewTimer(pingPeriod)
		defer timer.Stop()

		ping := timer.C
		var nulByte [1]byte

		for {
			select {
			case <-closed:
				// All done.
				return

			case <-written:
				// If a packet as written, we can reset the idle timer. Also,
				// Ensure we don't race against the timer.
				if !timer.Stop() {
					<-ping
				}

			case <-ping:
				// Put a nul byte in the stream.
				f.Write(nulByte[:])

			}

			timer.Reset(pingPeriod)
		}
	}()
	return f
}

func (f *fallback) Close() error {
	close(f.closed)
	<-f.done
	return nil
}

func (f *fallback) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.written <- struct{}{}
	return f.Writer.Write(p)
}

// newFlushWriter returns a http.ResponseWriter which calls Flush() on w if it
// has that method, for every write. Note that the returned ResponseWriter is
// now of a different underlying type and no longer supports all of the
// interfaces which the input may support.
func newFlushWriter(w http.ResponseWriter) http.ResponseWriter {
	var (
		fw flushWriter
		ok bool
	)

	fw.ResponseWriter = w
	fw.w, ok = w.(interface {
		Write([]byte) (int, error)
		Flush()
	})
	if !ok {
		return w // Flushing not available.
	}
	return fw
}

type flushWriter struct {
	http.ResponseWriter

	w interface {
		Write([]byte) (int, error)
		Flush()
	}
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	fw.w.Flush()
	return n, err
}
