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

type UserError string

func (e UserError) Error() string {
	return string(e)
}

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
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     ghoauth.Endpoint,
	}
	return &Service{OauthConf: oauthConf, db: db}
}

// RedirectURL generates a URL to be followed by the client, where the client
// can choose through a UI to allow the application to use their information.
func (s *Service) RedirectURL(state string) string {
	return s.OauthConf.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange is a part of the OAuth2 contract, whereby we take the code returned
// to us by the service via the user, and then make a call to the server to
// exchange this code for an OAuth2 access token.
func (s *Service) Exchange(ctx context.Context, code string) (string, error) {
	token, err := s.OauthConf.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
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

	// The email we got back was empty, we search for a new one
	if u.Email == "" {
		emails, _, err := client.Users.ListEmails(ctx, nil)
		if err != nil {
			return u, err
		}
		for _, e := range emails {
			if e.GetPrimary() {
				u.Email = e.GetEmail()
			}
		}
	}

	// If we still have no email, error
	if u.Email == "" {
		return u, UserError("No valid email found")
	}

	u, err = models.CreateOrUpdateUser(s.db, u, createNew)

	return u, err
}
