package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginAndValidate(t *testing.T) {
	m := NewManager(time.Hour)
	tok, err := m.Login("alice")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}
	s, ok := m.Validate(tok)
	if !ok {
		t.Fatal("Validate returned !ok for fresh token")
	}
	if s.User != "alice" {
		t.Errorf("expected user alice, got %q", s.User)
	}
}

func TestRevoke(t *testing.T) {
	m := NewManager(time.Hour)
	tok, _ := m.Login("alice")
	m.Revoke(tok)
	if _, ok := m.Validate(tok); ok {
		t.Error("expected revoked token to fail Validate")
	}
}

func TestExpiry(t *testing.T) {
	m := NewManager(10 * time.Millisecond)
	tok, _ := m.Login("alice")
	time.Sleep(20 * time.Millisecond)
	if _, ok := m.Validate(tok); ok {
		t.Error("expected expired token to fail Validate")
	}
}

func TestCleanup(t *testing.T) {
	m := NewManager(10 * time.Millisecond)
	_, _ = m.Login("alice")
	time.Sleep(20 * time.Millisecond)
	m.Cleanup()
	if n := m.numSessions(); n != 0 {
		t.Errorf("expected 0 sessions after cleanup, got %d", n)
	}
}

// TestMiddlewareDualTransport verifies the middleware accepts both a
// session cookie and an Authorization: Bearer header carrying the same
// opaque session token.
func TestMiddlewareDualTransport(t *testing.T) {
	m := NewManager(time.Hour)
	tok, _ := m.Login("alice")

	ok := func(t *testing.T, r *http.Request, wantStatus int) {
		t.Helper()
		w := httptest.NewRecorder()
		var called bool
		h := m.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			called = true
		}))
		h.ServeHTTP(w, r)
		if w.Code != wantStatus {
			t.Errorf("status: got %d want %d", w.Code, wantStatus)
		}
		if (w.Code == http.StatusOK) != called {
			t.Errorf("handler called=%v but status=%d", called, w.Code)
		}
	}

	t.Run("cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/x", nil)
		r.AddCookie(&http.Cookie{Name: cookieName, Value: tok})
		ok(t, r, http.StatusOK)
	})

	t.Run("bearer", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/x", nil)
		r.Header.Set("Authorization", "Bearer "+tok)
		ok(t, r, http.StatusOK)
	})

	t.Run("bearer-overrides-cookie", func(t *testing.T) {
		// Garbage cookie + good Bearer → still authorized.
		r := httptest.NewRequest("GET", "/x", nil)
		r.AddCookie(&http.Cookie{Name: cookieName, Value: "garbage"})
		r.Header.Set("Authorization", "Bearer "+tok)
		ok(t, r, http.StatusOK)
	})

	t.Run("no-credentials", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/x", nil)
		ok(t, r, http.StatusUnauthorized)
	})

	t.Run("bad-bearer", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/x", nil)
		r.Header.Set("Authorization", "Bearer not-a-real-token")
		ok(t, r, http.StatusUnauthorized)
	})

	t.Run("malformed-authorization-header", func(t *testing.T) {
		// Wrong scheme: should fall back to cookie (which is missing) → 401.
		r := httptest.NewRequest("GET", "/x", nil)
		r.Header.Set("Authorization", "Basic "+tok)
		ok(t, r, http.StatusUnauthorized)
	})
}
