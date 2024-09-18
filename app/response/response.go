package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response structure
type Response struct {
	// Internal error code. For now we mostly reuse HTTP status codes
	Code int `json:"code"`
	// Response data
	Data interface{} `json:"data,omitempty"`
	// Response message
	Message string `json:"message,omitempty"`
	// Error message
	Error string `json:"error,omitempty"`
}

// Send a JSON response
func sendResponse(ctx *gin.Context, httpStatus int, code int, data interface{}, message string, error string) {
	response := Response{
		Code:    code,
		Data:    data,
		Message: message,
		Error:   error,
	}
	ctx.JSON(httpStatus, response)
}

// Success response
func Success(ctx *gin.Context, data interface{}) {
	sendResponse(ctx, http.StatusOK, http.StatusOK, data, "success", "")
}

// Parameter validation failure response
func ParamValidateFail(ctx *gin.Context, msg string) {
	sendResponse(ctx, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, nil, "", msg)
}

// General failure response
func Fail(ctx *gin.Context, msg string, errcode int) {
	sendResponse(ctx, http.StatusBadRequest, errcode, nil, "", msg)
}

// Validation failure response with data
func ValidateFail(ctx *gin.Context, data map[string][]string) {
	sendResponse(ctx, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, data, "", http.StatusText(http.StatusUnprocessableEntity))
}

// Failure response with custom error
func FailWithErr(ctx *gin.Context, err TransferProtocolError) {
	sendResponse(ctx, http.StatusBadRequest, err.code, nil, "", err.message)
}
