package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

// RequestID adds a stable request ID to each request and response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = newRequestID()
		}

		c.Set("request_id", requestID)
		c.Writer.Header().Set(requestIDHeader, requestID)
		c.Next()
	}
}

// Logger writes structured request logs through slog.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		requestID, _ := c.Get("request_id")
		logger.Info("http request",
			slog.String("request_id", stringify(requestID)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int("bytes", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
			slog.Duration("duration", time.Since(startedAt)),
		)
	}
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(bytes[:])
}

func stringify(value any) string {
	if stringValue, ok := value.(string); ok {
		return stringValue
	}

	return ""
}
