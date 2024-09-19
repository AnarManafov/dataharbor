package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AnarManafov/data_lake_ui/app/common"
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
	ticker, done := NewSanitationScheduler()
	if ticker == nil {
		t.Error("Expected ticker to be non-nil")
	}
	if done == nil {
		t.Error("Expected done channel to be non-nil")
	}
	ticker.Stop()
	close(done)
}

// TestCleanStagingDir tests the CleanStagingDir function.
func TestCleanStagingDir(t *testing.T) {
	// Create a temporary directory for testing
	dir, err := os.MkdirTemp("", "staging")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Set up the common.XrdConfig
	common.XrdConfig.StagingPath = dir
	common.XrdConfig.StagingTmpDirPrefix = "tmp_"

	// Create a temporary subdirectory
	tmpDir := filepath.Join(dir, "tmp_test")
	if err := os.Mkdir(tmpDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Set the modification time to 3 hours ago
	oldTime := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(tmpDir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Run the CleanStagingDir function
	CleanStagingDir()

	// Check if the directory was removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Errorf("Expected directory %s to be removed", tmpDir)
	}
}

// TestSanitationJob tests the SanitationJob function.
func TestSanitationJob(t *testing.T) {
	ticker := time.NewTicker(1 * time.Second)
	done := make(chan bool)

	go func() {
		time.Sleep(2 * time.Second)
		done <- true
	}()

	start := time.Now()
	SanitationJob(ticker, done)
	elapsed := time.Since(start)

	if elapsed < 2*time.Second {
		t.Errorf("Expected SanitationJob to run for at least 2 seconds, ran for %v", elapsed)
	}
}
