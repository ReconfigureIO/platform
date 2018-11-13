package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/batch"
	"github.com/golang/mock/gomock"
)

var emptyReport = "{\"moduleName\":\"\",\"partName\":\"\",\"lutSummary\":{\"description\":\"\",\"used\":0,\"available\":0,\"utilisation\":0,\"detail\":null},\"regSummary\":{\"description\":\"\",\"used\":0,\"available\":0,\"utilisation\":0,\"detail\":null},\"blockRamSummary\":{\"description\":\"\",\"used\":0,\"available\":0,\"utilisation\":0,\"detail\":null},\"ultraRamSummary\":{\"description\":\"\",\"used\":0,\"available\":0,\"utilisation\":0},\"dspBlockSummary\":{\"description\":\"\",\"used\":0,\"available\":0,\"utilisation\":0},\"weightedAverage\":{\"description\":\"\",\"used\":0,\"available\":0,\"utilisation\":0}}"

func TestServiceInterface(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	s := batch.NewMockService(mockCtrl)
	s.EXPECT().RunSimulation("foo", "bar", "test").Return("foobar", nil)
	ss, err := s.RunSimulation("foo", "bar", "test")
	if err != nil || ss != "foobar" {
		t.Error("unexpected result")
	}
}

func TestSimulationReport(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	simRepo := models.NewMockSimulationRepo(mockCtrl)
	simRepo.EXPECT().ByIDForUser("foosim", "foouser").Return(models.Simulation{ID: "foosim"}, nil)
	simRepo.EXPECT().GetReport("foosim").Return(models.SimulationReport{}, nil)
	s := Simulation{
		Repo: simRepo,
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("reco_user", models.User{ID: "foouser"})
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "foosim"})
	s.Report(c)
	if c.Writer.Status() != 200 {
		t.Error("Expected 200 status, got: ", c.Writer.Status())
	}
}

func TestSimulationCreateReport(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	simRepo := models.NewMockSimulationRepo(mockCtrl)
	simRepo.EXPECT().ByID("foosim").Return(models.Simulation{ID: "foosim", Token: "footoken"}, nil)
	simRepo.EXPECT().StoreReport("foosim", models.Report{}).Return(nil)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?token=footoken", strings.NewReader(emptyReport))
	c.Request.Header.Add("Content-Type", "application/vnd.reconfigure.io/reports-v1+json")
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "foosim"})

	s := Simulation{
		Repo: simRepo,
	}
	s.CreateReport(c)
	if c.Writer.Status() != 200 {
		t.Error("Expected 200 status, got: ", c.Writer.Status())
	}
}
