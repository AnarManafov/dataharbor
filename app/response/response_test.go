package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	Success(ctx, "test data")

	assert.Equal(t, http.StatusOK, w.Code)
	expectedBody := `{"code":200,"data":"test data","message":"success"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestParamValidateFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	ParamValidateFail(ctx, "validation failed")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	expectedBody := `{"code":422,"error":"validation failed"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	Fail(ctx, "failure message", 500)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	expectedBody := `{"code":500,"error":"failure message"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestValidateFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	data := map[string][]string{"field": {"error"}}
	ValidateFail(ctx, data)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	expectedBody := `{"code":422,"data":{"field":["error"]},"error":"Unprocessable Entity"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestFailWithErr(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	err := TransferProtocolError{code: 500, message: "internal error"}
	FailWithErr(ctx, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	expectedBody := `{"code":500,"error":"internal error"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		status       int
		message      string
		expectedBody string
	}{
		{
			name:         "bad request error",
			status:       http.StatusBadRequest,
			message:      "Invalid input",
			expectedBody: `{"code":400,"error":"Invalid input"}`,
		},
		{
			name:         "unauthorized error",
			status:       http.StatusUnauthorized,
			message:      "Not authenticated",
			expectedBody: `{"code":401,"error":"Not authenticated"}`,
		},
		{
			name:         "internal server error",
			status:       http.StatusInternalServerError,
			message:      "Something went wrong",
			expectedBody: `{"code":500,"error":"Something went wrong"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			Error(ctx, tt.status, tt.message)

			assert.Equal(t, tt.status, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	data := map[string]string{"field": "error message"}
	ValidationError(ctx, data)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	expectedBody := `{"code":422,"data":{"field":"error message"},"error":"Unprocessable Entity"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestErrorWithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		httpStatus   int
		code         int
		message      string
		expectedBody string
	}{
		{
			name:         "custom code with bad request",
			httpStatus:   http.StatusBadRequest,
			code:         1001,
			message:      "Custom error",
			expectedBody: `{"code":1001,"error":"Custom error"}`,
		},
		{
			name:         "different http status and code",
			httpStatus:   http.StatusConflict,
			code:         2002,
			message:      "Resource conflict",
			expectedBody: `{"code":2002,"error":"Resource conflict"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			ErrorWithCode(ctx, tt.httpStatus, tt.code, tt.message)

			assert.Equal(t, tt.httpStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		status       int
		data         interface{}
		expectedBody string
	}{
		{
			name:         "simple string data",
			status:       http.StatusOK,
			data:         "hello",
			expectedBody: `"hello"`,
		},
		{
			name:         "map data",
			status:       http.StatusCreated,
			data:         map[string]string{"key": "value"},
			expectedBody: `{"key":"value"}`,
		},
		{
			name:   "struct data",
			status: http.StatusOK,
			data: struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}{Name: "test", Count: 42},
			expectedBody: `{"name":"test","count":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			JSON(ctx, tt.status, tt.data)

			assert.Equal(t, tt.status, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}
