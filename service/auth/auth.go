package auth

import (
	"context"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
)

// Service allows authenticating a user with a third party OAuth2 endpoint.
// See service/auth/github for an example which implements this.
type Service interface {
	RedirectURL(state string) string
	Exchange(ctx context.Context, token string) (string, error)
	GetOrCreateUser(ctx context.Context, accessToken string, createNew bool) (models.User, error)
}

// NOPService implements an authenticator which never denies access.
// This is unsafe to use in any non-test environment.
// The Exchange() method returns the token passed in as the server-side token.
type NOPService struct {
	DB *gorm.DB
}

// RedirectURL generates a URL to be handed to a client. In the case of
// NOPService, cut out the middle man and go straight to /oauth/callback, which
// ordinarily the 3rd party would be responsible for directing the user to.s
func (s *NOPService) RedirectURL(state string) string {
	return "/oauth/callback?state=" + state
}

// Exchange is a part of the OAuth2 contract, whereby we take the code returned
// to us by the service via the user, and then make a call to the server to
// exchange this code for an OAuth2 access token.
// For the NOPService, we just return the code as the access token.
func (s *NOPService) Exchange(ctx context.Context, code string) (string, error) {
	return code, nil
}

// GetOrCreateUser returns the user in the database and creates it if it does not exist.
// For the NOPServe, we always return the same test user.
func (s *NOPService) GetOrCreateUser(ctx context.Context, accessToken string, createNew bool) (models.User, error) {
	return models.CreateOrUpdateUser(s.DB, models.User{
		GithubName:        "reconfigureio-test-user",
		GithubID:          1234,
		GithubAccessToken: accessToken,
		Email:             "test-user@reconfigure.io",
	}, true)
}
