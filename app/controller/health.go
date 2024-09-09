package controller

import (
	"github.com/AnarManafov/data_lake_ui/app/response"

	"github.com/gin-gonic/gin"
)

func Health(ctx *gin.Context) {
	response.Success(ctx, "ok")
}
