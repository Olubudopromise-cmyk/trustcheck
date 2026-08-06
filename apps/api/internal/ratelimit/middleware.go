package ratelimit

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RateLimit returns middleware that enforces the limiter per client IP.
// Requests over the limit receive a 429 response with a Retry-After header
// indicating the number of seconds until the client may try again. The
// middleware is only registered on the routes it should protect (POST
// /verify); /health, /docs, and /openapi.yaml are unaffected.
func RateLimit(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retryAfter := l.Allow(c.ClientIP())
		if !ok {
			seconds := int64(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.FormatInt(seconds, 10))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}
