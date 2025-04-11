package common

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestParseSystemConfig(t *testing.T) {
	// Set up test configuration
	viper.Reset()
	viper.Set("server.port", 22000)
	viper.Set("server.debug", true)

	// Call the function to be tested
	ParseSystemConfig()

	// Assert the results
	assert.Equal(t, 22000, ServerConfig.Port, "Expected Port to be 22000, but got %d", ServerConfig.Port)
	assert.Equal(t, true, ServerConfig.Debug, "Expected Debug to be true, but got %v", ServerConfig.Debug)

	// Reset for other tests
	ServerConfig = ServerConfigType{}
}
