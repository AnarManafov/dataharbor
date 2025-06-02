package middleware

import (
	"github.com/AnarManafov/dataharbor/app/util"

	"github.com/gin-gonic/gin"
)

func TraceMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tid := ctx.GetHeader("X-Tid")
		if len(tid) == 0 {
			tid = util.NextUid()
		}

		ctx.Set("tid", tid)

		ctx.Next()
	}

}
