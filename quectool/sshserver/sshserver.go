// Package sshserver runs an embedded SSH daemon for shell access to the
// device. It reuses the HTTP basic-auth credentials via auth.Verifier so
// there is a single user database for both surfaces.
//
// Host keys are generated (ed25519) on first run and persisted to disk; the
// fingerprint is logged at startup so the operator can verify it on first
// connect. Optional public key auth is supported via an authorized_keys file.
package sshserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/snowzach/golib/log"
	gossh "golang.org/x/crypto/ssh"

	"github.com/snowzach/quectool/quectool/auth"
)

type Config struct {
	Addr            string
	HostKeyFile     string
	AuthorizedKeys  string // path to authorized_keys file; empty disables pubkey auth
	Shell           string
	ShellArgs       []string
	Verifier        *auth.Verifier
	IdleTimeout     time.Duration
	MaxAuthTries    int
}

type Server struct {
	cfg *Config
	srv *ssh.Server
}

func New(cfg *Config) (*Server, error) {
	if cfg.Addr == "" {
		cfg.Addr = ":2222"
	}
	if cfg.HostKeyFile == "" {
		cfg.HostKeyFile = "host_key"
	}
	if cfg.Shell == "" {
		cfg.Shell = "/bin/sh"
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}
	if cfg.MaxAuthTries == 0 {
		cfg.MaxAuthTries = 3
	}

	signer, err := loadOrCreateHostKey(cfg.HostKeyFile)
	if err != nil {
		return nil, fmt.Errorf("host key: %w", err)
	}
	log.Infof("ssh host key fingerprint: %s", gossh.FingerprintSHA256(signer.PublicKey()))

	authorizedKeys, err := loadAuthorizedKeys(cfg.AuthorizedKeys)
	if err != nil {
		return nil, fmt.Errorf("authorized_keys: %w", err)
	}

	s := &Server{cfg: cfg}

	s.srv = &ssh.Server{
		Addr:        cfg.Addr,
		HostSigners: []ssh.Signer{signer},
		Handler:     s.handle,
		IdleTimeout: cfg.IdleTimeout,
		MaxTimeout:  0,
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": sftpSubsystemHandler,
		},
	}

	// Password auth via the shared verifier.
	if cfg.Verifier != nil {
		s.srv.PasswordHandler = func(ctx ssh.Context, password string) bool {
			ok := cfg.Verifier.Check(ctx.User(), password)
			if !ok {
				log.Infof("ssh password auth failed: user=%s remote=%s", ctx.User(), ctx.RemoteAddr())
			}
			return ok
		}
	}

	// Public key auth (optional). Compares against entries in authorized_keys.
	if len(authorizedKeys) > 0 {
		s.srv.PublicKeyHandler = func(ctx ssh.Context, key ssh.PublicKey) bool {
			for _, ak := range authorizedKeys {
				if ssh.KeysEqual(ak, key) {
					return true
				}
			}
			log.Infof("ssh pubkey auth failed: user=%s remote=%s", ctx.User(), ctx.RemoteAddr())
			return false
		}
	}

	return s, nil
}

func (s *Server) ListenAndServe() error {
	log.Infof("ssh listening on %s", s.cfg.Addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Close() error {
	return s.srv.Close()
}

// handle services a single SSH session.
func (s *Server) handle(sess ssh.Session) {
	cmd := exec.Command(s.cfg.Shell, s.cfg.ShellArgs...)
	cmd.Env = append(cmd.Env, sess.Environ()...)
	cmd.Env = append(cmd.Env,
		"USER="+sess.User(),
		"HOME=/root",
	)

	ptyReq, winCh, isPty := sess.Pty()
	if isPty {
		cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term)
		f, err := pty.Start(cmd)
		if err != nil {
			fmt.Fprintf(sess, "failed to start pty: %v\r\n", err)
			_ = sess.Exit(1)
			return
		}
		defer f.Close()

		// Initial size + resize handling.
		_ = pty.Setsize(f, &pty.Winsize{
			Rows: uint16(ptyReq.Window.Height),
			Cols: uint16(ptyReq.Window.Width),
		})
		go func() {
			for win := range winCh {
				_ = pty.Setsize(f, &pty.Winsize{
					Rows: uint16(win.Height),
					Cols: uint16(win.Width),
				})
			}
		}()

		// Bidirectional copy. When the session closes (client disconnect or
		// process exit), Close on either end unblocks the other Copy.
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = io.Copy(f, sess)
			// Client closed input; signal EOF to the shell.
			_ = f.Close()
		}()
		go func() {
			defer wg.Done()
			_, _ = io.Copy(sess, f)
		}()

		err = cmd.Wait()
		_ = f.Close()
		wg.Wait()

		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				_ = sess.Exit(ee.ExitCode())
				return
			}
			_ = sess.Exit(1)
			return
		}
		_ = sess.Exit(0)
		return
	}

	// Non-PTY: scp/exec/etc. Hook stdio directly.
	cmd.Stdin = sess
	cmd.Stdout = sess
	cmd.Stderr = sess.Stderr()
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			_ = sess.Exit(ee.ExitCode())
			return
		}
		_ = sess.Exit(1)
		return
	}
	_ = sess.Exit(0)
}

// sftpSubsystemHandler services an SFTP request inside an authenticated SSH
// session. The pkg/sftp server runs over the session's stdio channel, so we
// just hand it sess and sess.Stderr().
func sftpSubsystemHandler(sess ssh.Session) {
	srv, err := sftp.NewServer(sess)
	if err != nil {
		log.Errorf("sftp subsystem init failed: %v", err)
		_ = sess.Exit(1)
		return
	}
	if err := srv.Serve(); err != nil && !errors.Is(err, io.EOF) {
		log.Errorf("sftp serve error: %v", err)
		_ = sess.Exit(1)
		return
	}
	_ = srv.Close()
	_ = sess.Exit(0)
}

// loadOrCreateHostKey reads an ed25519 host key from disk, creating a new one
// on first run and persisting it with 0600 permissions.
func loadOrCreateHostKey(path string) (gossh.Signer, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		signer, err := gossh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse host key %s: %w", path, err)
		}
		return signer, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	pemBlock, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}
	out := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return nil, fmt.Errorf("write host key %s: %w", path, err)
	}
	log.Infof("generated new ssh host key: %s", path)
	return gossh.ParsePrivateKey(out)
}

// loadAuthorizedKeys reads and parses an authorized_keys file. Empty path
// returns nil (pubkey auth disabled).
func loadAuthorizedKeys(path string) ([]ssh.PublicKey, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var keys []ssh.PublicKey
	for {
		key, _, _, rest, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			// Stop on first parse error after at least some keys parsed; report
			// otherwise.
			if len(keys) == 0 && len(strings.TrimSpace(string(data))) > 0 {
				return nil, err
			}
			break
		}
		keys = append(keys, key)
		data = rest
		if len(data) == 0 {
			break
		}
	}
	return keys, nil
}
