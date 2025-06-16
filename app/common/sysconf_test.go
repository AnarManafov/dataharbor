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
