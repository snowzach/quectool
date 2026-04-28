package mainrpc

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/snowzach/golib/log"
	"github.com/snowzach/quectool/quectool/credfile"
	"github.com/snowzach/quectool/quectool/session"
)

const sessionTTLSeconds = 24 * 60 * 60

type loginReq struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

func (s *Server) AuthLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body loginReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid json")
			return
		}
		if !s.verifier.Check(body.User, body.Password) {
			time.Sleep(500 * time.Millisecond) // brute-force speed bump
			log.Infof("login failed user=%s remote=%s", body.User, r.RemoteAddr)
			writeErr(w, http.StatusUnauthorized, "ERR_BAD_CREDENTIALS", "invalid credentials")
			return
		}
		tok, err := s.sessions.Login(body.User)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "ERR_INTERNAL", "session error")
			return
		}
		// Cookie for browser clients; token in body for non-browser clients
		// (curl, scripts) that prefer Authorization: Bearer.
		session.SetCookie(w, r, tok, sessionTTLSeconds)
		writeJSON(w, http.StatusOK, map[string]string{
			"user":  body.User,
			"token": tok,
		})
	}
}

func (s *Server) AuthLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if tok := session.TokenFromRequest(r); tok != "" {
			s.sessions.Revoke(tok)
		}
		session.ClearCookie(w)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) AuthWhoami() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := session.TokenFromRequest(r)
		if tok == "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"user":          "",
				"authenticated": false,
			})
			return
		}
		sess, ok := s.sessions.Validate(tok)
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{
				"user":          "",
				"authenticated": false,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user":          sess.User,
			"authenticated": true,
		})
	}
}

type passwdReq struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// AuthChangePassword rotates the bcrypt hash for the session's user. The new
// value is bcrypted, written to server.auth.credentials_file (htpasswd-style),
// and swapped into the in-memory verifier — so the change takes effect
// immediately AND survives a restart.
//
// The credentials file is the only thing this endpoint ever writes; the
// bootstrap config (server.auth.password in YAML/env) is left untouched.
// Recovery if forgotten: delete the credentials file and restart.
func (s *Server) AuthChangePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body passwdReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid json")
			return
		}
		if body.New == "" {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "new password required")
			return
		}
		if len(body.New) < 6 {
			writeErr(w, http.StatusBadRequest, "ERR_WEAK_PASSWORD", "new password must be at least 6 characters")
			return
		}

		tok := session.TokenFromRequest(r)
		sess, ok := s.sessions.Validate(tok)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "ERR_UNAUTHENTICATED", "no session")
			return
		}

		// Verify the old password against the user owning this session.
		if !s.verifier.Check(sess.User, body.Old) {
			time.Sleep(500 * time.Millisecond)
			log.Infof("password change failed user=%s remote=%s", sess.User, r.RemoteAddr)
			writeErr(w, http.StatusUnauthorized, "ERR_BAD_CREDENTIALS", "current password incorrect")
			return
		}

		if s.credPath == "" {
			writeErr(w, http.StatusConflict, "ERR_NO_CREDENTIALS_FILE",
				"server.auth.credentials_file is unset; cannot persist a new password")
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(body.New), bcrypt.DefaultCost)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "ERR_INTERNAL", "could not hash password")
			return
		}

		err = credfile.Write(s.credPath, sess.User, string(hashed))
		switch {
		case errors.Is(err, os.ErrPermission):
			writeErr(w, http.StatusConflict, "ERR_NOT_WRITABLE",
				"credentials file is not writable: "+s.credPath)
			return
		case err != nil:
			log.Errorf("write credentials %s: %v", s.credPath, err)
			writeErr(w, http.StatusInternalServerError, "ERR_INTERNAL",
				"could not update credentials file")
			return
		}

		// Swap the in-memory hash so the new password takes effect immediately
		// without a restart.
		s.verifier.SetCredential(sess.User, string(hashed))
		log.Infof("password changed user=%s remote=%s", sess.User, r.RemoteAddr)

		w.WriteHeader(http.StatusNoContent)
	}
}

// Shared error envelope helpers (used by typed handlers too).
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": msg, "code": code})
}
