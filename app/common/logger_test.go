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

	"github.com/AnarManafov/dataharbor/app/config"
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
	viper.Set("logger.custom", map[string]any{
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
	_ = os.Remove("test.log")
}

func TestInitLogger_Console(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Setup custom logger configuration
	viper.Set("logger.console", map[string]any{
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

	// Create invalid logger configuration
	invalidConfig := &config.LoggingConfig{
		Level: "invalid_level", // This will fallback to info level
		Console: config.ConsoleConfig{
			Enabled: false,
		},
		File: config.FileConfig{
			Enabled: false,
		},
	}

	// Initialize the logger with invalid config
	InitLogger(invalidConfig)

	// Check if Logger is initialized (it should be, even with invalid config)
	// Our new implementation provides a fallback development logger
	if Logger == nil {
		t.Fatal("Expected Logger to be initialized even with invalid configuration, but it was nil")
	}
}

func TestInitLogger_EnvConfig(t *testing.T) {
	// Clear any existing logger configurations
	viper.Reset()
	DestroyLogger()

	// Set environment variables for logger configuration
	_ = os.Setenv("LOGGER_TYPE", "file")
	_ = os.Setenv("LOGGER_PATH", "env_test.log")
	_ = os.Setenv("LOGGER_DRIVER", "file")

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
	_ = os.Remove("env_test.log")
	_ = os.Unsetenv("LOGGER_TYPE")
	_ = os.Unsetenv("LOGGER_PATH")
	_ = os.Unsetenv("LOGGER_DRIVER")
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
	// This test is deprecated as parseConsoleConf function was removed
	// in favor of createConsoleCore which requires the full config structure
	t.Skip("parseConsoleConf function has been removed in the unified configuration refactor")
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

func TestConcatTid_NilContext(t *testing.T) {
	// Test with nil context
	result := concatTid(nil, "template")
	assert.Equal(t, "template", result, "concatTid with nil context should return template unchanged")
}

func TestGetZapLevel(t *testing.T) {
	testCases := []struct {
		levelName string
		expected  zapcore.Level
	}{
		{"debug", zap.DebugLevel},
		{"info", zap.InfoLevel},
		{"warn", zap.WarnLevel},
		{"error", zap.ErrorLevel},
		{"invalid", zapcore.InvalidLevel},
		{"", zapcore.InvalidLevel},
		{"INFO", zapcore.InvalidLevel}, // Case-sensitive
	}

	for _, tc := range testCases {
		t.Run(tc.levelName, func(t *testing.T) {
			result := getZapLevel(tc.levelName)
			assert.Equal(t, tc.expected, result, "getZapLevel(%q) should return %v", tc.levelName, tc.expected)
		})
	}
}

func TestCreateConsoleCore(t *testing.T) {
	testCases := []struct {
		name   string
		config *config.LoggingConfig
	}{
		{
			name: "json format with console level",
			config: &config.LoggingConfig{
				Level: "info",
				Console: config.ConsoleConfig{
					Enabled: true,
					Format:  "json",
					Level:   "debug",
				},
			},
		},
		{
			name: "text format with fallback to global level",
			config: &config.LoggingConfig{
				Level: "warn",
				Console: config.ConsoleConfig{
					Enabled: true,
					Format:  "text",
					Level:   "", // Empty level, should fallback to global
				},
			},
		},
		{
			name: "console format (default)",
			config: &config.LoggingConfig{
				Level: "error",
				Console: config.ConsoleConfig{
					Enabled: true,
					Format:  "console",
					Level:   "info",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			core := createConsoleCore(tc.config)
			assert.NotNil(t, core, "createConsoleCore should return a non-nil core")
		})
	}
}

func TestCreateFileCore(t *testing.T) {
	// Create a temporary directory for log files
	tmpDir := t.TempDir()

	testCases := []struct {
		name   string
		config *config.LoggingConfig
	}{
		{
			name: "json format with file level",
			config: &config.LoggingConfig{
				Level: "info",
				File: config.FileConfig{
					Enabled:    true,
					Filename:   tmpDir + "/test_json.log",
					Format:     "json",
					Level:      "debug",
					MaxSize:    10,
					MaxBackups: 3,
					MaxAge:     7,
					Compress:   false,
				},
			},
		},
		{
			name: "text format with fallback to global level",
			config: &config.LoggingConfig{
				Level: "warn",
				File: config.FileConfig{
					Enabled:    true,
					Filename:   tmpDir + "/test_text.log",
					Format:     "text",
					Level:      "", // Empty level, should fallback to global
					MaxSize:    5,
					MaxBackups: 2,
					MaxAge:     30,
					Compress:   true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			core := createFileCore(tc.config)
			assert.NotNil(t, core, "createFileCore should return a non-nil core")
		})
	}
}

func TestInitLoggerFromViper(t *testing.T) {
	// Clear any existing logger
	DestroyLogger()

	// Call InitLoggerFromViper
	InitLoggerFromViper()

	// Check that logger is initialized
	assert.NotNil(t, Logger, "InitLoggerFromViper should initialize Logger")
}

func TestInitLoggerWithConfig_NilConfig(t *testing.T) {
	DestroyLogger()

	// Initialize with nil config
	initLoggerWithConfig(nil)

	// Logger should be initialized with development logger
	assert.NotNil(t, Logger, "initLoggerWithConfig with nil should initialize Logger")
}

func TestInitLoggerWithConfig_ConsoleAndFile(t *testing.T) {
	DestroyLogger()
	tmpDir := t.TempDir()

	cfg := &config.LoggingConfig{
		Level: "info",
		Console: config.ConsoleConfig{
			Enabled: true,
			Format:  "text",
			Level:   "debug",
		},
		File: config.FileConfig{
			Enabled:    true,
			Filename:   tmpDir + "/combined.log",
			Format:     "json",
			Level:      "info",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     7,
		},
	}

	initLoggerWithConfig(cfg)

	assert.NotNil(t, Logger, "initLoggerWithConfig with console and file should initialize Logger")
}

func TestInitLoggerWithConfig_NoOutputsEnabled(t *testing.T) {
	DestroyLogger()

	cfg := &config.LoggingConfig{
		Level: "info",
		Console: config.ConsoleConfig{
			Enabled: false,
		},
		File: config.FileConfig{
			Enabled: false,
		},
	}

	initLoggerWithConfig(cfg)

	// Should fallback to development logger
	assert.NotNil(t, Logger, "initLoggerWithConfig with no outputs should still initialize Logger")
}
