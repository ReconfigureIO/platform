//+build integration

package github

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
)

func TestCreateOrUpdateUser(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		u := models.User{
			GithubID:          123,
			GithubName:        "foo",
			Email:             "baz",
			GithubAccessToken: "foobar",
		}

		// test no create
		createdUser, err := createOrUpdateUser(db, u, false)
		if err != gorm.ErrRecordNotFound {
			t.Error(err)
		}
		if createdUser.GithubName != "" {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", models.User{}, createdUser)
		}

		// test create
		createdUser, err = createOrUpdateUser(db, u, true)
		if err != nil {
			t.Error(err)
		}
		if createdUser.GithubName != u.GithubName {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", u, createdUser)
		}

		// test no overwrite
		u.Name = "not bar"
		createdUser, err = createOrUpdateUser(db, u, true)
		if err != nil {
			t.Error(err)
		}
		if createdUser.Name == u.Name {
			t.Fatalf("User's name got overwritten unexpectedly")
		}
	})
}
