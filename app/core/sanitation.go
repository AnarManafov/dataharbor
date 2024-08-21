package core

import (
	"log"
	"path"
	"strings"

	"github.com/AnarManafov/app/common"

	"os"
	"time"
)

func IsOlderThanXHours(_t time.Time, _x uint) bool {
	return time.Since(_t) > (time.Duration(_x) * time.Hour)
}

func NewSanitationScheduler() (ticker *time.Ticker, done chan bool) {
	// Start the sanitation job
	common.Logger.Info("Creating a sanitation check job...")
	// The job runs every X minutes
	ticker = time.NewTicker(time.Duration(common.XrdConfig.SanitationJobInterval) * time.Minute)
	done = make(chan bool)
	return ticker, done
}

func SanitationJob(ticker *time.Ticker, done chan bool) {
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
					if err == nil && strings.HasPrefix(file.Name(), common.XrdConfig.StagingTmpDirPrefix) && IsOlderThanXHours(file_inf.ModTime(), 2) {
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
}
