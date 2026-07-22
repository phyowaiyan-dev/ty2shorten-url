package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
)

// SetupGate redirects browser traffic to first-run setup until setup is complete.
func SetupGate(setup *services.SetupService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if setupAllowedPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		completed, err := setup.IsCompleted()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "setup status could not be checked"})
			c.Abort()
			return
		}
		if completed {
			c.Next()
			return
		}

		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Redirect(http.StatusSeeOther, "/setup")
			c.Abort()
			return
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "initial setup is required"})
		c.Abort()
	}
}

// AuthRequired protects administrator routes.
func AuthRequired(sessions *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := sessions.AdminID(c); ok {
			c.Next()
			return
		}

		if wantsHTML(c) {
			c.Redirect(http.StatusSeeOther, "/admin/login")
			c.Abort()
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		c.Abort()
	}
}

// CSRFRequired validates state-changing HTML form submissions.
func CSRFRequired(sessions *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sessions.ValidateCSRF(c) {
			c.Next()
			return
		}

		c.String(http.StatusBadRequest, "Invalid form token. Please go back and try again.")
		c.Abort()
	}
}

func setupAllowedPath(path string) bool {
	return path == "/setup" ||
		path == "/health" ||
		strings.HasPrefix(path, "/static/")
}

func wantsHTML(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}
