package route

import (
	"os"
	"testing"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
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
