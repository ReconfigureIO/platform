//+build integration

package api

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jinzhu/gorm"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/batch"
	"github.com/ReconfigureIO/platform/service/storage"
)

func TestGetPublicBuilds(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		DB(db)
		// create a build in the DB
		builds := []models.Build{
			{
				Project: models.Project{
					Name: "reco-examples",
				},
			},
			{
				Project: models.Project{
					User: models.User{
						ID: "user1",
					},
				},
			},
		}
		for i := range builds {
			err := db.Create(&(builds[i])).Error
			if err != nil {
				t.Fatal(err)
			}
		}
		publicProjectID := builds[0].Project.ID
		pBuilds, err := Build{PublicProjectID: publicProjectID}.publicBuilds()
		if err != nil {
			t.Fatal(err)
		}
		if l := len(pBuilds); l != 1 {
			t.Fatalf("Expected %d build, found %v", 1, l)
		}
		if pBuilds[0].ID != builds[0].ID {
			t.Fatalf("Expected build ID %s, found %s", builds[0].ID, pBuilds[0].ID)
		}
	})
}

func TestDownloadArtifact(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		DB(db)
		now := time.Now()

		user := models.User{
			GithubID: 1,
			Email:    "foo@bar.com",
		}
		db.Create(&user)

		builds := []models.Build{
			{
				Token: "foobar",
				Project: models.Project{
					User: models.User{
						ID: user.ID,
					},
				},
				BatchJob: models.BatchJob{
					ID:      1,
					BatchID: "foobar",
					Events: []models.BatchJobEvent{
						models.BatchJobEvent{
							ID:         "1",
							BatchJobID: 1,
							Timestamp:  now.Add(-5 * time.Minute),
							Status:     "QUEUED",
						},
						models.BatchJobEvent{
							ID:         "2",
							BatchJobID: 1,
							Timestamp:  now.Add(-4 * time.Minute),
							Status:     "STARTED",
						},
						models.BatchJobEvent{
							ID:         "3",
							BatchJobID: 1,
							Timestamp:  now.Add(-3 * time.Minute),
							Status:     "COMPLETED",
						},
					},
				},
			},
		}

		for i := range builds {
			err := db.Create(&(builds[i])).Error
			if err != nil {
				t.Fatal(err)
			}
		}

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		storageService := storage.NewMockService(mockCtrl)
		storageService.EXPECT().Download("builds/"+builds[0].ID+"/artifacts.zip").Return(ioutil.NopCloser(bytes.NewReader([]byte("foo"))), nil)

		build := Build{
			Storage: storageService,
		}
		r := gin.Default()
		r.GET("builds/:id/artifacts", build.DownloadArtifact)

		// Test if human user auth lets you download artifacts
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/builds/"+builds[0].ID+"/artifacts", nil)
		req.SetBasicAuth(strconv.Itoa(user.GithubID), user.Token)
		r.ServeHTTP(w, req)

		if w.Code == 200 {
			t.Error("Human user was allowed to download artifacts")
		}

		// Test if machine token auth lets you download artifacts
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/builds/"+builds[0].ID+"/artifacts?token="+builds[0].Token, nil)
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Fatalf("Machine could not download artifact, response code: %v", w.Code)
		}

		if w.Body.String() != "foo" {
			t.Fatalf("Expected artifact contents to be foo, got %v \n", w.Body.String())
		}

	})
}

func TestBuildInput(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	storageService := storage.NewMockService(mockCtrl)
	batchRepo := models.NewMockBatchRepo(mockCtrl)
	buildRepo := models.NewMockBuildRepo(mockCtrl)
	batchService := batch.NewMockService(mockCtrl)

	build := models.Build{
		ID:    "foobarID",
		Token: "foobartoken",
		Project: models.Project{
			User: models.User{
				ID:       "foobarUserID",
				GithubID: 1234,
				Token:    "foobarUserToken",
			},
		},
	}

	apiBuild := Build{
		Storage:   storageService,
		Repo:      buildRepo,
		BatchRepo: batchRepo,
		AWS:       batchService,
	}
	r := gin.Default()
	r.POST("builds/:id/input", apiBuild.Input)

	buildRepo.EXPECT().ByID(build.ID).Return(build, nil)
	storageService.EXPECT().Upload("builds/"+build.ID+"/build.tar.gz", nil).Return("", nil)
	batchService.EXPECT().RunBuild(
		build,
		"https://"+r.BasePath()+"builds/"+build.ID+"/events?token="+build.Token,
		"https://"+r.BasePath()+"builds/"+build.ID+"/reports?token="+build.Token,
	).Return("foobarBatchJobID", nil)
	batchRepo.EXPECT().New("foobarBatchJobID").Return(models.BatchJob{})
	buildRepo.EXPECT().AddBatchJobToBuild(build, models.BatchJob{}).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/builds/"+build.ID+"/input", nil)
	req.SetBasicAuth(strconv.Itoa(build.Project.User.GithubID), build.Project.User.Token)
	r.ServeHTTP(w, req)
}
