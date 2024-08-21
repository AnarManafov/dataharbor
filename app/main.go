package main

import (
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"

	"github.com/AnarManafov/app/common"
	"github.com/AnarManafov/app/config"

	"github.com/gin-gonic/gin"

	"os"
	"time"
)

func isOlderThanXHours(_t time.Time, _x uint) bool {
	return time.Since(_t) > (time.Duration(_x) * time.Hour)
}

func main() {
	config.InitCmd()
	config.Init()

	if !common.ServerConfig.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// Start the sanitation job
	common.Logger.Info("Creating a sanitation check job...")
	// The job runs every hour
	// TODO: Move the job interval to the config
	ticker := time.NewTicker(time.Duration(1) * time.Hour)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				common.Logger.Info("DBG: Sanitation check job - done")
				return
			case t := <-ticker.C:
				common.Logger.Info("Sanitation check of staging dir at ", t)
				files, err := os.ReadDir(common.XrdConfig.StagingPath)
				if err != nil {
					log.Fatal(err)
				}

				for _, file := range files {
					if file.IsDir() {
						common.Logger.Info("Check staging dir: " + file.Name())
						file_inf, err := file.Info()
						if err == nil && strings.HasPrefix(file.Name(), "stg_") && isOlderThanXHours(file_inf.ModTime(), 2) {
							dirToRemove := path.Join(common.XrdConfig.StagingPath, file.Name())
							common.Logger.Info("Need to remove staging dir: " + dirToRemove)
							if err := os.RemoveAll(dirToRemove); err != nil {
								common.Logger.Error(err)
							} else {
								common.Logger.Info("Removed staging dir: " + dirToRemove)
							}
						}
					}
				}
			}
		}
	}()

	r := gin.New()
	RegisterRoutes(r)

	port := strconv.Itoa(common.ServerConfig.Port)
	fmt.Printf("start server at port: %s\n", port)

	err := r.Run(":" + port)
	if err != nil {
		common.Logger.Fatal(err)
	}

	ticker.Stop()
	done <- true
}
