package routes

import (
	"net/url"

	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/handlers"
	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/handlers/profile"
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/auth"
	"github.com/ReconfigureIO/platform/service/batch"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/ReconfigureIO/platform/service/storage"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupRoutes sets up api routes.
func SetupRoutes(
	config config.RecoConfig,
	secretKey string,
	apiBaseURL url.URL,
	r *gin.Engine,
	db *gorm.DB,
	awsService batch.Service,
	events events.EventService,
	leads leads.Leads,
	storage storage.Service,
	deploy deployment.Service,
	publicProjectID string,
	authService auth.Service,
	simRepo models.SimulationRepo,
	buildRepo models.BuildRepo,
	batchRepo models.BatchRepo,
) *gin.Engine {

	// setup common routes
	store := sessions.NewCookieStore([]byte(secretKey))
	r.Use(sessions.Sessions("paus", store))
	r.Use(middleware.SessionAuth(db))

	// setup index
	if config.Env == "development-on-prem" {
		r.GET("/", handlers.IndexOnPrem)
		SetupAuthOnPrem(r, db)
	} else {
		r.GET("/", handlers.Index)

		// Setup authenticated admin
		authMiddleware := gin.BasicAuth(gin.Accounts{
			"admin": "ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264",
		})
		admin := r.Group("/admin", authMiddleware)
		SetupAdmin(admin, db, leads)

		// signup & login flow
		SetupAuth(r, db, leads, authService)
	}

	apiRoutes := r.Group("/", middleware.TokenAuth(db, events, config), middleware.RequiresUser())

	billing := api.Billing{}
	profile := profile.Profile{
		DB:    db,
		Leads: leads,
	}
	billingRoutes := apiRoutes.Group("/user")
	{
		billingRoutes.GET("", profile.Get)
		billingRoutes.PUT("", profile.Update)
		billingRoutes.GET("/payment-info", billing.Get)
		billingRoutes.POST("/payment-info", billing.Replace)
		billingRoutes.GET("/hours-remaining", billing.RemainingHours)
	}

	build := api.Build{
		APIBaseURL:      apiBaseURL,
		Events:          events,
		Storage:         storage,
		PublicProjectID: publicProjectID,
		AWS:             awsService,
		Repo:            buildRepo,
		BatchRepo:       batchRepo,
	}
	buildRoute := apiRoutes.Group("/builds")
	{
		buildRoute.GET("", build.List)
		buildRoute.POST("", build.Create)
		buildRoute.GET("/:id", build.Get)
		buildRoute.PUT("/:id/input", build.Input)
		buildRoute.GET("/:id/logs", build.Logs)
		buildRoute.GET("/:id/reports", build.Report)
		if config.Env == "development-on-prem" {
			buildRoute.GET("/:id/artifacts", build.DownloadArtifact)
		}
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

	simulation := api.Simulation{
		APIBaseURL: apiBaseURL,
		AWS:        awsService,
		Events:     events,
		Storage:    storage,
		Repo:       simRepo,
	}
	simulationRoute := apiRoutes.Group("/simulations")
	{
		simulationRoute.GET("", simulation.List)
		simulationRoute.POST("", simulation.Create)
		simulationRoute.GET("/:id", simulation.Get)
		simulationRoute.PUT("/:id/input", simulation.Input)
		simulationRoute.GET("/:id/logs", simulation.Logs)
		simulationRoute.GET("/:id/reports", simulation.Report)
	}

	graph := api.Graph{
		APIBaseURL: apiBaseURL,
		AWS:        awsService,
		Events:     events,
		Storage:    storage,
	}
	graphRoute := apiRoutes.Group("/graphs")
	{
		graphRoute.GET("", graph.List)
		graphRoute.POST("", graph.Create)
		graphRoute.GET("/:id", graph.Get)
		graphRoute.PUT("/:id/input", graph.Input)
		graphRoute.GET("/:id/graph", graph.Download)
	}

	deployment := api.Deployment{
		APIBaseURL:       apiBaseURL,
		Events:           events,
		Storage:          storage,
		DeployService:    deploy,
		AWS:              awsService,
		UseSpotInstances: config.FeatureUseSpotInstances,
		PublicProjectID:  publicProjectID,
	}
	deploymentRoute := apiRoutes.Group("/deployments")
	{
		deploymentRoute.GET("", deployment.List)
		deploymentRoute.POST("", deployment.Create)
		deploymentRoute.GET("/:id", deployment.Get)
		deploymentRoute.GET("/:id/logs", deployment.Logs)
	}

	eventRoutes := r.Group("", middleware.TokenAuth(db, events, config))
	{
		eventRoutes.POST("/builds/:id/events", build.CreateEvent)
		eventRoutes.POST("/simulations/:id/events", simulation.CreateEvent)
		eventRoutes.POST("/graphs/:id/events", graph.CreateEvent)
		eventRoutes.POST("/deployments/:id/events", deployment.CreateEvent)
	}

	reportRoutes := r.Group("", middleware.TokenAuth(db, events, config))
	{
		reportRoutes.POST("/builds/:id/reports", build.CreateReport)
		reportRoutes.POST("/simulations/:id/reports", simulation.CreateReport)
	}
	return r
}
