package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AnarManafov/data_lake_ui/app/request"
	"github.com/AnarManafov/data_lake_ui/app/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Helper function to convert map to response.DirItemResp
func convertToDirItemResp(item map[string]interface{}) response.DirectoryItemResponse {
	return response.DirectoryItemResponse{
		Name:     item["name"].(string),
		Type:     item["type"].(string),
		DateTime: item["date_time"].(string),
		Size:     uint64(item["size"].(float64)),
	}
}

// Function to convert specified fields from float64 to int
// WORKAROUND: By default, json.Unmarshal treats all numbers in JSON as float64 because JSON itself does not distinguish between integer and floating-point numbers.
// So, we need to convert them to int if they are supposed to be int.
func convertFieldsToInt(data gin.H, fields []string) {
	for _, field := range fields {
		if val, ok := data[field].(float64); ok {
			data[field] = int(val)
		}
	}
}

func TestFetchInitialDir(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FetchInitialDir(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"code":200,"data":"/tmp/","message":"success"}`, w.Body.String())
}

func TestFetchHostName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FetchHostName(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"code":200,"data":"localhost","message":"success"}`, w.Body.String())
}

func TestFetchDirItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		requestBody  request.DirectoryItemsRequest
		mockFiles    []xrdDirEntry
		mockError    error
		expectedCode int
		expectedBody gin.H
	}{
		{
			name: "valid request",
			requestBody: request.DirectoryItemsRequest{
				Path:     "/valid/path",
				PageSize: 500,
			},
			mockFiles: []xrdDirEntry{
				{name: "file1", dt: time.Now(), size: 123, isDir: false},
				{name: "dir1", dt: time.Now(), size: 0, isDir: true},
			},
			expectedCode: http.StatusOK,
			expectedBody: gin.H{
				"code": 200,
				"items": []response.DirectoryItemResponse{
					{Name: "file1", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 123, Type: "file"},
					{Name: "dir1", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 0, Type: "dir"},
				},
				"totalItems":         2,
				"pageSize":           500,
				"totalPages":         1,
				"totalFileCount":     1,
				"totalFolderCount":   1,
				"cumulativeFileSize": 123,
			},
		},
		{
			name: "empty directory path",
			requestBody: request.DirectoryItemsRequest{
				Path: "",
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
				"error": "empty directory path to list",
			},
		},
		{
			name: "directory read error",
			requestBody: request.DirectoryItemsRequest{
				Path: "/valid/path",
			},
			mockError:    errors.New("read error"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: gin.H{
				"code":  400,
				"error": "read error",
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
			c.Request, _ = http.NewRequest(http.MethodPost, "/dir-items", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			// Call the function
			fetchDirItems(c, MockReadDir, "host", 123, false)

			// Convert actual response to expected type
			var actualBody gin.H
			if err := json.Unmarshal(w.Body.Bytes(), &actualBody); err != nil {
				t.Fatalf("Failed to unmarshal response body: %v", err)
			}
			if items, ok := actualBody["items"].([]interface{}); ok {
				var convertedItems []response.DirectoryItemResponse
				for _, item := range items {
					convertedItems = append(convertedItems, convertToDirItemResp(item.(map[string]interface{})))
				}
				actualBody["items"] = convertedItems
			}

			// Convert specified fields to int
			fieldsToConvert := []string{"code", "cumulativeFileSize", "pageSize", "totalFileCount", "totalFolderCount", "totalItems", "totalPages"}
			convertFieldsToInt(actualBody, fieldsToConvert)

			assert.Equal(t, tt.expectedBody, actualBody, "mismatch (-expected +actual)")
		})
	}
}

func TestFetchDirItemsByPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		requestBody  request.DirectoryItemsRequest
		mockFiles    []xrdDirEntry
		mockError    error
		expectedCode int
		expectedBody gin.H
	}{
		{
			name: "valid request",
			requestBody: request.DirectoryItemsRequest{
				Path:     "/valid/path",
				Page:     2,
				PageSize: 6,
			},
			mockFiles: []xrdDirEntry{
				{name: "file1", dt: time.Now(), size: 123, isDir: false},
				{name: "dir1", dt: time.Now(), size: 0, isDir: true},
				{name: "file2", dt: time.Now(), size: 123, isDir: false},
				{name: "dir2", dt: time.Now(), size: 0, isDir: true},
				{name: "file3", dt: time.Now(), size: 123, isDir: false},
				{name: "dir3", dt: time.Now(), size: 0, isDir: true},
				{name: "file4", dt: time.Now(), size: 123, isDir: false},
				{name: "dir4", dt: time.Now(), size: 0, isDir: true},
				{name: "file5", dt: time.Now(), size: 123, isDir: false},
				{name: "dir5", dt: time.Now(), size: 0, isDir: true},
				{name: "file6", dt: time.Now(), size: 123, isDir: false},
				{name: "dir6", dt: time.Now(), size: 0, isDir: true},
			},
			expectedCode: http.StatusOK,
			expectedBody: gin.H{
				"code": 200,
				"items": []response.DirectoryItemResponse{
					{Name: "file4", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 123, Type: "file"},
					{Name: "dir4", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 0, Type: "dir"},
					{Name: "file5", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 123, Type: "file"},
					{Name: "dir5", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 0, Type: "dir"},
					{Name: "file6", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 123, Type: "file"},
					{Name: "dir6", DateTime: time.Now().Format("2006-01-02 15:04:05"), Size: 0, Type: "dir"},
				},
			},
		},
		{
			name: "invalid page number",
			requestBody: request.DirectoryItemsRequest{
				Path: "/valid/path",
				Page: 0,
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
				"error": "invalid page number",
			},
		},
		{
			name: "empty directory path",
			requestBody: request.DirectoryItemsRequest{
				Path: "",
				Page: 1,
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
				"error": "empty directory path to list",
			},
		},
		{
			name: "directory read error",
			requestBody: request.DirectoryItemsRequest{
				Path: "/valid/path",
				Page: 1,
			},
			mockError:    errors.New("read error"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: gin.H{
				"code":  400,
				"error": "read error",
			},
		},
		{
			name: "page number out of range",
			requestBody: request.DirectoryItemsRequest{
				Path: "/valid/path",
				Page: 2,
			},
			mockFiles: []xrdDirEntry{
				{name: "file1", dt: time.Now(), size: 123, isDir: false},
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
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
			fetchDirItems(c, MockReadDir, "host", 123, true)

			// Convert actual response to expected type
			var actualBody gin.H
			if err := json.Unmarshal(w.Body.Bytes(), &actualBody); err != nil {
				t.Fatalf("Failed to unmarshal response body: %v", err)
			}
			if items, ok := actualBody["items"].([]interface{}); ok {
				var convertedItems []response.DirectoryItemResponse
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

func TestFetchFileStagedForDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		requestBody  request.DirectoryItemsRequest
		mockError    error
		expectedCode int
		expectedBody gin.H
	}{
		{
			name: "valid request",
			requestBody: request.DirectoryItemsRequest{
				Path: "/valid/file",
			},
			expectedCode: http.StatusOK,
			expectedBody: gin.H{
				"code": 200,
				"data": map[string]interface{}{
					"path": "/staged/file",
				},
				"message": "success",
			},
		},
		{
			name: "empty file path",
			requestBody: request.DirectoryItemsRequest{
				Path: "",
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: gin.H{
				"code":  400,
				"error": "empty file path for staging",
			},
		},
		{
			name: "staging error",
			requestBody: request.DirectoryItemsRequest{
				Path: "/valid/file",
			},
			mockError:    errors.New("staging error"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: gin.H{
				"code":  400,
				"error": "staging error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock StageFile function
			MockStageFile := func(host string, port uint, path string) (string, error) {
				if tt.mockError != nil {
					return "", tt.mockError
				}
				return "/staged/file", nil
			}

			// Create a new gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create request body
			body, _ := json.Marshal(tt.requestBody)
			c.Request, _ = http.NewRequest(http.MethodPost, "/file-staged-for-download", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			// Call the function
			fetchFileStagedForDownload(c, MockStageFile, "host", 123)

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
			assert.Equal(t, tt.expectedBody, actualBody)
		})
	}
}
