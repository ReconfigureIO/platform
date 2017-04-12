package routes

import (
	"github.com/ReconfigureIO/platform/api"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.RouterGroup) {
	build := api.Build{}
	buildRoute := r.Group("/builds")
	{
		buildRoute.GET("", build.List)
		buildRoute.POST("", build.Create)
		buildRoute.GET("/:id", build.Get)
		buildRoute.PUT("/:id/input", build.Input)
		buildRoute.GET("/:id/logs", build.Logs)
	}

	project := api.Project{}
	projectRoute := r.Group("/projects")
	{
		projectRoute.GET("", project.List)
		projectRoute.POST("", project.Create)
		projectRoute.PUT("/:id", project.Update)
		projectRoute.GET("/:id", project.Get)
	}

	simulation := api.Simulation{}
	simulationRoute := r.Group("/simulations")
	{
		simulationRoute.GET("", simulation.List)
		simulationRoute.POST("", simulation.Create)
		simulationRoute.GET("/:id", simulation.Get)
		simulationRoute.PUT("/:id/input", simulation.Input)
		simulationRoute.GET("/:id/logs", simulation.Logs)
	}
}
