package github

import (
	"context"
	"os"

	"github.com/ReconfigureIO/platform/models"
	"github.com/google/go-github/github"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"
	ghoauth "golang.org/x/oauth2/github"
)

// Service is Github service.
type Service struct {
	OauthConf *oauth2.Config
	db        *gorm.DB
}

// New creates a new Github service.
func New(db *gorm.DB) *Service {
	oauthConf := &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Scopes:       []string{"user"},
		Endpoint:     ghoauth.Endpoint,
	}
	return &Service{OauthConf: oauthConf, db: db}
}

// GetOrCreateUser fetches or create a user.
// Given an access token, fetch the user data from github, and assign
// update or create the user in the db.
func (s *Service) GetOrCreateUser(ctx context.Context, accessToken string, createNew bool) (models.User, error) {
	oauthClient := s.OauthConf.Client(context.Background(), &oauth2.Token{AccessToken: accessToken})
	client := github.NewClient(oauthClient)

	ghUser, _, err := client.Users.Get(ctx, "")

	if err != nil {
		return models.User{}, err
	}

	u := models.User{
		GithubID:          ghUser.GetID(),
		GithubName:        ghUser.GetLogin(),
		Name:              ghUser.GetName(),
		Email:             ghUser.GetEmail(),
		GithubAccessToken: accessToken,
	}

	q := s.db.Where(models.User{GithubID: ghUser.GetID()})

	var user models.User
	if err = q.First(&user).Error; err != nil {
		// not found
		user = models.NewUser()
		if err != gorm.ErrRecordNotFound {
			return user, err
		}
		if !createNew {
			return user, err
		}
	}

	err = q.Attrs(user).Assign(u).FirstOrInit(&u).Error
	if err != nil {
		return u, err
	}
	s.db.Save(&u)
	return u, err
}
