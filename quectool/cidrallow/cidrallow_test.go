package cidrallow

import "testing"

func TestAllow(t *testing.T) {
	m, err := New([]string{"192.168.225.0/24", "127.0.0.0/8", "::1/128"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cases := []struct {
		addr string
		want bool
	}{
		// Allowed: in-prefix.
		{"192.168.225.7:51234", true},
		{"127.0.0.1:51234", true},
		{"[::1]:51234", true},
		// IPv4-mapped IPv6 of an allowed IPv4.
		{"[::ffff:192.168.225.7]:51234", true},
		// Bare host (no port) is supported too.
		{"192.168.225.7", true},
		// Blocked: outside prefix.
		{"10.0.0.1:51234", false},
		{"100.64.5.7:51234", false}, // CGNAT — common cellular range
		{"8.8.8.8:51234", false},
		{"[2001:db8::1]:51234", false},
		// Malformed → fail closed.
		{"garbage", false},
		{"", false},
	}
	for _, c := range cases {
		if got := m.Allow(c.addr); got != c.want {
			t.Errorf("Allow(%q) = %v, want %v", c.addr, got, c.want)
		}
	}
}

func TestEmpty_AllowsAll(t *testing.T) {
	m, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Allow("8.8.8.8:1234") {
		t.Error("empty list should allow everything")
	}
	if !m.Allow("garbage") {
		t.Error("empty list should allow even malformed (allow-all = no enforcement)")
	}
}

func TestNew_BadCIDR(t *testing.T) {
	if _, err := New([]string{"not-a-cidr"}); err == nil {
		t.Error("expected error for malformed CIDR")
	}
}

func TestNew_BlankEntriesIgnored(t *testing.T) {
	m, err := New([]string{"  ", "127.0.0.0/8", ""})
	if err != nil {
		t.Fatal(err)
	}
	if !m.Allow("127.0.0.1:0") {
		t.Error("expected 127.0.0.1 to be allowed")
	}
	if m.Allow("8.8.8.8:0") {
		t.Error("expected 8.8.8.8 to be blocked")
	}
}
