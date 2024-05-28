package controller

import (
	"github.com/AnarManafov/app/response"

	"github.com/gin-gonic/gin"
)

func Health(ctx *gin.Context) {
	response.Success(ctx, "ok")
}
