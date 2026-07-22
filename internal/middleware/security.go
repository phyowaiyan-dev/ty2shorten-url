package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxRequestBodyBytes = 32 << 20

// SecurityHeaders adds baseline browser security headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		headers.Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; img-src 'self' data:; style-src 'self'; script-src 'self'")

		c.Next()
	}
}

// LimitRequestBody caps request bodies before handlers read them.
func LimitRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
		c.Next()
	}
}
