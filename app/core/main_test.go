package core

import (
	"os"
	"testing"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Initialize logger
	common.InitLogger()

	// Set up test config with sanitation job interval
	testConfig := &config.Config{
		XRD: config.XRDConfig{
			SanitationJobInterval: 30,
		},
	}
	config.SetConfig(testConfig)

	// Run tests with timeout protection
	os.Exit(m.Run())
}
