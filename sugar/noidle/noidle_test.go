package noidle

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ReconfigureIO/pingproto"

	"github.com/fortytw2/leaktest"
)

func TestFlushWriter(t *testing.T) {
	// TestFlushWriter checks that - when the application does not support
	// pingproto - the noidle Upgrade appW calls flush for every write.

	defer leaktest.Check(t)()

	w := &fakeResponseWriter{}
	r := httptest.NewRequest("GET", "/", nil)
	appW := Upgrade(w, r)
	defer appW.Close()

	_, _ = io.WriteString(appW, "Hello, world")

	if !w.calledWrite {
		t.Errorf("w.calledWrite = %t", w.calledWrite)
	}
	if !w.calledFlush {
		t.Errorf("w.calledFlush = %t", w.calledFlush)
	}
}

func TestUpgradeFlushing(t *testing.T) {
	// TestUpgradeFlushing ensures that when a single byte is written, it is
	// seen by the client immediately.
	//
	// Both PingProto and Plain-old-http clients are tested.

	t.Run("ClientPingProto",
		testUpgrade{
			pingproto.NewHTTPClient(nil),
		}.run,
	)
	t.Run("ClientPlain",
		testUpgrade{http.DefaultClient}.run,
	)
}

func TestFallbackPings(t *testing.T) {
	// TestFallbackPings ensures that when we use the noidle.fallback
	// (plain-old-http-clients), an idle server still sends pingproto packets.
	//
	// The idea is that the server just sits there, not doing anything. The ping
	// period is reduced so that the client recieves a ping. Once the client
	// sees the ping, we're done.
	//
	// Note that there is no analogous test for pingproto, since pingproto has a
	// test for this behaviour internally and we can't change the poll period of
	// pingproto.

	pingPeriodMu.Lock()
	defer pingPeriodMu.Unlock()
	oldPingPeriod := pingPeriod
	pingPeriod = 10 * time.Microsecond
	defer func() { pingPeriod = oldPingPeriod }()

	clientReceivedByte := make(chan struct{})

	defer leaktest.Check(t)()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appW := Upgrade(w, r)
		defer func() {
			err := appW.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		select {
		case <-clientReceivedByte:
		case <-time.After(1 * time.Second):
			// Should be much longer than the scheduling delay.
			t.Fatal("timeout waiting for <-clientReceivedByte")
		}
	}))
	defer s.Close()

	resp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
		return
	}
	defer resp.Body.Close()

	// Note: we may get more than one ping reponse. OK. We'll just read the first one.
	var buf [1]byte
	n, err := resp.Body.Read(buf[:])
	if err != nil {
		t.Fatal(err)
	}

	got := string(buf[:n])
	if got != "\x00" {
		t.Fatalf("unexpected response: %q", got)
	}
	close(clientReceivedByte)
}

type testUpgrade struct {
	httpClient *http.Client
}

func (tu testUpgrade) run(t *testing.T) {
	clientReceivedByte := make(chan struct{})

	defer leaktest.Check(t)()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appW := Upgrade(w, r)
		defer func() {
			err := appW.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		_, err := appW.Write([]byte{'z'})
		if err != nil {
			t.Fatal(err)
		}

		select {
		case <-clientReceivedByte:
		case <-time.After(1 * time.Second):
			// Should be much longer than the scheduling delay.
			t.Fatal("timeout waiting for <-clientReceivedByte")
		}
	}))
	defer s.Close()

	resp, err := tu.httpClient.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var buf [16]byte
	n, err := resp.Body.Read(buf[:])
	if err != nil {
		t.Fatal(err)
	}

	got := string(buf[:n])
	if got != "z" {
		t.Fatalf("unexpected response: %q", got)
	}

	close(clientReceivedByte)
}

type fakeResponseWriter struct {
	http.ResponseWriter      // nil; so that we implement ResponseWriter.
	calledWrite, calledFlush bool
}

func (f *fakeResponseWriter) Write(p []byte) (int, error) {
	f.calledWrite = true
	return len(p), nil
}

func (f *fakeResponseWriter) Flush() {
	f.calledFlush = true
}
