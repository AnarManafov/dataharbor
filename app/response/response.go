package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Response(ctx *gin.Context, httpStatus int, code int, data interface{}, msg map[string]string) {
	response := gin.H{"code": code, "data": data}
	if errorMsg, exists := msg["error"]; exists {
		response["error"] = errorMsg
	} else if msgMsg, exists := msg["message"]; exists {
		response["msg"] = msgMsg
	}
	ctx.JSON(httpStatus, response)
}

func Success(ctx *gin.Context, data interface{}) {
	Response(ctx, http.StatusOK, http.StatusOK, data, map[string]string{"message": "success"})
}

func ParamValidateFail(ctx *gin.Context, msg string) {
	Response(ctx, http.StatusOK, http.StatusUnprocessableEntity, nil, map[string]string{"error": msg})
}

func Fail(ctx *gin.Context, msg string, errcode int) {
	Response(ctx, http.StatusOK, errcode, nil, map[string]string{"error": msg})
}

func ValidateFail(ctx *gin.Context, data map[string][]string) {
	Response(ctx, http.StatusUnprocessableEntity, 422, data, map[string]string{"error": "params validation error"})
}

func FailWithErr(ctx *gin.Context, err BusErr) {
	Response(ctx, http.StatusOK, err.code, nil, map[string]string{"error": err.message})
}
