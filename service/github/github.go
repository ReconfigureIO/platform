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
func (s *Service) GetOrCreateUser(context context.Context, accessToken string) (models.User, error) {
	oauthClient := s.OauthConf.Client(oauth2.NoContext, &oauth2.Token{AccessToken: accessToken})
	client := github.NewClient(oauthClient)

	user, _, err := client.Users.Get(context, "")

	if err != nil {
		return models.User{}, err
	}

	u := models.User{
		GithubID:          user.GetID(),
		GithubName:        user.GetLogin(),
		Name:              user.GetName(),
		Email:             user.GetEmail(),
		GithubAccessToken: accessToken,
	}

	q := s.db.Where(models.User{GithubID: user.GetID()})
	err = q.Attrs(models.NewUser()).Assign(u).FirstOrInit(&u).Error
	if err != nil {
		return u, err
	}
	s.db.Save(&u)
	return u, err
}

// GetUser fetches a user.
// Given an access token, fetch the user data from github, and get an existing user, and update it with the latest info.
func (s *Service) GetUser(context context.Context, accessToken string) (models.User, error) {
	oauthClient := s.OauthConf.Client(oauth2.NoContext, &oauth2.Token{AccessToken: accessToken})
	client := github.NewClient(oauthClient)

	user, _, err := client.Users.Get(context, "")

	if err != nil {
		return models.User{}, err
	}

	u := models.User{
		GithubID:          user.GetID(),
		GithubName:        user.GetLogin(),
		Name:              user.GetName(),
		Email:             user.GetEmail(),
		GithubAccessToken: accessToken,
	}

	oldUser := models.User{}

	q := s.db.Where(models.User{GithubID: user.GetID()})
	err = q.First(&oldUser).Error
	if err != nil {
		return u, err
	}

	err = s.db.Model(&oldUser).Update(u).Error
	if err != nil {
		return u, err
	}

	return oldUser, err
}
