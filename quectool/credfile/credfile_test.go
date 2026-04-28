package credfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials")

	if err := Write(p, "admin", "$2a$10$AbCdEf.GhIjKl"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	user, hash, err := Read(p)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if user != "admin" || hash != "$2a$10$AbCdEf.GhIjKl" {
		t.Errorf("got %q/%q, want admin/$2a...", user, hash)
	}

	// Hashes can contain ':' inside? No — bcrypt only uses [./A-Za-z0-9$].
	// But verify a hash with '$' (the only special char we care about) is
	// preserved verbatim through round-trip.
	hashWithDollar := "$2a$10$" + "n.GVl/eTVQwQ4FUDmhCgOOVWNa3eYqQz9xoRaZ8bmzABCDEFG12345"
	if err := Write(p, "admin", hashWithDollar); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_, hash, err = Read(p)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if hash != hashWithDollar {
		t.Errorf("hash mangled: %q", hash)
	}
}

func TestRead_Missing(t *testing.T) {
	_, _, err := Read(filepath.Join(t.TempDir(), "nope"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("want os.ErrNotExist, got %v", err)
	}
}

func TestRead_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials")
	if err := os.WriteFile(p, []byte("\n# only a comment\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Read(p)
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("want ErrEmpty, got %v", err)
	}
}

func TestRead_MalformedLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials")
	if err := os.WriteFile(p, []byte("no-colon-here\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(p); err == nil {
		t.Error("expected error on malformed line")
	}
}

func TestRead_IgnoresCommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials")
	body := "# header comment\n" +
		"\n" +
		"   \n" +
		"# another\n" +
		"admin:hash\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	user, hash, err := Read(p)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if user != "admin" || hash != "hash" {
		t.Errorf("got %q/%q", user, hash)
	}
}

func TestWrite_Mode0600(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials")
	if err := Write(p, "admin", "hash"); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 0600", st.Mode().Perm())
	}
}

func TestWrite_RejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "credentials")
	cases := []struct{ user, hash string }{
		{"", "hash"},
		{"admin", ""},
		{"adm:in", "hash"},
		{"admin\n", "hash"},
		{"admin", "ha\nsh"},
	}
	for _, c := range cases {
		if err := Write(p, c.user, c.hash); err == nil {
			t.Errorf("Write(%q, %q) should have errored", c.user, c.hash)
		}
	}
}
