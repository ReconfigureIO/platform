package routes

import (
	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/handlers"
	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/handlers/profile"
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/ReconfigureIO/platform/service/storage"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupRoutes sets up api routes.
func SetupRoutes(config config.RecoConfig, secretKey string, r *gin.Engine, db *gorm.DB, events events.EventService, leads leads.Leads, storage storage.Service, deploy deployment.Service, publicProjectID string) *gin.Engine {
	// setup common routes
	store := sessions.NewCookieStore([]byte(secretKey))
	r.Use(sessions.Sessions("paus", store))
	r.Use(middleware.SessionAuth(db))

	// setup index
	r.GET("/", handlers.Index)

	// Setup authenticated admin
	authMiddleware := gin.BasicAuth(gin.Accounts{
		"admin": "ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264",
	})
	admin := r.Group("/admin", authMiddleware)
	SetupAdmin(admin, db, leads)

	// signup & login flow
	SetupAuth(r, db, leads)

	apiRoutes := r.Group("/", middleware.TokenAuth(db, events), middleware.RequiresUser())

	billing := api.Billing{}
	profile := profile.Profile{DB: db, Leads: leads}
	billingRoutes := apiRoutes.Group("/user")
	{
		billingRoutes.GET("", profile.Get)
		billingRoutes.PUT("", profile.Update)
		billingRoutes.GET("/payment-info", billing.Get)
		billingRoutes.POST("/payment-info", billing.Replace)
		billingRoutes.GET("/hours-remaining", billing.RemainingHours)
	}

	build := api.Build{Events: events, Storage: storage}
	buildRoute := apiRoutes.Group("/builds")
	{
		buildRoute.GET("", build.List)
		buildRoute.POST("", build.Create)
		buildRoute.GET("/:id", build.Get)
		buildRoute.PUT("/:id/input", build.Input)
		buildRoute.GET("/:id/logs", build.Logs)
		buildRoute.GET("/:id/reports", build.Report)
	}

	project := api.Project{
		Events:          events,
		PublicProjectID: publicProjectID,
	}
	projectRoute := apiRoutes.Group("/projects")
	{
		projectRoute.GET("", project.List)
		projectRoute.POST("", project.Create)
		projectRoute.PUT("/:id", project.Update)
		projectRoute.GET("/:id", project.Get)
	}

	simulation := api.NewSimulation(events, storage)
	simulationRoute := apiRoutes.Group("/simulations")
	{
		simulationRoute.GET("", simulation.List)
		simulationRoute.POST("", simulation.Create)
		simulationRoute.GET("/:id", simulation.Get)
		simulationRoute.PUT("/:id/input", simulation.Input)
		simulationRoute.GET("/:id/logs", simulation.Logs)
	}

	graph := api.Graph{Events: events, Storage: storage}
	graphRoute := apiRoutes.Group("/graphs")
	{
		graphRoute.GET("", graph.List)
		graphRoute.POST("", graph.Create)
		graphRoute.GET("/:id", graph.Get)
		graphRoute.PUT("/:id/input", graph.Input)
		graphRoute.GET("/:id/graph", graph.Download)
	}

	deployment := api.Deployment{
		Events:           events,
		Storage:          storage,
		DeployService:    deploy,
		UseSpotInstances: config.FeatureUseSpotInstances,
	}

	deploymentRoute := apiRoutes.Group("/deployments")
	{
		deploymentRoute.GET("", deployment.List)
		deploymentRoute.POST("", deployment.Create)
		deploymentRoute.GET("/:id", deployment.Get)
		deploymentRoute.GET("/:id/logs", deployment.Logs)
	}

	eventRoutes := r.Group("", middleware.TokenAuth(db, events))
	{
		eventRoutes.POST("/builds/:id/events", build.CreateEvent)
		eventRoutes.POST("/simulations/:id/events", simulation.CreateEvent)
		eventRoutes.POST("/graphs/:id/events", graph.CreateEvent)
		eventRoutes.POST("/deployments/:id/events", deployment.CreateEvent)
	}

	reportRoutes := r.Group("", middleware.TokenAuth(db, events))
	{
		reportRoutes.POST("/builds/:id/reports", build.CreateReport)
	}
	return r
}
