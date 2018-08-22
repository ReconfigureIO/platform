//+build integration

package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jinzhu/gorm"

	"github.com/ReconfigureIO/platform/models"
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
		// create a build in the DB
		builds := []models.Build{
			{
				ID: "foobar",
				Project: models.Project{
					User: models.User{
						ID: "user1",
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

		user := models.User{
			ID:       "user1",
			GithubID: 1,
			Email:    "foo@bar.com",
		}
		db.Create(&user)

		for i := range builds {
			err := db.Create(&(builds[i])).Error
			if err != nil {
				t.Fatal(err)
			}
		}
		fmt.Println("I made the objects in the DB boss")

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		storageService := storage.NewMockService(mockCtrl)

		context := &gin.Context{
			Params: []gin.Param{
				gin.Param{
					Key:   "id",
					Value: builds[0].ID,
				},
			},
			Keys: make(map[string]interface{})
		}

		apiBuild := Build{
			Storage: storageService,
		}
		storageService.EXPECT().Download("builds/"+builds[0].ID+"/artifacts.zip").Return(ioutil.NopCloser(bytes.NewReader([]byte("foo"))), nil)

		fmt.Println("I got right up to just before we make the call boss")
		apiBuild.DownloadArtifact(context)

	})
}
