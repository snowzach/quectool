package session

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	cookieName  = "quectool_session"
	bearerPfx   = "Bearer "
)

type ctxKey struct{}

// TokenFromRequest extracts the session token from either the Authorization
// header (Authorization: Bearer <token>) or the session cookie. The header
// takes precedence so non-browser clients (curl, scripts) can use Bearer auth
// without any cookie-jar handling.
func TokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, bearerPfx) {
		return strings.TrimSpace(h[len(bearerPfx):])
	}
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value
	}
	return ""
}

// Middleware returns a chi-compatible middleware that requires a valid
// session token, transported as either an Authorization: Bearer header or
// the session cookie. On 401 it writes the standard error envelope.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := TokenFromRequest(r)
		if tok == "" {
			writeUnauth(w)
			return
		}
		s, ok := m.Validate(tok)
		if !ok {
			writeUnauth(w)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, s)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext returns the Session attached to a request authenticated by
// Middleware, if any.
func FromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(ctxKey{}).(*Session)
	return s, ok
}

// SetCookie writes the session cookie on a successful login.
func SetCookie(w http.ResponseWriter, r *http.Request, token string, ttlSeconds int) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   ttlSeconds,
	})
}

// ClearCookie writes a cookie that immediately expires for logout.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func writeUnauth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "authentication required",
		"code":  "ERR_UNAUTHENTICATED",
	})
}
