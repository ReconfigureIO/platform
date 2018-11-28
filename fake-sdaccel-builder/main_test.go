package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
)

type eventMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func findIP() net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&(net.FlagLoopback) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			panic(err)
		}
		for _, addr := range addrs {
			if ip, ok := addr.(*net.IPNet); ok {
				if theIP := ip.IP.To4(); len(theIP) == net.IPv4len {
					return theIP
				}
			}
		}
	}
	return nil
}

func prebuildImage() (string, error) {
	cmd := exec.Command(
		"docker", "build", "--quiet", "./",
	)
	id, err := cmd.CombinedOutput()
	if len(id) == 0 {
		panic("No output from docker build")
	}
	return string(id[:len(id)-3]), err
}

type context struct {
	imageID string
}

func TestFakeSdaccelBuilder(t *testing.T) {
	id, err := prebuildImage()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context{
		imageID: id,
	}
	t.Run("shell-scripts", func(t *testing.T) {
		t.Run("graph.sh", ctx.testGraphDotSh)
		t.Run("build.sh", ctx.testBuildDotSh)
		t.Run("simulate.sh", ctx.testSimulateDotSh)
	})
}

func (c context) testGraphDotSh(t *testing.T) {
	t.Parallel()
	var started bool
	var completed bool
	expectedLineCount := 103

	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		var event eventMessage
		err = json.Unmarshal(body, &event)
		if err != nil {
			panic(err)
		}
		switch event.Status {
		case "STARTED":
			started = true
		case "COMPLETED":
			completed = true
		}
	}))

	ip := findIP()
	var err error
	s.Listener, err = net.Listen("tcp4", ip.String()+":0")
	if err != nil {
		panic(err)
	}
	s.Start()
	defer s.Close()

	cmd := exec.Command(
		"docker", "run", "--rm",
		"--env=CALLBACK_URL="+s.URL+"/events/",
		"--env=SLEEP_PERIOD=0.001",
		""+c.imageID,
		"/opt/graph.sh",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Docker run output: %s \n", string(output[:]))
		t.Fatal(err)
	}
	count := bytes.Count(output, []byte("\n"))

	if !started {
		t.Error("Did not recieve STARTED event from script\n")
	}
	if !completed {
		t.Error("Did not recieve COMPLETED event from script\n")
	}
	if count < expectedLineCount {
		t.Errorf("Did not recieve expected number of output lines. Expected: %v Recieved: %v\n", expectedLineCount, count)
	}
}

func (c context) testBuildDotSh(t *testing.T) {
	t.Parallel()
	var started bool
	var completed bool
	expectedLineCount := 104

	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		var event eventMessage
		err = json.Unmarshal(body, &event)
		if err != nil {
			panic(err)
		}
		switch event.Status {
		case "STARTED":
			started = true
		case "COMPLETED":
			completed = true
		}
	}))

	ip := findIP()
	var err error
	s.Listener, err = net.Listen("tcp4", ip.String()+":0")
	if err != nil {
		panic(err)
	}
	s.Start()
	defer s.Close()

	cmd := exec.Command(
		"docker", "run", "--rm",
		"--env=CALLBACK_URL="+s.URL+"/events/",
		"--env=SLEEP_PERIOD=0.001",
		""+c.imageID,
		"/opt/graph.sh",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	count := bytes.Count(output, []byte("\n"))

	if !started {
		t.Error("Did not recieve STARTED event from script\n")
	}
	if !completed {
		t.Error("Did not recieve COMPLETED event from script\n")
	}
	if count < expectedLineCount {
		t.Errorf("Did not recieve expected number of output lines. Expected: %v Recieved: %v\n", expectedLineCount, count)
	}
}

func (c context) testSimulateDotSh(t *testing.T) {
	t.Parallel()
	var started bool
	var completed bool
	expectedLineCount := 102

	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		var event eventMessage
		err = json.Unmarshal(body, &event)
		if err != nil {
			panic(err)
		}
		switch event.Status {
		case "STARTED":
			started = true
		case "COMPLETED":
			completed = true
		}
	}))

	ip := findIP()
	var err error
	s.Listener, err = net.Listen("tcp4", ip.String()+":0")
	if err != nil {
		panic(err)
	}
	s.Start()
	defer s.Close()

	cmd := exec.Command(
		"docker", "run", "--rm",
		"--env=CALLBACK_URL="+s.URL+"/events/",
		"--env=SLEEP_PERIOD=0.001",
		""+c.imageID,
		"/opt/graph.sh",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	count := bytes.Count(output, []byte("\n"))

	if !started {
		t.Error("Did not recieve STARTED event from script\n")
	}
	if !completed {
		t.Error("Did not recieve COMPLETED event from script\n")
	}
	if count < expectedLineCount {
		t.Errorf("Did not recieve expected number of output lines. Expected: %v Recieved: %v\n", expectedLineCount, count)
	}
}
