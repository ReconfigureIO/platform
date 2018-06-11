package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/kr/pretty"
)

func main() {
	handler := &handler{}
	http.ListenAndServe(":9090", handler)
}

type handler struct{}

const xAmzTarget = "X-Amz-Target"

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	amzTarget := r.Header.Get(xAmzTarget)
	endpoint, target := rpartition(amzTarget, ".")
	endpointName, endpointVersion := rpartition(endpoint, "_")

	if endpointName != "Logs" {
		msg := fmt.Sprintf("Unsupported endpoint: %q", endpointName)
		http.Error(w, msg, http.StatusNotImplemented)
		return
	}

	// TODO(pwaller): Endpoint versions check?
	_ = endpointVersion

	switch target {
	case "PutLogEvents":
		h.PutLogEvents(w, r)
	default:
		log.Printf("Unsupported request: %q", target)
		r.Write(os.Stderr)
		msg := fmt.Sprintf("Unsupported target: %q", target)
		http.Error(w, msg, http.StatusNotImplemented)
		return
	}

}

func rpartition(s, sep string) (string, string) {
	pos := strings.LastIndex(s, sep)
	if pos == -1 {
		return "", s
	}
	return s[:pos], s[pos+1:]
}

func (h *handler) PutLogEvents(w http.ResponseWriter, r *http.Request) {
	var payload cloudwatchlogs.PutLogEventsInput
	err := jsonutil.UnmarshalJSON(&payload, r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	pretty.Print(payload)
}
