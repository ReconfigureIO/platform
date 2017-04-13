package github

import (
	"context"
	"os"

	"github.com/ReconfigureIO/platform/models"
	"github.com/google/go-github/github"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

type GithubService struct {
	OauthConf *oauth2.Config
	db        *gorm.DB
}

func NewService(db *gorm.DB) *GithubService {
	oauthConf := &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Scopes:       []string{"user"},
		Endpoint:     githuboauth.Endpoint,
	}
	return &GithubService{OauthConf: oauthConf, db: db}
}

// Given an access token, fetch the user data from github, and assign
// update or create the user in the db.
func (s *GithubService) GetOrCreateUser(context context.Context, accessToken string) (models.User, error) {
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
	err = q.Assign(u).FirstOrInit(&u).Error
	if err != nil {
		return u, err
	}
	s.db.Save(&u)
	return u, err
}
