// Package session provides in-memory cookie-session storage backed by
// random opaque tokens. Tokens are 32 bytes of crypto/rand encoded in
// base64-URL, so they are already unguessable; no separate signature.
//
// On process restart, all sessions are invalidated (map is in-memory).
// A single user device tolerates this fine.
package session

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type Session struct {
	User      string
	ExpiresAt time.Time
}

type Manager struct {
	ttl  time.Duration
	mu   sync.RWMutex
	data map[string]Session
}

func NewManager(ttl time.Duration) *Manager {
	return &Manager{ttl: ttl, data: map[string]Session{}}
}

// Login creates a new session for the given user and returns the token.
// Caller must validate credentials before calling.
func (m *Manager) Login(user string) (string, error) {
	tok, err := newToken()
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.data[tok] = Session{User: user, ExpiresAt: time.Now().Add(m.ttl)}
	m.mu.Unlock()
	return tok, nil
}

func (m *Manager) Validate(tok string) (*Session, bool) {
	m.mu.RLock()
	s, ok := m.data[tok]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(s.ExpiresAt) {
		m.Revoke(tok)
		return nil, false
	}
	return &s, true
}

func (m *Manager) Revoke(tok string) {
	m.mu.Lock()
	delete(m.data, tok)
	m.mu.Unlock()
}

// Cleanup removes expired entries. Caller schedules periodically.
func (m *Manager) Cleanup() {
	now := time.Now()
	m.mu.Lock()
	for k, s := range m.data {
		if now.After(s.ExpiresAt) {
			delete(m.data, k)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) numSessions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

func newToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
