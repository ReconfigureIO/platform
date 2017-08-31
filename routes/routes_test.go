// +build integration

package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func init() {
	// Switch to test mode so you don't get such noisy output
	gin.SetMode(gin.TestMode)
}

func TestIndexHandler(t *testing.T) {
	gormConnDets := os.Getenv("DATABASE_URL")
	if gormConnDets == "" {
		t.Skip()
		return
	}

	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		t.Error(err)
		return
	}

	// Setup router
	r := gin.Default()
	r.LoadHTMLGlob("../templates/*")
	r = SetupRoutes("secretKey", r, db)

	// Create a mock request to the index.
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
}
