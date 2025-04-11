package common

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ServerConfigType contains server configuration
type ServerConfigType struct {
	Debug bool
	Port  int
}

// ServerConfig holds server configuration
var ServerConfig ServerConfigType

// ParseSystemConfig parses system configuration from viper
func ParseSystemConfig() {
	// Parse server config
	ServerConfig.Debug = viper.GetBool("server.debug")
	ServerConfig.Port = viper.GetInt("server.port")
	if ServerConfig.Port == 0 {
		// Default port if not specified
		ServerConfig.Port = 8080
	}

	// Log current configuration
	if Logger != nil {
		Logger.Infof("Server configuration - Debug: %v, Port: %d", ServerConfig.Debug, ServerConfig.Port)
	}
}

// GetLogger returns the global logger
func GetLogger() *zap.SugaredLogger {
	// Return logger if already initialized
	if Logger != nil {
		return Logger
	}

	// Otherwise initialize a basic logger
	logger, _ := zap.NewProduction()
	Logger = logger.Sugar()
	return Logger
}
