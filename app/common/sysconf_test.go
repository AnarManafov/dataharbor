package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

// productionLoggerToFile returns a production-config logger writing JSON to a
// temp file, plus a reader for the lines written so far.
func productionLoggerToFile(t *testing.T, build func(zap.Config) (*zap.Logger, error)) (*zap.Logger, func() []string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prod.log")
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{path}
	cfg.ErrorOutputPaths = []string{path}
	l, err := build(cfg)
	if err != nil {
		t.Fatalf("build logger: %v", err)
	}
	return l, func() []string {
		_ = l.Sync()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read log: %v", err)
		}
		return strings.Split(strings.TrimSpace(string(b)), "\n")
	}
}

// TestNewProductionLogger_RedactsAndSamples pins GH-82: the fallback logger
// returned by GetLogger redacts secrets and keeps zap's production sampling.
func TestNewProductionLogger_RedactsAndSamples(t *testing.T) {
	const sentinel = "SENTINELsamplerSecret0123456789"
	const n = 500

	l, lines := productionLoggerToFile(t, newProductionLogger)
	l.Info("upstream rejected Authorization: Bearer " + sentinel)
	got := lines()
	if assert.Len(t, got, 1) {
		assert.NotContains(t, got[0], sentinel)
		assert.Contains(t, got[0], "[REDACTED]")
	}

	// Production sampling: first 100 identical messages per second, then
	// every 100th. n identical entries within one tick must be cut down.
	for i := 0; i < n; i++ {
		l.Info("repeated sampled message")
	}
	sampled := len(lines()) - 1
	assert.Greater(t, sampled, 0)
	assert.Less(t, sampled, n/2, "sampling is not active: %d of %d entries written", sampled, n)

	// Control: the old construction (redacting core wrapped around the
	// sampler) writes every entry, so the assertion above detects the defect.
	old, oldLines := productionLoggerToFile(t, func(cfg zap.Config) (*zap.Logger, error) {
		return cfg.Build(zap.WrapCore(newRedactingCore))
	})
	for i := 0; i < n; i++ {
		old.Info("repeated sampled message")
	}
	assert.Len(t, oldLines(), n)
}
