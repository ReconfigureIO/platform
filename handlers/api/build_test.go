//+build integration

package api

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
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
		publicProjectID = builds[0].Project.ID
		pBuilds, err := Build{}.publicBuilds()
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
