package auth

import (
	"net/http"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Index handles request to the site root.
func Index(c *gin.Context) {
	session := sessions.Default(c)
	user, loggedIn := middleware.CheckUser(c)

	if !loggedIn {
		session.Clear()
		session.Save()

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"logged_in": false,
		})
		return
	}
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"logged_in": true,
		"login":     user.GithubName,
		"name":      user.Name,
		"gh_id":     user.GithubID,
		"email":     user.Email,
		"token":     user.Token,
	})
}
