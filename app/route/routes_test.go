package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AnarManafov/data_lake_ui/app/controller"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	RegisterRoutes(r)

	tests := []struct {
		method   string
		endpoint string
		handler  gin.HandlerFunc
	}{
		{"GET", "/health", controller.Health},
		{"GET", "/initial_dir", controller.FetchInitialDir},
		{"GET", "/host_name", controller.FetchHostName},
		{"POST", "/dir", controller.FetchDirItems},
		{"POST", "/dir/page", controller.FetchDirItemsByPage},
		{"POST", "/stage_file", controller.FetchFileStagedForDownload},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest(tt.method, tt.endpoint, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}
