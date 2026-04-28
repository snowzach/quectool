// Package auth provides shared credential verification for the HTTP basic-auth
// middleware and the SSH server. Credentials are loaded once and reused for
// both surfaces so there is a single user database.
package auth

import (
	"crypto/subtle"
	"maps"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// Verifier holds the user→password (or bcrypt hash) map and provides constant-
// time comparison. Credentials can be swapped at runtime (e.g. after a UI-
// initiated password change) and the verifier remains safe for concurrent use.
type Verifier struct {
	mu    sync.RWMutex
	creds map[string]string
}

func New(creds map[string]string) *Verifier {
	cp := make(map[string]string, len(creds))
	maps.Copy(cp, creds)
	return &Verifier{creds: cp}
}

// Check returns true if user/pass are valid. A stored value beginning with
// "$2" is treated as a bcrypt hash; otherwise constant-time string compare.
func (v *Verifier) Check(user, pass string) bool {
	v.mu.RLock()
	stored, ok := v.creds[user]
	v.mu.RUnlock()
	if !ok {
		return false
	}
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(pass)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(pass)) == 1
}

// SetCredential replaces the stored credential for user with stored (a bcrypt
// hash or plaintext). Used by the UI password-change flow after the new value
// is persisted to the config file.
func (v *Verifier) SetCredential(user, stored string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.creds[user] = stored
}
