package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ReconfigureIO/platform/api"
	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/migration"
	"github.com/ReconfigureIO/platform/routes"
)

func TestPing(t *testing.T) {

	ts := httptest.NewServer(setupGin())

	req, _ := http.NewRequest("GET", "/ping", nil)
	resp := httptest.NewRecorder()
	ts.ServeHTTP(resp, req)

	assert.Equal(t, resp.Body.String(), "pong pong")
}

func setupDB() {

}

func setupGin() {

	gin.SetMode(gin.TestMode)

	r := gin.Default()

	secretKey := os.Getenv("SECRET_KEY_BASE")

	// setup components
	db := setupDB()

	store := sessions.NewCookieStore([]byte(secretKey))
	r.Use(sessions.Sessions("paus", store))
	r.Use(auth.SessionAuth(db))

	r.LoadHTMLGlob("templates/*")

	// ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	routes.SetupRoutes(r, db)

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
