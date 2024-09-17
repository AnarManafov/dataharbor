package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
	"github.com/AnarManafov/data_lake_ui/app/core"
	"github.com/AnarManafov/data_lake_ui/app/route"

	"github.com/gin-gonic/gin"
)

func main() {
	initialize()
	stop := make(chan struct{})
	startServer(stop)
}

func initialize() {
	common.InitLogger()
	config.InitCmd()
	config.Init()
}

func startServer(stop chan struct{}) {
	if !common.ServerConfig.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	route.RegisterRoutes(r)

	port := strconv.Itoa(common.ServerConfig.Port)
	fmt.Printf("Starting server on port: %s\n", port)

	ticker, done := core.NewSanitationScheduler()
	go core.SanitationJob(ticker, done)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			common.Logger.Fatal(err)
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		common.Logger.Fatal("Server forced to shutdown:", err)
	}

	ticker.Stop()
	done <- true
}
