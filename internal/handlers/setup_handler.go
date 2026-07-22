package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
)

// SetupHandler serves the first-run setup wizard.
type SetupHandler struct {
	setup   *services.SetupService
	session *session.Manager
}

// NewSetupHandler constructs a SetupHandler.
func NewSetupHandler(setup *services.SetupService, sessions *session.Manager) *SetupHandler {
	return &SetupHandler{setup: setup, session: sessions}
}

// Show renders the setup form unless setup is already complete.
func (h *SetupHandler) Show(c *gin.Context) {
	completed, err := h.setup.IsCompleted()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Setup unavailable", "Setup status could not be checked."))
		return
	}
	if completed {
		c.Redirect(http.StatusSeeOther, "/admin/login")
		return
	}

	h.render(c, http.StatusOK, services.SetupForm{}, nil)
}

// Store validates setup input and completes initial setup.
func (h *SetupHandler) Store(c *gin.Context) {
	completed, err := h.setup.IsCompleted()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Setup unavailable", "Setup status could not be checked."))
		return
	}
	if completed {
		c.HTML(http.StatusForbidden, "public/forbidden.html", forbiddenView("Access denied", "Initial setup has already been completed."))
		return
	}

	form := services.SetupForm{
		SiteName:        c.PostForm("site_name"),
		SiteDescription: c.PostForm("site_description"),
		AdminName:       c.PostForm("admin_name"),
		AdminEmail:      c.PostForm("admin_email"),
		AdminPassword:   c.PostForm("admin_password"),
		PasswordConfirm: c.PostForm("password_confirm"),
		AndroidURL:      c.PostForm("android_url"),
		AppleURL:        c.PostForm("apple_url"),
		DefaultURL:      c.PostForm("default_url"),
		PublicBaseURL:   c.PostForm("public_base_url"),
		SupportEmail:    c.PostForm("support_email"),
	}

	if err := h.setup.Complete(form); err != nil {
		var validationErr services.SetupValidationError
		switch {
		case errors.As(err, &validationErr):
			h.render(c, http.StatusUnprocessableEntity, form, validationErr.FieldErrors)
		case errors.Is(err, services.ErrSetupCompleted):
			c.HTML(http.StatusForbidden, "public/forbidden.html", forbiddenView("Access denied", "Initial setup has already been completed."))
		default:
			c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Setup failed", "Setup could not be completed. Please check the form and try again."))
		}
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/login")
}

func (h *SetupHandler) render(c *gin.Context, status int, form services.SetupForm, fieldErrors map[string]string) {
	if fieldErrors == nil {
		fieldErrors = map[string]string{}
	}

	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Setup unavailable", "A secure form token could not be created."))
		return
	}

	c.HTML(status, "public/setup.html", gin.H{
		"Title":       "Setup ty2shorten-url",
		"CSRFToken":   csrfToken,
		"Form":        form,
		"FieldErrors": fieldErrors,
	})
}

func errorView(title, message string) gin.H {
	return gin.H{"Title": title, "Heading": title, "Message": message}
}

func notFoundView(title, message string) gin.H {
	return gin.H{"Title": title, "Heading": title, "Message": message, "RedirectDelay": 5, "AutoRedirectURL": "/"}
}

func forbiddenView(title, message string) gin.H {
	return gin.H{"Title": title, "Heading": title, "Message": message, "RedirectDelay": 5, "AutoRedirectURL": "/"}
}
