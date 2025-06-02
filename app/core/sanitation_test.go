package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/stretchr/testify/assert"
)

func TestIsOlderThanXHours(t *testing.T) {
	type args struct {
		_t time.Time
		_x uint
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"", args{time.Date(
			2021, 8, 15, 14, 30, 45, 100, time.Local), 3}, true},
		{"", args{time.Now().Add(-time.Hour * time.Duration(2)), 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOlderThanXHours(tt.args._t, tt.args._x); got != tt.want {
				t.Errorf("isOlderThanXHours(%v, %v) = %v, want %v", tt.args._t, tt.args._x, got, tt.want)
			}
		})
	}
}

// TestNewSanitationScheduler tests the NewSanitationScheduler function.
func TestNewSanitationScheduler(t *testing.T) {
	// Set a non-zero interval for the test to avoid the panic
	oldInterval := common.XrdConfig.SanitationJobInterval
	common.XrdConfig.SanitationJobInterval = 30 // 30 minutes
	defer func() {
		// Restore the original value after the test
		common.XrdConfig.SanitationJobInterval = oldInterval
	}()

	// Call the function to be tested
	ticker, done := NewSanitationScheduler()

	// Verify that ticker and done channel were created
	assert.NotNil(t, ticker)
	assert.NotNil(t, done)

	// Stop the ticker
	ticker.Stop()

	// IMPORTANT: Don't try to send to the done channel as it blocks
	// Use a select with timeout instead to avoid test hanging
	select {
	case done <- true:
		// Channel send succeeded
	case <-time.After(10 * time.Millisecond):
		// Timed out, but this is okay - we just want to make sure
		// the test doesn't block indefinitely
	}
}

// TestCheckAndRemoveOldFiles tests the CheckAndRemoveOldFiles function.
func TestCheckAndRemoveOldFiles(t *testing.T) {
	// Create a temporary directory for testing
	dir, err := os.MkdirTemp("", "staging")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Store the original config values
	originalPath := common.XrdConfig.StagingPath
	originalPrefix := common.XrdConfig.StagingTmpDirPrefix

	// Set up the common.XrdConfig for testing
	common.XrdConfig.StagingPath = dir
	common.XrdConfig.StagingTmpDirPrefix = "tmp_"

	// Restore the original values after the test
	defer func() {
		common.XrdConfig.StagingPath = originalPath
		common.XrdConfig.StagingTmpDirPrefix = originalPrefix
	}()

	// Create a temporary subdirectory
	tmpDir := filepath.Join(dir, "tmp_test")
	if err := os.Mkdir(tmpDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Set the modification time to 25 hours ago (to ensure it's older than threshold)
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(tmpDir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Run the CheckAndRemoveOldFiles function
	CheckAndRemoveOldFiles()

	// Check if the directory was removed
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err), "Expected directory %s to be removed", tmpDir)
}

// TestSanitationJob tests the SanitationJob function.
func TestSanitationJob(t *testing.T) {
	// Create a buffered channel to prevent blocking
	done := make(chan bool, 1)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// Use a timeout to ensure test doesn't hang
	go func() {
		// Send done signal with timeout protection
		select {
		case done <- true:
			// Signal sent
		case <-time.After(100 * time.Millisecond):
			t.Logf("Timed out sending to done channel")
		}
	}()

	// Run the SanitationJob function with a timeout
	finished := make(chan bool)
	go func() {
		SanitationJob(ticker, done)
		finished <- true
	}()

	// Wait for job to finish with timeout
	select {
	case <-finished:
		// Test completed normally
	case <-time.After(1 * time.Second):
		t.Fatal("SanitationJob test timed out")
	}
}
