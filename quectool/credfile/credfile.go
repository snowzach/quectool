// Package credfile reads and writes the small "live credentials" file
// (htpasswd-style: one line, "user:bcrypt-hash") that overrides whatever
// server.auth.{username,password} was set to in the bootstrap config.
//
// Why a separate file (not server.auth.password in the YAML):
//   - the YAML stays purely declarative; password rotations never touch it
//   - SERVER_AUTH_PASSWORD env vars don't silently override UI password changes
//   - "forgot the password" reset is `rm <credentials_file> && restart`
package credfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrEmpty is returned by Read when the file exists but contains no
// usable credential line. Caller should treat this as "no override
// in effect" and fall back to bootstrap config.
var ErrEmpty = errors.New("credentials file is empty")

// Read returns the user and stored hash from path. If the file does not
// exist, returns os.ErrNotExist (caller usually treats this as "no
// override in effect"). Lines starting with '#' and blank lines are
// ignored. Only the first credential line is honored — multi-user is
// not currently supported.
func Read(path string) (user, stored string, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// htpasswd-style: split on the FIRST colon. The bcrypt hash
		// contains '$' but never ':', so this is unambiguous.
		idx := strings.IndexByte(line, ':')
		if idx <= 0 || idx == len(line)-1 {
			return "", "", fmt.Errorf("credentials: malformed line %q (want user:hash)", line)
		}
		return line[:idx], line[idx+1:], nil
	}
	return "", "", ErrEmpty
}

// Write atomically replaces path with a single user:stored line. The
// parent directory is created if needed. Mode is 0600 — the file holds
// a credential and shouldn't be world-readable.
func Write(path, user, stored string) error {
	if user == "" {
		return errors.New("credentials: empty username")
	}
	if stored == "" {
		return errors.New("credentials: empty hash")
	}
	if strings.ContainsAny(user, ":\n") {
		return errors.New("credentials: username may not contain ':' or newline")
	}
	if strings.ContainsAny(stored, "\n") {
		return errors.New("credentials: hash may not contain newline")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".credentials-*.tmp")
	if err != nil {
		return fmt.Errorf("tempfile: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once rename succeeds

	if err := os.Chmod(tmpPath, 0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod tmp: %w", err)
	}
	if _, err := fmt.Fprintf(tmp, "%s:%s\n", user, stored); err != nil {
		tmp.Close()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
