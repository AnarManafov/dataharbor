package main

import (
	"fmt"
	"strconv"

	"github.com/AnarManafov/app/common"
	"github.com/AnarManafov/app/config"
	"github.com/AnarManafov/app/core"

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
	fmt.Printf("start server at port: %s\n", port)

	ticker, done := core.NewSanitationScheduler()
	go core.SanitationJob(ticker, done)

	err := r.Run(":" + port)
	if err != nil {
		common.Logger.Fatal(err)
	}

	ticker.Stop()
	done <- true
}
