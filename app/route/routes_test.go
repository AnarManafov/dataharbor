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
		{"GET", "/initial_dir", controller.GetInitialDir},
		{"GET", "/host_name", controller.GetHostName},
		{"POST", "/dir", controller.GetDirItems},
		{"POST", "/dir/page", controller.GetDirItemsByPage},
		{"POST", "/stage_file", controller.GetFileStagedForDownload},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest(tt.method, tt.endpoint, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}
