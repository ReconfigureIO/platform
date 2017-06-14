package api

import (
	"testing"
	"time"
)

func TestTimeToSQLStr(t *testing.T) {
	utcTime := time.Date(2010, 2, 11, 3, 20, 30, 0, time.UTC)
	expected := "2010-02-01 00:00:00"
	if ms := timeToSQLStr(monthStart(utcTime)); ms != expected {
		t.Errorf("Expected %v found %v", expected, ms)
	}
}
