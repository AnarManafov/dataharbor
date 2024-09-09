package middleware

import (
	"fmt"

	"github.com/AnarManafov/data_lake_ui/app/response"
	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				response.Fail(c, fmt.Sprint(err), 400)
				return
			}
		}()
		c.Next()
	}
}
