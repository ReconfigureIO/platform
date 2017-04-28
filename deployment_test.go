package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPing(t *testing.T) {

	setupDB()
	defer clearDB()

	ts := httptest.NewServer(Setup())

	req, _ := http.NewRequest("GET", "/ping", nil)
	resp := httptest.NewRecorder()
	ts.ServeHTTP(resp, req)

	assert.Equal(t, resp.Body.String(), "pong pong")
}
