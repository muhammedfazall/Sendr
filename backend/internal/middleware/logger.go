package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// loggerKey is the gin context key holding the request-scoped logger.
const loggerKey = "logger"

// Logger returns the request-scoped logger installed by StructuredLogger.
func Logger(c *gin.Context) *slog.Logger {
	if v, ok := c.Get(loggerKey); ok {
		if l, ok := v.(*slog.Logger); ok {
			return l
		}
	}
	return slog.Default()
}

// StructuredLogger emits one structured log line per request.
func StructuredLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		l := logger
		if reqID := c.GetString(RequestIDKey); reqID != "" {
			l = logger.With("request_id", reqID)
		}
		c.Set(loggerKey, l)

		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		bytesWritten := c.Writer.Size()
		if bytesWritten < 0 {
			bytesWritten = 0
		}

		l.LogAttrs(c.Request.Context(), level, "request completed",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Int("bytes", bytesWritten),
			slog.Duration("duration", time.Since(start)),
		)
	}
}
