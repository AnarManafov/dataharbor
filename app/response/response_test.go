package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	msg := map[string]string{"message": "test message"}
	Response(ctx, http.StatusOK, 200, "test data", msg)

	assert.Equal(t, http.StatusOK, w.Code)
	expectedBody := `{"code":200,"data":"test data","msg":"test message"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	Success(ctx, "test data")

	assert.Equal(t, http.StatusOK, w.Code)
	expectedBody := `{"code":200,"data":"test data","msg":"success"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestParamValidateFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	ParamValidateFail(ctx, "validation failed")

	assert.Equal(t, http.StatusOK, w.Code)
	expectedBody := `{"code":422,"data":null,"error":"validation failed"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

func TestFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	Fail(ctx, "failure message", 500)

	assert.Equal(t, http.StatusOK, w.Code)
	expectedBody := `{"code":500,"data":null,"error":"failure message"}`
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

	assert.Equal(t, http.StatusOK, w.Code)
	expectedBody := `{"code":500,"data":null,"error":"internal error"}`
	assert.JSONEq(t, expectedBody, w.Body.String())
}
