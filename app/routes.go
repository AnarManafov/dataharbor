package main

import (
	"app/controller"
	"app/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Use(middleware.RecoveryMiddleware(),
		middleware.TraceMiddware(),
		middleware.AccessLogger(),
		middleware.CORSMiddleware())

	r.GET("/health", controller.Health)

	r.GET("home_dir", controller.GetHomeDir)

	r.POST("dir", controller.GetDirItems)
}
