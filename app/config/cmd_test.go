package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInitCmd_Version tests the version flag
func TestInitCmd_Version(t *testing.T) {
	// Save original values
	origVersion := Version
	origBuildTime := BuildTime
	origGitCommit := GitCommit
	origArgs := os.Args

	// Set test values
	Version = "1.2.3"
	BuildTime = "2024-01-01T00:00:00Z"
	GitCommit = "abc123def456"

	defer func() {
		Version = origVersion
		BuildTime = origBuildTime
		GitCommit = origGitCommit
		os.Args = origArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Simulate --version flag
	os.Args = []string{"cmd", "--version"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	shouldContinue := InitCmd()
	assert.False(t, shouldContinue, "InitCmd should return false when --version is passed")
}

// TestInitCmd_NoFlags tests InitCmd without flags
func TestInitCmd_NoFlags(t *testing.T) {
	// Save original values
	origArgs := os.Args
	origConfigFile := ConfigFile

	defer func() {
		os.Args = origArgs
		ConfigFile = origConfigFile
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Simulate no flags
	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	shouldContinue := InitCmd()
	assert.True(t, shouldContinue, "InitCmd should return true when no flags are passed")
	assert.Equal(t, "./config/application.yaml", ConfigFile, "ConfigFile should be set to default value")
}

// TestInitCmd_ConfigFlag tests custom config path
func TestInitCmd_ConfigFlag(t *testing.T) {
	// Save original values
	origArgs := os.Args
	origConfigFile := ConfigFile

	defer func() {
		os.Args = origArgs
		ConfigFile = origConfigFile
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Simulate --config flag
	customPath := "/custom/path/config.yaml"
	os.Args = []string{"cmd", "--config", customPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	shouldContinue := InitCmd()
	assert.True(t, shouldContinue, "InitCmd should return true with --config flag")
	assert.Equal(t, customPath, ConfigFile, "ConfigFile should be set to custom path")
}

// TestInitCmd_ConfigFlagShort tests short form of config flag
func TestInitCmd_ConfigFlagShort(t *testing.T) {
	// Save original values
	origArgs := os.Args
	origConfigFile := ConfigFile

	defer func() {
		os.Args = origArgs
		ConfigFile = origConfigFile
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Simulate -config flag (short form)
	customPath := "/another/config.yaml"
	os.Args = []string{"cmd", "-config", customPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	shouldContinue := InitCmd()
	assert.True(t, shouldContinue, "InitCmd should return true with -config flag")
	assert.Equal(t, customPath, ConfigFile, "ConfigFile should be set to custom path")
}

// TestVersionVariables tests that version variables can be set
func TestVersionVariables(t *testing.T) {
	// Save original values
	origVersion := Version
	origBuildTime := BuildTime
	origGitCommit := GitCommit

	defer func() {
		Version = origVersion
		BuildTime = origBuildTime
		GitCommit = origGitCommit
	}()

	// Test that version variables can be modified (would be done by ldflags)
	Version = "test-version"
	BuildTime = "test-time"
	GitCommit = "test-commit"

	assert.Equal(t, "test-version", Version)
	assert.Equal(t, "test-time", BuildTime)
	assert.Equal(t, "test-commit", GitCommit)
}
