package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SessionConfig holds configuration for session middleware
type SessionConfig struct {
	Name            string        // Session cookie name
	Secret          string        // Secret key for session encryption
	CookiePath      string        // Path for the cookie
	CookieDomain    string        // Domain for the cookie
	CookieSecure    bool          // Secure flag for cookie
	CookieHTTPOnly  bool          // HTTPOnly flag for cookie
	CookieSameSite  string        // SameSite attribute for cookie
	Expiration      time.Duration // Session expiration time
	CleanupInterval time.Duration // Interval for cleaning up expired sessions
	Store           SessionStore  // Session store implementation
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		Name:            "session",
		Secret:          "mithril-secret-key-change-in-production",
		CookiePath:      "/",
		CookieDomain:    "",
		CookieSecure:    false, // Set to true in production with HTTPS
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
		Expiration:      24 * time.Hour,
		CleanupInterval: 1 * time.Hour,
		Store:           NewMemoryStore(),
	}
}

// SessionStore interface for session storage
type SessionStore interface {
	Get(sessionID string) (*Session, error)
	Set(sessionID string, session *Session) error
	Delete(sessionID string) error
	Cleanup() error
}

// Session represents a user session
type Session struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt time.Time              `json:"expires_at"`
}

// NewSession creates a new session
func NewSession(expiration time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:        generateSessionID(),
		Data:      make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(expiration),
	}
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Get retrieves a value from the session
func (s *Session) Get(key string) (interface{}, bool) {
	value, exists := s.Data[key]
	return value, exists
}

// Set sets a value in the session
func (s *Session) Set(key string, value interface{}) {
	s.Data[key] = value
	s.UpdatedAt = time.Now()
}

// Delete removes a value from the session
func (s *Session) Delete(key string) {
	delete(s.Data, key)
	s.UpdatedAt = time.Now()
}

// Clear removes all data from the session
func (s *Session) Clear() {
	s.Data = make(map[string]interface{})
	s.UpdatedAt = time.Now()
}

// MemoryStore implements SessionStore using in-memory storage
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		sessions: make(map[string]*Session),
	}

	// Start cleanup goroutine
	go store.startCleanup()

	return store
}

// Get retrieves a session by ID
func (ms *MemoryStore) Get(sessionID string) (*Session, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	session, exists := ms.sessions[sessionID]
	if !exists || session.IsExpired() {
		return nil, fmt.Errorf("session not found or expired")
	}

	return session, nil
}

// Set stores a session
func (ms *MemoryStore) Set(sessionID string, session *Session) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.sessions[sessionID] = session
	return nil
}

// Delete removes a session
func (ms *MemoryStore) Delete(sessionID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.sessions, sessionID)
	return nil
}

// Cleanup removes expired sessions
func (ms *MemoryStore) Cleanup() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	for id, session := range ms.sessions {
		if session.ExpiresAt.Before(now) {
			delete(ms.sessions, id)
		}
	}

	return nil
}

// startCleanup starts the cleanup goroutine
func (ms *MemoryStore) startCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		_ = ms.Cleanup()
	}
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// SessionManager manages sessions
type SessionManager struct {
	config SessionConfig
	store  SessionStore
}

// NewSessionManager creates a new session manager
func NewSessionManager(config SessionConfig) *SessionManager {
	return &SessionManager{
		config: config,
		store:  config.Store,
	}
}

// getSessionID retrieves session ID from cookie
func (sm *SessionManager) getSessionID(ctx *fiber.Ctx) string {
	return ctx.Cookies(sm.config.Name)
}

// setSessionCookie sets the session cookie
func (sm *SessionManager) setSessionCookie(ctx *fiber.Ctx, sessionID string) {
	cookie := &fiber.Cookie{
		Name:     sm.config.Name,
		Value:    sessionID,
		Path:     sm.config.CookiePath,
		Domain:   sm.config.CookieDomain,
		Secure:   sm.config.CookieSecure,
		HTTPOnly: sm.config.CookieHTTPOnly,
		SameSite: sm.config.CookieSameSite,
		Expires:  time.Now().Add(sm.config.Expiration),
	}
	ctx.Cookie(cookie)
}

// getOrCreateSession retrieves existing session or creates a new one
func (sm *SessionManager) getOrCreateSession(ctx *fiber.Ctx) (*Session, error) {
	sessionID := sm.getSessionID(ctx)

	if sessionID != "" {
		session, err := sm.store.Get(sessionID)
		if err == nil {
			return session, nil
		}
	}

	// Create new session
	session := NewSession(sm.config.Expiration)
	sessionID = session.ID

	if err := sm.store.Set(sessionID, session); err != nil {
		return nil, err
	}

	sm.setSessionCookie(ctx, sessionID)
	return session, nil
}

// Handler returns the session middleware handler
func (sm *SessionManager) Handler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		session, err := sm.getOrCreateSession(ctx)
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{
				"error": "Failed to initialize session",
			})
		}

		// Store session in context
		ctx.Locals("session", session)

		// Save session after request
		defer func() {
			_ = sm.store.Set(session.ID, session)
		}()

		return ctx.Next()
	}
}

// SessionMiddleware creates a session middleware with default configuration
func SessionMiddleware(config ...SessionConfig) fiber.Handler {
	cfg := DefaultSessionConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	manager := NewSessionManager(cfg)
	return manager.Handler()
}

// GetSession retrieves the session from context
func GetSession(ctx *fiber.Ctx) *Session {
	if session := ctx.Locals("session"); session != nil {
		return session.(*Session)
	}
	return nil
}

// Session helpers
func SessionGet(ctx *fiber.Ctx, key string) (interface{}, bool) {
	session := GetSession(ctx)
	if session == nil {
		return nil, false
	}
	return session.Get(key)
}

func SessionSet(ctx *fiber.Ctx, key string, value interface{}) {
	session := GetSession(ctx)
	if session != nil {
		session.Set(key, value)
	}
}

func SessionDelete(ctx *fiber.Ctx, key string) {
	session := GetSession(ctx)
	if session != nil {
		session.Delete(key)
	}
}

func SessionClear(ctx *fiber.Ctx) {
	session := GetSession(ctx)
	if session != nil {
		session.Clear()
	}
}

func SessionDestroy(ctx *fiber.Ctx) {
	session := GetSession(ctx)
	if session != nil {
		// Clear the session cookie
		ctx.Cookie(&fiber.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-1 * time.Hour),
			HTTPOnly: true,
		})
	}
}
