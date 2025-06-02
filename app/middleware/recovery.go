package middleware

import (
	"fmt"
	"net/http"

	"github.com/AnarManafov/dataharbor/app/response"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				response.Fail(c, fmt.Sprint(err), http.StatusBadRequest)
				return
			}
		}()
		c.Next()
	}
}
