package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Throttle caps the number of in-flight requests. Excess requests are rejected
// with 429 instead of blocking, matching chi's middleware.Throttle behavior.
func Throttle(max int) gin.HandlerFunc {
	sem := make(chan struct{}, max)

	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			c.AbortWithStatus(http.StatusTooManyRequests)
		}
	}
}
