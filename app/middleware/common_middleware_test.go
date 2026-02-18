package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

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
