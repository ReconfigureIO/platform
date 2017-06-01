package routes

import (
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/api"
	"github.com/ReconfigureIO/platform/api/profile"
	"github.com/ReconfigureIO/platform/auth"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupRoutes sets up api routes.
func SetupRoutes(r gin.IRouter, db *gorm.DB) {
	// Setup index & signup flow
	auth.Setup(r, db)

	// Setup admin
	authMiddleware := gin.BasicAuth(gin.Accounts{
		"admin": "ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264",
	})
	admin := r.Group("/admin", authMiddleware)
	auth.SetupAdmin(admin, db)

	apiRoutes := r.Group("/", auth.TokenAuth(db), auth.RequiresUser())

	if os.Getenv("RECO_FEATURE_BILLING") == "1" {
		fmt.Println("enabling billing api endpoints")

		billing := api.Billing{}
		profile := profile.Profile{db}
		billingRoutes := apiRoutes.Group("/user")
		{
			billingRoutes.GET("", profile.Get)
			billingRoutes.PUT("", profile.Update)
			billingRoutes.POST("/payment-info", billing.Replace)
		}
	}

	build := api.Build{}
	buildRoute := apiRoutes.Group("/builds")
	{
		buildRoute.GET("", build.List)
		buildRoute.POST("", build.Create)
		buildRoute.GET("/:id", build.Get)
		buildRoute.PUT("/:id/input", build.Input)
		buildRoute.GET("/:id/logs", build.Logs)
	}

	project := api.Project{}
	projectRoute := apiRoutes.Group("/projects")
	{
		projectRoute.GET("", project.List)
		projectRoute.POST("", project.Create)
		projectRoute.PUT("/:id", project.Update)
		projectRoute.GET("/:id", project.Get)
	}

	simulation := api.NewSimulation()
	simulationRoute := apiRoutes.Group("/simulations")
	{
		simulationRoute.GET("", simulation.List)
		simulationRoute.POST("", simulation.Create)
		simulationRoute.GET("/:id", simulation.Get)
		simulationRoute.PUT("/:id/input", simulation.Input)
		simulationRoute.GET("/:id/logs", simulation.Logs)
	}

	deployment := api.Deployment{}
	deploymentRoute := apiRoutes.Group("/deployments")
	{
		deploymentRoute.GET("", deployment.List)
		deploymentRoute.POST("", deployment.Create)
		deploymentRoute.GET("/:id", deployment.Get)
		deploymentRoute.GET("/:id/logs", deployment.Logs)
	}

	eventRoutes := r.Group("", auth.TokenAuth(db))
	{
		eventRoutes.POST("/builds/:id/events", build.CreateEvent)
		eventRoutes.POST("/simulations/:id/events", simulation.CreateEvent)
		eventRoutes.POST("/deployments/:id/events", deployment.CreateEvent)
	}

}
