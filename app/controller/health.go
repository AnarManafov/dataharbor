package controller

import (
	"github.com/AnarManafov/dataharbor/app/response"

	"github.com/gin-gonic/gin"
)

// HealthCheck provides a simple endpoint to verify the API is up and running
// This function was renamed from Health to avoid conflict with auth.go
func HealthCheck(ctx *gin.Context) {
	response.Success(ctx, "ok")
}
