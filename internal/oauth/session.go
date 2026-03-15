package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Session represents an authenticated user session.
type Session struct {
	ID        string
	Email     string
	Name      string
	Picture   string
	Token     *oauth2.Token
	CreatedAt time.Time
}

// SessionStore is a thread-safe in-memory session store.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionStore creates a new empty session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session and returns it.
func (s *SessionStore) Create(email, name, picture string, token *oauth2.Token) *Session {
	id := generateID()
	session := &Session{
		ID:        id,
		Email:     email,
		Name:      name,
		Picture:   picture,
		Token:     token,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()
	return session
}

// Get retrieves a session by ID, or nil if not found.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

// Delete removes a session by ID.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// generateID generates a 32-byte random hex-encoded string.
func generateID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
