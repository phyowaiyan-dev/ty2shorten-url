package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	sessionCookieName = "ty2_session"
	csrfCookieName    = "ty2_csrf"
	defaultMaxAge     = 12 * time.Hour
)

var (
	errInvalidCookie = errors.New("invalid signed cookie")
	errExpiredCookie = errors.New("signed cookie expired")
)

type sessionPayload struct {
	AdminID uint   `json:"admin_id"`
	Expires int64  `json:"expires"`
	Nonce   string `json:"nonce"`
}

type csrfPayload struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

// Manager signs auth session and CSRF cookies.
type Manager struct {
	secret       []byte
	secureCookie bool
	maxAge       time.Duration
}

// NewManager constructs a Manager.
func NewManager(secret string, secureCookie bool) *Manager {
	if strings.TrimSpace(secret) == "" {
		secret = "development-insecure-session-secret"
	}

	return &Manager{
		secret:       []byte(secret),
		secureCookie: secureCookie,
		maxAge:       defaultMaxAge,
	}
}

// AdminID returns the authenticated administrator ID from the session cookie.
func (m *Manager) AdminID(c *gin.Context) (uint, bool) {
	raw, err := c.Cookie(sessionCookieName)
	if err != nil {
		return 0, false
	}

	var payload sessionPayload
	if err := m.decode(raw, &payload); err != nil {
		return 0, false
	}
	if payload.AdminID == 0 {
		return 0, false
	}

	return payload.AdminID, true
}

// SetAdmin replaces the current session with a signed administrator session.
func (m *Manager) SetAdmin(c *gin.Context, adminID uint) error {
	nonce, err := randomToken()
	if err != nil {
		return err
	}
	payload := sessionPayload{
		AdminID: adminID,
		Expires: time.Now().Add(m.maxAge).Unix(),
		Nonce:   nonce,
	}
	value, err := m.encode(payload)
	if err != nil {
		return err
	}

	m.setCookie(c, sessionCookieName, value, int(m.maxAge.Seconds()), true)
	return nil
}

// ClearAdmin deletes the current session cookie.
func (m *Manager) ClearAdmin(c *gin.Context) {
	m.setCookie(c, sessionCookieName, "", -1, true)
}

// CSRFToken returns a valid CSRF token, creating one when needed.
func (m *Manager) CSRFToken(c *gin.Context) (string, error) {
	raw, err := c.Cookie(csrfCookieName)
	if err == nil {
		var payload csrfPayload
		if decodeErr := m.decode(raw, &payload); decodeErr == nil && payload.Token != "" {
			return payload.Token, nil
		}
	}

	token, err := randomToken()
	if err != nil {
		return "", err
	}
	payload := csrfPayload{
		Token:   token,
		Expires: time.Now().Add(m.maxAge).Unix(),
	}
	value, err := m.encode(payload)
	if err != nil {
		return "", err
	}

	m.setCookie(c, csrfCookieName, value, int(m.maxAge.Seconds()), true)
	return token, nil
}

// ValidateCSRF validates a submitted form token against the signed CSRF cookie.
func (m *Manager) ValidateCSRF(c *gin.Context) bool {
	formToken := c.PostForm("csrf_token")
	if formToken == "" {
		return false
	}

	raw, err := c.Cookie(csrfCookieName)
	if err != nil {
		return false
	}

	var payload csrfPayload
	if err := m.decode(raw, &payload); err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(formToken), []byte(payload.Token)) == 1
}

func (m *Manager) encode(payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal signed cookie: %w", err)
	}

	bodyPart := base64.RawURLEncoding.EncodeToString(body)
	signature := m.sign(bodyPart)
	return bodyPart + "." + signature, nil
}

func (m *Manager) decode(value string, payload any) error {
	bodyPart, signature, ok := strings.Cut(value, ".")
	if !ok {
		return errInvalidCookie
	}
	if subtle.ConstantTimeCompare([]byte(m.sign(bodyPart)), []byte(signature)) != 1 {
		return errInvalidCookie
	}

	body, err := base64.RawURLEncoding.DecodeString(bodyPart)
	if err != nil {
		return fmt.Errorf("%w: %v", errInvalidCookie, err)
	}
	if err := json.Unmarshal(body, payload); err != nil {
		return fmt.Errorf("%w: %v", errInvalidCookie, err)
	}

	expiresAt, err := expires(payload)
	if err != nil {
		return err
	}
	if expiresAt <= time.Now().Unix() {
		return errExpiredCookie
	}

	return nil
}

func (m *Manager) sign(body string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *Manager) setCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   m.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func expires(payload any) (int64, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload expiry: %w", err)
	}

	var values map[string]any
	if err := json.Unmarshal(body, &values); err != nil {
		return 0, fmt.Errorf("unmarshal payload expiry: %w", err)
	}

	value, ok := values["expires"]
	if !ok {
		return 0, errInvalidCookie
	}

	switch typed := value.(type) {
	case float64:
		return int64(typed), nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, errInvalidCookie
	}
}
