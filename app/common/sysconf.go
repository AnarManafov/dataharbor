package common

import (
	"go.uber.org/zap"
)

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
