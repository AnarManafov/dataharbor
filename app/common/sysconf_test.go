package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLogger(t *testing.T) {
	// Test that GetLogger returns a valid logger
	logger := GetLogger()
	assert.NotNil(t, logger, "Expected logger to be initialized")
}

func TestGetLogger_WhenLoggerIsNil(t *testing.T) {
	// Save current logger
	originalLogger := Logger

	// Set logger to nil
	Logger = nil

	// GetLogger should create a new logger
	result := GetLogger()

	assert.NotNil(t, result, "GetLogger should return a non-nil logger")
	assert.NotNil(t, Logger, "Logger global should be initialized")

	// Restore original logger
	Logger = originalLogger
}

func TestGetLogger_WhenLoggerIsInitialized(t *testing.T) {
	// Initialize logger first
	InitLogger()

	// GetLogger should return the same logger
	logger1 := GetLogger()
	logger2 := GetLogger()

	assert.NotNil(t, logger1)
	assert.NotNil(t, logger2)
	assert.Equal(t, logger1, logger2, "GetLogger should return the same logger instance")
}

func TestGetLogger_ReturnsUsableLogger(t *testing.T) {
	logger := GetLogger()

	// Should not panic when using the logger
	assert.NotPanics(t, func() {
		logger.Info("test message")
		logger.Debug("debug message")
		logger.Warn("warning message")
	})
}
