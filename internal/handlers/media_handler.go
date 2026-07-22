package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
)

// MediaHandler serves administrator-uploaded public media.
type MediaHandler struct {
	media *services.MediaService
}

// NewMediaHandler constructs a MediaHandler.
func NewMediaHandler(media *services.MediaService) *MediaHandler {
	return &MediaHandler{media: media}
}

// Show serves a single media object by its safe generated filename.
func (h *MediaHandler) Show(c *gin.Context) {
	file, mimeType, cleanup, err := h.media.Open(c.Param("name"))
	if err != nil {
		if errors.Is(err, services.ErrMediaMissing) {
			c.String(http.StatusNotFound, "media not found")
			return
		}
		c.String(http.StatusInternalServerError, "media unavailable")
		return
	}
	defer cleanup()

	c.Header("Cache-Control", "public, max-age=86400")
	c.DataFromReader(http.StatusOK, -1, mimeType, file, nil)
}
