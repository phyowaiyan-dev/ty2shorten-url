package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/session"
)

// AuthHandler serves administrator login and logout.
type AuthHandler struct {
	auth    *services.AuthService
	session *session.Manager
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(auth *services.AuthService, sessions *session.Manager) *AuthHandler {
	return &AuthHandler{auth: auth, session: sessions}
}

// ShowLogin renders the administrator login form.
func (h *AuthHandler) ShowLogin(c *gin.Context) {
	if _, ok := h.session.AdminID(c); ok {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}

	h.renderLogin(c, http.StatusOK, "", "")
}

// Login authenticates an administrator and replaces the session.
func (h *AuthHandler) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	admin, err := h.auth.Login(email, password, c.ClientIP())
	if err != nil {
		message := "Invalid email or password."
		if errors.Is(err, services.ErrLoginThrottled) {
			message = "Too many attempts. Please wait a few minutes and try again."
		}
		h.renderLogin(c, http.StatusUnauthorized, email, message)
		return
	}

	if err := h.session.SetAdmin(c, admin.ID); err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Login failed", "A secure session could not be created."))
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

// Logout clears the administrator session.
func (h *AuthHandler) Logout(c *gin.Context) {
	h.session.ClearAdmin(c)
	c.Redirect(http.StatusSeeOther, "/admin/login")
}

func (h *AuthHandler) renderLogin(c *gin.Context, status int, email string, message string) {
	csrfToken, err := h.session.CSRFToken(c)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "public/error.html", errorView("Login unavailable", "A secure form token could not be created."))
		return
	}

	c.HTML(status, "public/login.html", gin.H{
		"Title":     "Admin login",
		"CSRFToken": csrfToken,
		"Email":     email,
		"Error":     message,
	})
}
