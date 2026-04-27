// Package tlsgen generates self-signed TLS certificates for serving the
// QuecTool web UI over HTTPS. Used by the `quectool gencert` subcommand for
// explicit operator-driven generation, and called from server startup to
// auto-create a cert on first boot when none exists yet.
package tlsgen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Config controls cert generation. Zero values use sensible defaults.
type Config struct {
	CertPath string
	KeyPath  string
	// Hosts is the SAN list. If empty, AllInterfaceHosts() is used —
	// every up interface IP plus loopback names.
	Hosts []string
	// Validity is the cert's lifetime. Zero defaults to 10 years.
	Validity time.Duration
}

// Result returns the resolved values used during generation, useful for
// logging/displaying to the operator.
type Result struct {
	Hosts    []string
	NotAfter time.Time
}

// Generate writes a fresh self-signed cert/key pair to cfg.CertPath and
// cfg.KeyPath. Existing files at those paths are overwritten — the caller
// decides whether to call this based on file presence.
func Generate(cfg Config) (Result, error) {
	if cfg.CertPath == "" || cfg.KeyPath == "" {
		return Result{}, fmt.Errorf("tlsgen: CertPath and KeyPath required")
	}
	hosts := cfg.Hosts
	if len(hosts) == 0 {
		var err error
		hosts, err = AllInterfaceHosts()
		if err != nil {
			return Result{}, err
		}
	}
	validity := cfg.Validity
	if validity == 0 {
		validity = 10 * 365 * 24 * time.Hour
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Result{}, fmt.Errorf("generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return Result{}, fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "QuecTool",
			Organization: []string{"QuecTool"},
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return Result{}, fmt.Errorf("create cert: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.CertPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("mkdir cert dir: %w", err)
	}
	if err := writePEM(cfg.CertPath, "CERTIFICATE", der, 0o644); err != nil {
		return Result{}, fmt.Errorf("write cert: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return Result{}, fmt.Errorf("marshal key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.KeyPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("mkdir key dir: %w", err)
	}
	if err := writePEM(cfg.KeyPath, "EC PRIVATE KEY", keyDER, 0o600); err != nil {
		return Result{}, fmt.Errorf("write key: %w", err)
	}

	return Result{Hosts: hosts, NotAfter: tmpl.NotAfter}, nil
}

// EnsureExists is the server-startup helper: if either path is missing,
// generate a fresh pair. Returns nil-Result if the files already exist
// (i.e. nothing was done).
func EnsureExists(cfg Config) (Result, bool, error) {
	if fileExists(cfg.CertPath) && fileExists(cfg.KeyPath) {
		return Result{}, false, nil
	}
	r, err := Generate(cfg)
	return r, err == nil, err
}

// AllInterfaceHosts returns the default SAN list: every up-interface IP
// (excluding loopback / unspecified), plus localhost / 127.0.0.1 / ::1
// so the device can reach itself by name without warnings.
func AllInterfaceHosts() ([]string, error) {
	out := []string{"localhost", "127.0.0.1", "::1"}
	seen := map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			s := ip.String()
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writePEM(path, typ string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: typ, Bytes: der}); err != nil {
		return err
	}
	return f.Sync()
}
