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
