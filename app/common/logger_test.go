// logger_test.go
package common

import (
	"bytes"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitLogger_Default(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()

	// Initialize the logger
	InitLogger()

	// Check if Logger is initialized
	if Logger == nil {
		t.Fatal("Expected Logger to be initialized, but it was nil")
	}
}

func TestInitLogger_Custom(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Setup custom logger configuration
	viper.Set("logger.custom", map[string]interface{}{
		"type":   "file",
		"path":   "test.log",
		"driver": "file",
	})

	// Initialize the logger
	InitLogger()

	// Check if Logger is initialized
	if Logger == nil {
		t.Fatal("Expected Logger to be initialized, but it was nil")
	}

	// Clean up
	os.Remove("test.log")
}

func TestInitLogger_Console(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Setup custom logger configuration
	viper.Set("logger.console", map[string]interface{}{
		"driver": "console",
		"level":  "info",
	})

	// Initialize the logger
	InitLogger()

	// Check if Logger is initialized
	if Logger == nil {
		t.Fatal("Expected Logger to be initialized, but it was nil")
	}
}

func TestInitLogger_Reinitialization(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Initialize the logger
	InitLogger()

	// Reinitialize the logger
	InitLogger()

	// Check if Logger is still initialized
	if Logger == nil {
		t.Fatal("Expected Logger to be initialized after re-initialization, but it was nil")
	}
}

func TestInitLogger_InvalidConfig(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Setup invalid logger configuration
	viper.Set("logger.custom", map[string]interface{}{
		"type": "invalid",
	})

	t.Log("DEBUG")
	// Initialize the logger
	InitLogger()

	// Check if Logger is nil or if an error is logged
	if Logger != nil {
		t.Fatal("Expected Logger to be nil with invalid configuration, but it was initialized")
	}
}

func TestInitLogger_EnvConfig(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Set environment variables for logger configuration
	os.Setenv("LOGGER_TYPE", "file")
	os.Setenv("LOGGER_PATH", "env_test.log")
	os.Setenv("LOGGER_DRIVER", "file")

	// Print environment variables to verify
	t.Logf("LOGGER_TYPE: %s", os.Getenv("LOGGER_TYPE"))
	t.Logf("LOGGER_PATH: %s", os.Getenv("LOGGER_PATH"))
	t.Logf("LOGGER_DRIVER: %s", os.Getenv("LOGGER_DRIVER"))

	// Initialize the logger
	InitLogger()

	// Print logger configuration to verify
	t.Logf("Logger configuration: %s", viper.GetStringMap("logger"))

	// Check if Logger is initialized
	if Logger == nil {
		t.Fatal("Expected Logger to be initialized with environment variables, but it was nil")
	}

	// Clean up
	os.Remove("env_test.log")
	os.Unsetenv("LOGGER_TYPE")
	os.Unsetenv("LOGGER_PATH")
	os.Unsetenv("LOGGER_DRIVER")
}

func TestLogger_Output(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	writer := zapcore.AddSync(&buf)

	// Create a custom zapcore.Core that writes to the buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		writer,
		zapcore.DebugLevel,
	)

	// Initialize the logger with the custom core
	Logger = zap.New(core).Sugar()

	// Log a message
	Logger.Info("Test message")

	// Use testify to check if the buffer contains the expected message
	assert.Contains(t, buf.String(), "Test message", "Expected log output to contain 'Test message'")
}

func TestLogger_Levels(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	writer := zapcore.AddSync(&buf)

	// Create a custom zapcore.Core that writes to the buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		writer,
		zapcore.DebugLevel,
	)

	// Initialize the logger with the custom core
	Logger = zap.New(core).Sugar()

	// Create a mock gin.Context
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Log messages at different levels
	Debugf(c, "Debug message %d", 1)
	Infof(c, "Info message %d", 2)
	Warnf(c, "Warn message %d", 3)
	Errorf(c, "Error message %d", 4)

	// Use testify to check if the buffer contains the expected messages
	assert.Contains(t, buf.String(), "Debug message", "Expected log output to contain 'Debug message'")
	assert.Contains(t, buf.String(), "Info message", "Expected log output to contain 'Info message'")
	assert.Contains(t, buf.String(), "Warn message", "Expected log output to contain 'Warn message'")
	assert.Contains(t, buf.String(), "Error message", "Expected log output to contain 'Error message'")
}

func TestParseConsoleConf(t *testing.T) {
	// Initialize viper and set configuration values
	cnf := viper.New()
	cnf.Set("level", "info")

	// Call the parseConsoleConf function
	core := parseConsoleConf(cnf)

	// Assertions
	if core == nil {
		t.Fatalf("Expected zapcore.Core, got nil")
	}

	// Check if the core is configured correctly
	// This is a bit tricky since zapcore.Core doesn't expose its configuration directly
	// We can only check if it's not nil and assume it's configured correctly based on the input
	// Further checks would require more complex reflection or integration tests
}

func TestConcatTid(t *testing.T) {
	// Define test cases
	testCases := []struct {
		tid      string
		template string
		expected string
	}{
		{"12345", "template", " tid: 12345 template"},
		{"", "template", "template"},
		{"abcde", "template", " tid: abcde template"},
	}

	// Create a mock gin.Context
	gin.SetMode(gin.TestMode)
	for _, tc := range testCases {
		c, _ := gin.CreateTestContext(nil)
		c.Set("tid", tc.tid)

		// Run test cases
		result := concatTid(c, tc.template)
		if result != tc.expected {
			t.Errorf("concatTid(%q, %q) = %q; expected %q", tc.tid, tc.template, result, tc.expected)
		}
	}
}
