package core

import (
	"os"
	"testing"

	"github.com/AnarManafov/data_lake_ui/app/common"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Initialize logger
	common.InitLogger()

	// Ensure default sanitation job interval is set to avoid panics
	common.XrdConfig.SanitationJobInterval = 30

	// Run tests with timeout protection
	os.Exit(m.Run())
}
