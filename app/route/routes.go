package route

import (
	"github.com/AnarManafov/data_lake_ui/app/controller"
	"github.com/AnarManafov/data_lake_ui/app/middleware"

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
	r.GET("/initial_dir", controller.FetchInitialDir)
	r.GET("/host_name", controller.FetchHostName)
	r.POST("/dir", controller.FetchDirItems)
	r.POST("/dir/page", controller.FetchDirItemsByPage)
	r.POST("/stage_file", controller.FetchFileStagedForDownload)
}
