package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response structure for standardized API responses
type Response struct {
	Code    int    `json:"code"`              // Internal code (typically HTTP status code)
	Data    any    `json:"data,omitempty"`    // Response payload
	Message string `json:"message,omitempty"` // User-friendly message
	Error   string `json:"error,omitempty"`   // Error message if applicable
}

// sendResponse sends a standardized JSON response
func sendResponse(ctx *gin.Context, httpStatus int, code int, data any, message string, errorMsg string) {
	response := Response{
		Code:    code,
		Data:    data,
		Message: message,
		Error:   errorMsg,
	}
	ctx.JSON(httpStatus, response)
}

// Success sends a successful response with data
func Success(ctx *gin.Context, data any) {
	sendResponse(ctx, http.StatusOK, http.StatusOK, data, "success", "")
}

// Error sends an error response with the given status code and message
// This consolidates multiple error response functions into a single, flexible one
func Error(ctx *gin.Context, status int, message string) {
	sendResponse(ctx, status, status, nil, "", message)
}

// ValidationError sends a response for validation failures
func ValidationError(ctx *gin.Context, data any) {
	statusCode := http.StatusUnprocessableEntity
	sendResponse(ctx, statusCode, statusCode, data, "", http.StatusText(statusCode))
}

// ErrorWithCode sends an error response with a custom error code
func ErrorWithCode(ctx *gin.Context, httpStatus int, code int, message string) {
	sendResponse(ctx, httpStatus, code, nil, "", message)
}

// Fail sends a general failure response with a message and error code
// Maintained for backward compatibility with existing code
func Fail(ctx *gin.Context, msg string, errcode int) {
	sendResponse(ctx, http.StatusBadRequest, errcode, nil, "", msg)
}

// ParamValidateFail sends a parameter validation failure response
// Maintained for backward compatibility with existing code
func ParamValidateFail(ctx *gin.Context, msg string) {
	sendResponse(ctx, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, nil, "", msg)
}

// ValidateFail sends a validation failure response with data
// Maintained for backward compatibility with existing code
func ValidateFail(ctx *gin.Context, data map[string][]string) {
	sendResponse(ctx, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, data, "", http.StatusText(http.StatusUnprocessableEntity))
}

// FailWithErr sends a failure response with a custom error type
// Maintained for backward compatibility with existing code
func FailWithErr(ctx *gin.Context, err TransferProtocolError) {
	sendResponse(ctx, http.StatusBadRequest, err.code, nil, "", err.message)
}

// JSON sends a raw JSON response without using the standard Response structure
// Use this only for special cases where the standard Response format doesn't fit
func JSON(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, data)
}
