package core

import (
	"log"
	"path"
	"strings"

	"github.com/AnarManafov/data_lake_ui/app/common"

	"os"
	"time"
)

// isOlderThanXHours checks if the given time is older than X hours.
// It returns true if the time is older than X hours, otherwise false.
func isOlderThanXHours(t time.Time, x uint) bool {
	return time.Since(t) > (time.Duration(x) * time.Hour)
}

// NewSanitationScheduler creates a new sanitation scheduler.
// It returns a ticker that runs the sanitation job every X minutes and a done channel.
func NewSanitationScheduler() (*time.Ticker, chan bool) {
	// Start the sanitation job
	common.Logger.Info("Creating a sanitation check job...")
	// The job runs every X minutes
	ticker := time.NewTicker(time.Duration(common.XrdConfig.SanitationJobInterval) * time.Minute)
	done := make(chan bool)
	return ticker, done
}

// CleanStagingDir cleans the staging directory by removing directories that are older than 2 hours.
func CleanStagingDir() {
	files, err := os.ReadDir(common.XrdConfig.StagingPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			common.Logger.Info("Check staging dir: " + file.Name())
			fileInfo, err := file.Info()
			if err == nil && strings.HasPrefix(file.Name(), common.XrdConfig.StagingTmpDirPrefix) && isOlderThanXHours(fileInfo.ModTime(), 2) {
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

// SanitationJob runs the sanitation job at regular intervals.
// It cleans the staging directory by removing directories that are older than 2 hours.
// The job starts immediately and continues running until the done channel receives a signal.
func SanitationJob(ticker *time.Ticker, done chan bool) {
	// Since the Tick is not called immediately, we force the clean at the start
	CleanStagingDir()

	for {
		select {
		case <-done:
			common.Logger.Info("Sanitation check job - done")
			return
		case t := <-ticker.C:
			common.Logger.Info("Sanitation check of staging dir at ", t)
			CleanStagingDir()
		}
	}
}
