package modem

import (
	"reflect"
	"testing"
)

func TestFindLine(t *testing.T) {
	lines := []string{
		`+QUIMSLOT: 1`,
		`+CGCONTRDP: 1,5,"internet","10.0.0.2"`,
		`OK`,
	}
	tests := []struct {
		name   string
		prefix string
		want   string
		ok     bool
	}{
		{"present", "+QUIMSLOT:", `+QUIMSLOT: 1`, true},
		{"prefix-of-multiple", "+CG", `+CGCONTRDP: 1,5,"internet","10.0.0.2"`, true},
		{"absent", "+NOPE:", "", false},
		{"empty-input", "+QUIMSLOT:", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := lines
			if tc.name == "empty-input" {
				input = nil
			}
			got, ok := FindLine(input, tc.prefix)
			if ok != tc.ok || got != tc.want {
				t.Errorf("got (%q, %v), want (%q, %v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestFindLines(t *testing.T) {
	lines := []string{
		`+QENG: "servingcell","NOCONN"`,
		`+QENG: "neighbourcell intra","LTE",100`,
		`+QENG: "neighbourcell intra","LTE",200`,
		`OK`,
	}
	got := FindLines(lines, "+QENG:")
	if len(got) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(got), got)
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{`1,2,3`, []string{"1", "2", "3"}},
		{`"a","b,c","d"`, []string{`a`, `b,c`, `d`}},
		{`1,"two",3`, []string{"1", "two", "3"}},
		{``, []string{""}},
		{`"unterminated`, []string{`unterminated`}},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got := SplitCSV(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SplitCSV(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"42", 42, true},
		{"-7", -7, true},
		{"0", 0, true},
		{"", 0, false},
		{"abc", 0, false},
		{" 3 ", 3, true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := ParseInt(tc.in)
			if got != tc.want || ok != tc.ok {
				t.Errorf("ParseInt(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestParseQuoted(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`"hello"`, "hello"},
		{`hello`, "hello"},
		{`""`, ""},
		{`"a"b"`, `a"b`},
		{``, ""},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := ParseQuoted(tc.in); got != tc.want {
				t.Errorf("ParseQuoted(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
