package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
)

// SEOHandler serves crawler resources.
type SEOHandler struct {
	seo *services.SEOService
}

// NewSEOHandler constructs an SEOHandler.
func NewSEOHandler(seo *services.SEOService) *SEOHandler {
	return &SEOHandler{seo: seo}
}

// Robots renders robots.txt.
func (h *SEOHandler) Robots(c *gin.Context) {
	body, err := h.seo.Robots()
	if err != nil {
		c.String(http.StatusInternalServerError, "User-agent: *\nDisallow: /\n")
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=300")
	c.String(http.StatusOK, body)
}

// Sitemap renders sitemap.xml.
func (h *SEOHandler) Sitemap(c *gin.Context) {
	body, status, err := h.seo.Sitemap()
	if err != nil {
		c.String(http.StatusInternalServerError, "sitemap unavailable")
		return
	}
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=300")
	c.String(status, string(body))
}
