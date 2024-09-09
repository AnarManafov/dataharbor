package main

import (
	"fmt"
	"strconv"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
	"github.com/AnarManafov/data_lake_ui/app/core"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitCmd()
	config.Init()

	if !common.ServerConfig.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	RegisterRoutes(r)

	port := strconv.Itoa(common.ServerConfig.Port)
	fmt.Printf("Starting server on port: %s\n", port)

	ticker, done := core.NewSanitationScheduler()
	go core.SanitationJob(ticker, done)

	err := r.Run(":" + port)
	if err != nil {
		common.Logger.Fatal(err)
	}

	ticker.Stop()
	done <- true
}
