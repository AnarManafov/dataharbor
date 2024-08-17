package main

import (
	"github.com/AnarManafov/app/controller"
	"github.com/AnarManafov/app/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Use(middleware.RecoveryMiddleware(),
		middleware.TraceMiddware(),
		middleware.AccessLogger(),
		middleware.CORSMiddleware())

	r.GET("/health", controller.Health)

	r.GET("home_dir", controller.GetHomeDir)

	r.GET("host_name", controller.GetHostName)

	r.POST("dir", controller.GetDirItems)
}
