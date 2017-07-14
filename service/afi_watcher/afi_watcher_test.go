package afi_watcher

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
)
struct fake_BuildRepo {

}

func (repo *fake_BuildRepo) GetBuildsWithStatus(statuses []string, limit int) ([]models.Build, error) {
	return nil, nil
}

func TestFindAfi(t *testing.T) {
	d := fake_BuildRepo{}
	err := FindAfi(d)
	if err != nil {
		t.Fatalf("bork bork", err)
	}
}