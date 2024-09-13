package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/request"
	"github.com/AnarManafov/data_lake_ui/app/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Initialize the test logger
	common.InitializeTestLogger()

	// Run the tests
	m.Run()
}

// Helper function to convert map to response.DirItemResp
func convertToDirItemResp(item map[string]interface{}) response.DirItemResp {
	return response.DirItemResp{
		Name:     item["name"].(string),
		Type:     item["type"].(string),
		DateTime: item["date_time"].(string),
		Size:     uint64(item["size"].(float64)),
	}
}

func TestGetDirItemsByPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		requestBody  request.DirItemsReq
		mockFiles    []xrdDirEntry
		mockError    error
		expectedCode int
		expectedBody gin.H
	}{
		{
			name: "valid request",
			requestBody: request.DirItemsReq{
				Path: "/valid/path",
				Page: 1,
			},
			mockFiles: []xrdDirEntry{
				{name: "file1", dt: time.Now(), size: 123, isDir: false},
				{name: "dir1", dt: time.Now(), size: 0, isDir: true},
			},
			expectedCode: http.StatusOK,
			expectedBody: gin.H{
				"code": 200,
				"items": []response.DirItemResp{
					{Name: "file1", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 123, Type: "file"},
					{Name: "dir1", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 0, Type: "dir"},
				},
			},
		},
		{
			name: "invalid page number",
			requestBody: request.DirItemsReq{
				Path: "/valid/path",
				Page: 0,
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
				"data":  nil,
				"error": "invalid page number",
			},
		},
		{
			name: "empty directory path",
			requestBody: request.DirItemsReq{
				Path: "",
				Page: 1,
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
				"data":  nil,
				"error": "empty directory path to list",
			},
		},
		{
			name: "directory read error",
			requestBody: request.DirItemsReq{
				Path: "/valid/path",
				Page: 1,
			},
			mockError:    errors.New("read error"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: gin.H{
				"code":  400,
				"data":  nil,
				"error": "read error",
			},
		},
		{
			name: "page number out of range",
			requestBody: request.DirItemsReq{
				Path: "/valid/path",
				Page: 2,
			},
			mockFiles: []xrdDirEntry{
				{name: "file1", dt: time.Now(), size: 123, isDir: false},
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
				"data":  nil,
				"error": "page number out of range",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock ReadDir function
			MockReadDir := func(ctx *gin.Context, host string, port uint, path string) ([]xrdDirEntry, error) {
				return tt.mockFiles, tt.mockError
			}

			// Create a new gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create request body
			body, _ := json.Marshal(tt.requestBody)
			c.Request, _ = http.NewRequest(http.MethodPost, "/dir-items-by-page", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			// Call the function
			_GetDirItemsByPage(c, MockReadDir, "host", 123)

			// Convert actual response to expected type
			var actualBody gin.H
			if err := json.Unmarshal(w.Body.Bytes(), &actualBody); err != nil {
				t.Fatalf("Failed to unmarshal response body: %v", err)
			}
			if items, ok := actualBody["items"].([]interface{}); ok {
				var convertedItems []response.DirItemResp
				for _, item := range items {
					convertedItems = append(convertedItems, convertToDirItemResp(item.(map[string]interface{})))
				}
				actualBody["items"] = convertedItems
			}

			// Convert code field to int
			if code, ok := actualBody["code"].(float64); ok {
				actualBody["code"] = int(code)
			}

			// Assert equality
			assert.Equal(t, tt.expectedBody, actualBody)
		})
	}
}
