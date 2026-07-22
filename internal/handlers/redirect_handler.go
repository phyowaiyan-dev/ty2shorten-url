package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
)

// RedirectHandler serves app-store and short-link redirects.
type RedirectHandler struct {
	redirects *services.RedirectService
}

// NewRedirectHandler constructs a RedirectHandler.
func NewRedirectHandler(redirects *services.RedirectService) *RedirectHandler {
	return &RedirectHandler{redirects: redirects}
}

// Android redirects to the configured Android destination.
func (h *RedirectHandler) Android(c *gin.Context) {
	destination, err := h.redirects.AndroidURL()
	if err != nil {
		h.missingDestination(c, "Android app link is not configured yet.")
		return
	}

	c.Set("analytics_route_type", "android")
	c.Set("analytics_destination_platform", "android")
	c.Redirect(http.StatusTemporaryRedirect, destination)
}

// Apple redirects to the configured Apple destination.
func (h *RedirectHandler) Apple(c *gin.Context) {
	destination, err := h.redirects.AppleURL()
	if err != nil {
		h.missingDestination(c, "Apple app link is not configured yet.")
		return
	}

	c.Set("analytics_route_type", "apple")
	c.Set("analytics_destination_platform", "apple")
	c.Redirect(http.StatusTemporaryRedirect, destination)
}

// Get redirects mobile devices to the matching app-store route.
func (h *RedirectHandler) Get(c *gin.Context) {
	target := services.DeviceTarget(c.GetHeader("User-Agent"))
	c.Set("analytics_route_type", "auto_detect")
	switch target {
	case "/android":
		c.Set("analytics_destination_platform", "android")
	case "/apple":
		c.Set("analytics_destination_platform", "apple")
	default:
		c.Set("analytics_destination_platform", "desktop")
	}
	c.Redirect(http.StatusTemporaryRedirect, target)
}

// ShortLink redirects an active custom short link.
func (h *RedirectHandler) ShortLink(c *gin.Context) {
	link, err := h.redirects.ResolveShortLink(c.Param("slug"))
	if err != nil {
		if errors.Is(err, services.ErrShortLinkNotFound) {
			c.HTML(http.StatusNotFound, "public/not_found.html", notFoundView("Link not found", "That short link is unavailable or has expired."))
			return
		}
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Redirect unavailable", "The short link could not be resolved."))
		return
	}

	c.Set("analytics_route_type", "short_link")
	c.Set("analytics_destination_platform", "custom")
	c.Set("analytics_short_link_slug", link.Slug)
	c.Set("analytics_short_link_id", link.ID)
	c.Redirect(http.StatusTemporaryRedirect, link.Destination)
}

func (h *RedirectHandler) missingDestination(c *gin.Context, message string) {
	c.HTML(http.StatusOK, "public/missing_destination.html", gin.H{
		"Title":   "Destination unavailable",
		"Heading": "Destination unavailable",
		"Message": message,
	})
}
