package middleware

import (
	"os"
	"testing"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
)

func TestMain(m *testing.M) {
	// Initialize the logger and the configuration
	common.InitLogger()
	config.InitCmd()

	// Run the tests
	exitCode := m.Run()

	// Exit with the appropriate status code
	os.Exit(exitCode)
}
