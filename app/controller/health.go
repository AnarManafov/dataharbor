package controller

import (
	"app/response"

	"github.com/gin-gonic/gin"
)

func Health(ctx *gin.Context) {
	response.Success(ctx, "ok")
}
