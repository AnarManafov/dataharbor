package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		expectedCode int
		expectedBody gin.H
	}{
		{
			name:         "health check",
			expectedCode: http.StatusOK,
			expectedBody: gin.H{
				"code": 200,
				"data": "ok",
				"msg":  "success",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Call the Health function
			Health(c)

			// Convert actual response to expected type
			var actualBody gin.H
			if err := json.Unmarshal(w.Body.Bytes(), &actualBody); err != nil {
				t.Fatalf("Failed to unmarshal response body: %v", err)
			}

			// Convert code field to int
			if code, ok := actualBody["code"].(float64); ok {
				actualBody["code"] = int(code)
			}

			// Assert equality
			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Equal(t, tt.expectedBody, actualBody)
		})
	}
}
