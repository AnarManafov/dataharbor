package config

import (
	"flag"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestFlagSet returns a private FlagSet so the tests never replace the
// process-wide flag.CommandLine (it also carries the test binary's -test.*
// flags; replacing it panics on `go test -count=N`).
func newTestFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("cmd", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// keepConfigFile restores the ConfigFile global that initCmd sets.
func keepConfigFile(t *testing.T) {
	t.Helper()
	orig := ConfigFile
	t.Cleanup(func() { ConfigFile = orig })
}

// TestInitCmd_Version tests the version flag
func TestInitCmd_Version(t *testing.T) {
	keepConfigFile(t)

	// Save original values
	origVersion := Version
	origBuildTime := BuildTime
	origGitCommit := GitCommit
	t.Cleanup(func() {
		Version = origVersion
		BuildTime = origBuildTime
		GitCommit = origGitCommit
	})

	// Set test values
	Version = "1.2.3"
	BuildTime = "2024-01-01T00:00:00Z"
	GitCommit = "abc123def456"

	shouldContinue := initCmd(newTestFlagSet(), []string{"--version"})
	assert.False(t, shouldContinue, "InitCmd should return false when --version is passed")
}

// TestInitCmd_NoFlags tests InitCmd without flags
func TestInitCmd_NoFlags(t *testing.T) {
	keepConfigFile(t)

	shouldContinue := initCmd(newTestFlagSet(), nil)
	assert.True(t, shouldContinue, "InitCmd should return true when no flags are passed")
	assert.Equal(t, "./config/application.yaml", ConfigFile, "ConfigFile should be set to default value")
}

// TestInitCmd_ConfigFlag tests custom config path
func TestInitCmd_ConfigFlag(t *testing.T) {
	keepConfigFile(t)
	customPath := "/custom/path/config.yaml"

	shouldContinue := initCmd(newTestFlagSet(), []string{"--config", customPath})
	assert.True(t, shouldContinue, "InitCmd should return true with --config flag")
	assert.Equal(t, customPath, ConfigFile, "ConfigFile should be set to custom path")
}

// TestInitCmd_ConfigFlagShort tests short form of config flag
func TestInitCmd_ConfigFlagShort(t *testing.T) {
	keepConfigFile(t)
	customPath := "/another/config.yaml"

	shouldContinue := initCmd(newTestFlagSet(), []string{"-config", customPath})
	assert.True(t, shouldContinue, "InitCmd should return true with -config flag")
	assert.Equal(t, customPath, ConfigFile, "ConfigFile should be set to custom path")
}

// TestInitCmd_UnknownFlag tests that a parse error stops start-up
func TestInitCmd_UnknownFlag(t *testing.T) {
	keepConfigFile(t)
	ConfigFile = "unchanged"

	shouldContinue := initCmd(newTestFlagSet(), []string{"--no-such-flag"})
	assert.False(t, shouldContinue, "InitCmd should return false on an unknown flag")
	assert.Equal(t, "unchanged", ConfigFile, "ConfigFile must not change on a parse error")
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
