package core

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

// Helper function to determine if a file/directory exceeds the retention period
func isOlderThanXHours(t time.Time, x uint) bool {
	return time.Since(t) > (time.Duration(x) * time.Hour)
}

// NewSanitationScheduler creates a background process to prevent disk space exhaustion
// by periodically cleaning up abandoned downloads and temporary files
func NewSanitationScheduler() (*time.Ticker, chan bool) {
	cfg := config.GetConfig()
	common.Logger.Info("Creating a sanitation check job...")

	// Default to 30 minutes if not configured to prevent zero interval
	interval := cfg.XRD.SanitationJobInterval
	if interval == 0 {
		interval = 30
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	done := make(chan bool)
	return ticker, done
}

// CheckAndRemoveOldFiles cleans up the staging directory by removing files
// older than 24 hours to prevent disk space exhaustion from abandoned transfers
func CheckAndRemoveOldFiles() {
	cfg := config.GetConfig()
	stagingPath := cfg.XRD.StagingPath
	if stagingPath == "" {
		common.Logger.Warn("Staging path is not configured, skipping sanitation")
		return
	}

	entries, err := os.ReadDir(stagingPath)
	if err != nil {
		common.Logger.Error("Failed to read staging directory:", err)
		return
	}

	for _, entry := range entries {
		// Only process directories with our prefix to avoid removing unrelated files
		if !entry.IsDir() || (cfg.XRD.StagingTmpDirPrefix != "" &&
			!strings.HasPrefix(entry.Name(), cfg.XRD.StagingTmpDirPrefix)) {
			continue
		}

		fullPath := filepath.Join(stagingPath, entry.Name())

		info, err := entry.Info()
		if err != nil {
			common.Logger.Error("Failed to get info for directory:", err)
			continue
		}

		// 24 hours threshold for cleaning up staged files
		if time.Since(info.ModTime()) > 24*time.Hour {
			common.Logger.Infof("Removing old staged directory: %s", fullPath)
			err := os.RemoveAll(fullPath)
			if err != nil {
				common.Logger.Errorf("Failed to remove staged directory: %s, error: %s", fullPath, err)
			}
		}
	}
}

// SanitationJob runs in the background to manage disk space by periodically
// cleaning up temporary files that may have been abandoned by client disconnects
func SanitationJob(ticker *time.Ticker, done chan bool) {
	common.Logger.Info("Starting sanitation job...")
	for {
		select {
		case <-ticker.C:
			common.Logger.Info("Running sanitation job...")
			CheckAndRemoveOldFiles()
		case <-done:
			common.Logger.Info("Stopping sanitation job...")
			return
		}
	}
}
