package main

import (
	"github.com/ReconfigureIO/platform/api"
	"github.com/gin-gonic/gin"
)

func setupRoutes(r *gin.RouterGroup) {
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
		simulationRoute.POST("/:id/events", simulation.CreateEvent)
		simulationRoute.GET("/:id/logs", simulation.Logs)
	}

	deployment := api.Deployment{}
	deploymentRoute := r.Group("/deployments")
	{
		deploymentRoute.GET("", deployment.List)
		deploymentRoute.POST("", deployment.Create)
		deploymentRoute.PUT("/:id", deployment.Update)
		deploymentRoute.GET("/:id", deployment.Get)
		deploymentRoute.PUT("/:id/input", deployment.Input)
		deploymentRoute.GET("/:id/logs", deployment.Logs)
	}
}
