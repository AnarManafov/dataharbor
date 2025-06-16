package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetXRDClient(t *testing.T) {
	// Test that GetXRDClient returns a valid client
	client := GetXRDClient()
	assert.NotNil(t, client, "Expected XRD client to be initialized")
	assert.NotNil(t, client.Logger, "Expected logger to be initialized")
}
