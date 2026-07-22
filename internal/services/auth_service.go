package services

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/models"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials is returned for any failed login attempt.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrLoginThrottled is returned after too many failed login attempts.
	ErrLoginThrottled = errors.New("too many login attempts; wait a few minutes and try again")
)

// LoginLimiter isolates login throttling behind a replaceable interface.
type LoginLimiter interface {
	Allow(key string) bool
	RegisterFailure(key string)
	Reset(key string)
}

// MemoryLoginLimiter is a small single-instance throttler.
type MemoryLoginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewMemoryLoginLimiter constructs a MemoryLoginLimiter.
func NewMemoryLoginLimiter(limit int, window time.Duration) *MemoryLoginLimiter {
	return &MemoryLoginLimiter{
		failures: map[string][]time.Time{},
		limit:    limit,
		window:   window,
	}
}

// Allow reports whether the key may attempt login.
func (l *MemoryLoginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.prune(key)
	return len(l.failures[key]) < l.limit
}

// RegisterFailure records a failed login attempt.
func (l *MemoryLoginLimiter) RegisterFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.prune(key)
	l.failures[key] = append(l.failures[key], time.Now())
}

// Reset clears login failures for a key.
func (l *MemoryLoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.failures, key)
}

func (l *MemoryLoginLimiter) prune(key string) {
	cutoff := time.Now().Add(-l.window)
	failures := l.failures[key]
	kept := failures[:0]
	for _, failure := range failures {
		if failure.After(cutoff) {
			kept = append(kept, failure)
		}
	}
	if len(kept) == 0 {
		delete(l.failures, key)
		return
	}
	l.failures[key] = kept
}

// AuthService owns administrator authentication.
type AuthService struct {
	admins  *repositories.AdminRepository
	limiter LoginLimiter
	logger  *slog.Logger
}

// NewAuthService constructs an AuthService.
func NewAuthService(admins *repositories.AdminRepository, limiter LoginLimiter, logger *slog.Logger) *AuthService {
	return &AuthService{admins: admins, limiter: limiter, logger: logger}
}

// Login authenticates an administrator with generic failure responses.
func (s *AuthService) Login(email, password, clientIP string) (*models.AdminUser, error) {
	normalizedEmail := normalizeEmail(email)
	key := strings.ToLower(normalizedEmail + "|" + clientIP)

	if s.limiter != nil && !s.limiter.Allow(key) {
		s.logFailure(clientIP, "throttled")
		return nil, ErrLoginThrottled
	}

	admin, err := s.admins.FindByEmail(normalizedEmail)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		s.registerFailure(key)
		s.logFailure(clientIP, "invalid_credentials")
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		s.registerFailure(key)
		s.logFailure(clientIP, "invalid_credentials")
		return nil, ErrInvalidCredentials
	}

	if s.limiter != nil {
		s.limiter.Reset(key)
	}

	return admin, nil
}

func (s *AuthService) logFailure(clientIP, reason string) {
	if s.logger != nil {
		s.logger.Warn("authentication failure", slog.String("client_ip", clientIP), slog.String("reason", reason))
	}
}

// AdminByID returns an administrator by ID.
func (s *AuthService) AdminByID(adminID uint) (*models.AdminUser, error) {
	admin, err := s.admins.FindByID(adminID)
	if err != nil {
		return nil, fmt.Errorf("load authenticated admin: %w", err)
	}
	if admin == nil {
		return nil, ErrInvalidCredentials
	}

	return admin, nil
}

func (s *AuthService) registerFailure(key string) {
	if s.limiter != nil {
		s.limiter.RegisterFailure(key)
	}
}
