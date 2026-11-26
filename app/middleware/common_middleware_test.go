package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

func setupTestLogger() {
	testConfig := &config.LoggingConfig{
		Level: "info",
		Console: config.ConsoleConfig{
			Enabled: true,
			Format:  "text",
			Level:   "debug",
		},
		File: config.FileConfig{
			Enabled: false,
		},
	}
	common.InitLogger(testConfig)
}

func TestRecovery_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestLogger()

	router := gin.New()
	router.Use(Recovery())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

func TestRecovery_WithPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestLogger()

	router := gin.New()
	router.Use(Recovery())

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTraceRequest_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestLogger()

	router := gin.New()
	router.Use(TraceRequest())

	var capturedTid string
	router.GET("/test", func(c *gin.Context) {
		capturedTid = c.GetString("tid")
		c.String(http.StatusOK, "success")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedTid, "Trace ID should be set")
	// UUID format check (basic)
	assert.Len(t, capturedTid, 36, "Trace ID should be UUID format")
}

func TestTraceRequest_ErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestLogger()

	router := gin.New()
	router.Use(TraceRequest())

	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "error")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTraceRequest_UniqueTraceIds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestLogger()

	router := gin.New()
	router.Use(TraceRequest())

	tids := make([]string, 0)
	router.GET("/test", func(c *gin.Context) {
		tids = append(tids, c.GetString("tid"))
		c.String(http.StatusOK, "success")
	})

	// Make multiple requests
	for range 5 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}

	// Check all trace IDs are unique
	tidMap := make(map[string]bool)
	for _, tid := range tids {
		assert.False(t, tidMap[tid], "Trace IDs should be unique")
		tidMap[tid] = true
	}
}

func TestRecovery_DoesNotLeakCredentials(t *testing.T) {
	const sentinel = "SENTINEL-SECRET-9f3a"

	prevMode := gin.Mode()
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(prevMode)

	prevWriter := gin.DefaultErrorWriter
	errBuf := &bytes.Buffer{}
	gin.DefaultErrorWriter = errBuf
	defer func() { gin.DefaultErrorWriter = prevWriter }()

	prevLogger := common.Logger
	core, logs := observer.New(zapcore.DebugLevel)
	common.Logger = zap.New(core).Sugar()
	defer func() { common.Logger = prevLogger }()

	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic?access_token="+sentinel, nil)
	req.Header.Set("Cookie", "sid="+sentinel)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, errBuf.String(), sentinel)

	var zapOut strings.Builder
	for _, e := range logs.All() {
		zapOut.WriteString(e.Message)
		for _, f := range e.Context {
			fmt.Fprint(&zapOut, f.String, f.Interface)
		}
	}
	assert.NotContains(t, zapOut.String(), sentinel)
	assert.Contains(t, zapOut.String(), "Panic recovered")
}
