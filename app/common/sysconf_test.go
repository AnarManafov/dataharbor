package common

import (
	"testing"

	"github.com/spf13/viper"
)

func TestParseSystemConfig(t *testing.T) {
	// Reset viper configuration to ensure test isolation
	viper.Reset()

	// Call the function to parse the system config
	ParseSystemConfig()

	// Check if the default port is set correctly
	if ServerConfig.Port != 22000 {
		t.Errorf("Expected Port to be 22000, but got %d", ServerConfig.Port)
	}

	// Check if the default debug value is set correctly
	if !ServerConfig.Debug {
		t.Errorf("Expected Debug to be true, but got %v", ServerConfig.Debug)
	}
}
