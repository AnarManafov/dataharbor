package common

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestParseXrdConfig(t *testing.T) {
	// Set up test configuration
	viper.Reset()
	viper.Set("xrd.host", "localhost")
	viper.Set("xrd.port", 1094)
	viper.Set("xrd.initial_dir", "/tmp/")
	viper.Set("xrd.xrd_client_bin_path", "/usr/bin/")
	viper.Set("xrd.process_timeout", 60)
	viper.Set("xrd.staging_path", "/staging")
	viper.Set("xrd.sanitation_job_interval", 30)
	viper.Set("xrd.staging_tmp_dir_prefix", "stg_")

	// Call the function to be tested
	ParseXrdConfig()

	// Assert the results
	assert.Equal(t, "localhost", XrdConfig.Host, "Expected host to be localhost, but got %s", XrdConfig.Host)
	assert.Equal(t, uint(1094), XrdConfig.Port, "Expected port to be 1094, but got %d", XrdConfig.Port)
	assert.Equal(t, "/tmp/", XrdConfig.InitialDir, "Expected initial_ir to be /tmp/, but got %s", XrdConfig.InitialDir)
	assert.Equal(t, "/usr/bin/", XrdConfig.XrdClientBinPath, "Expected XrdClientBinPath to be /usr/bin/, but got %s", XrdConfig.XrdClientBinPath)
	assert.Equal(t, uint(60), XrdConfig.ProcessTimeout, "Expected process_timeout to be 60, but got %d", XrdConfig.ProcessTimeout)

	// Reset for other tests
	XrdConfig = XrdConfigType{}
}
