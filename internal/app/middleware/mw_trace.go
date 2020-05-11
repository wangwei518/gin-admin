package middleware

import (
	icontext "github.com/wangwei518/gin-admin/internal/app/context"
	"github.com/wangwei518/gin-admin/pkg/logger"
	"github.com/wangwei518/gin-admin/pkg/util"
	"github.com/gin-gonic/gin"
)

// TraceMiddleware 跟踪ID中间件
func TraceMiddleware(skippers ...SkipperFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if SkipHandler(c, skippers...) {
			c.Next()
			return
		}

		// 优先从请求头中获取请求ID，如果没有则使用UUID
		traceID := c.GetHeader("X-Request-Id")
		if traceID == "" {
			traceID = util.NewTraceID()
		}

		ctx := icontext.NewTraceID(c.Request.Context(), traceID)
		ctx = logger.NewTraceIDContext(ctx, traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
