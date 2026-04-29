// Package cidrallow parses an allow-list of CIDR strings into a fast
// per-request matcher. Used as an HTTP middleware in front of the API and
// as a per-connection check in the SSH server, so the binary can listen on
// every interface (simple bind, no multi-listener juggling) while still
// gating access by source IP.
package cidrallow

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/snowzach/golib/log"
)

// Matcher answers "is this remote address allowed?". The empty matcher
// (no CIDRs configured) returns true for everyone — operators who set
// server.allow_cidrs to [] are explicitly opting out.
type Matcher struct {
	prefixes []netip.Prefix
	allowAll bool
}

// New parses the given CIDR strings into a Matcher. Returns an error if
// any string is malformed. An empty input list builds an allow-all matcher.
func New(cidrs []string) (*Matcher, error) {
	if len(cidrs) == 0 {
		return &Matcher{allowAll: true}, nil
	}
	m := &Matcher{prefixes: make([]netip.Prefix, 0, len(cidrs))}
	for _, raw := range cidrs {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, fmt.Errorf("cidrallow: invalid CIDR %q: %w", s, err)
		}
		m.prefixes = append(m.prefixes, p)
	}
	if len(m.prefixes) == 0 {
		m.allowAll = true
	}
	return m, nil
}

// Allow returns true if remoteAddr (host[:port], IPv4 or IPv6) is in any
// configured prefix. Returns false for malformed input — fail-closed.
func (m *Matcher) Allow(remoteAddr string) bool {
	if m == nil || m.allowAll {
		return true
	}
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	// Normalize 4-in-6 (e.g. ::ffff:192.168.225.1) to v4 so an IPv4 prefix
	// matches a dual-stack listener.
	addr = addr.Unmap()
	for _, p := range m.prefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// Middleware rejects HTTP requests whose RemoteAddr isn't in the allow-list.
// Logs blocked attempts at info level so operators can audit.
func (m *Matcher) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.Allow(r.RemoteAddr) {
			log.Infof("blocked request from %s for %s %s (not in server.allow_cidrs)",
				r.RemoteAddr, r.Method, r.URL.Path)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
