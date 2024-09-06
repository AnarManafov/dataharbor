package main

import (
	"github.com/AnarManafov/app/controller"
	"github.com/AnarManafov/app/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Apply middlewares
	r.Use(middleware.RecoveryMiddleware(),
		middleware.TraceMiddleware(),
		middleware.AccessLogger(),
		middleware.CORSMiddleware())

	// Define routes
	r.GET("/health", controller.Health)
	r.GET("/initial_dir", controller.GetInitialDir)
	r.GET("/host_name", controller.GetHostName)
	r.POST("/dir", controller.GetDirItems)
	r.POST("/stage_file", controller.GetFileStagedForDownload)
}
