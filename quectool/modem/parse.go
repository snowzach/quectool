package modem

import (
	"strconv"
	"strings"
)

// FindLine returns the first line in lines that begins with prefix (after
// stripping leading whitespace). Returns ("", false) if no match.
func FindLine(lines []string, prefix string) (string, bool) {
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimLeft(l, " \t"), prefix) {
			return l, true
		}
	}
	return "", false
}

// FindLines returns every line beginning with prefix.
func FindLines(lines []string, prefix string) []string {
	var out []string
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimLeft(l, " \t"), prefix) {
			out = append(out, l)
		}
	}
	return out
}

// SplitCSV splits a CSV-style AT field list, honoring double-quoted fields
// (so commas inside quotes are preserved). Quotes are stripped from output.
func SplitCSV(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ',' && !inQuote:
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	out = append(out, b.String())
	return out
}

// ParseInt parses a decimal int after trimming whitespace. ok=false on any
// parse error so callers can default cleanly.
func ParseInt(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, false
	}
	return n, true
}

// ParseQuoted strips a single pair of surrounding double quotes if present.
func ParseQuoted(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
