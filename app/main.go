package main

import (
	"fmt"
	"strconv"

	"github.com/AnarManafov/app/common"
	"github.com/AnarManafov/app/config"

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
	panic(r.Run(":" + port))
}
