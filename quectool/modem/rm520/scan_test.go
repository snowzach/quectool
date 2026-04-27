package rm520

import (
	"testing"
)

func TestParseScan(t *testing.T) {
	// +COPS: (3,"T-Mobile","T-Mobile","310260",7),(2,"AT&T","AT&T","310410",7),,(0-4),(0-2)
	lines := []string{
		`+COPS: (3,"T-Mobile","T-Mobile","310260",7),(2,"AT&T","AT&T","310410",7),,(0-4),(0-2)`,
	}
	got, err := parseScan(lines)
	if err != nil {
		t.Fatalf("parseScan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(got), got)
	}
	if got[0].Operator != "T-Mobile" || got[0].MCC != "310" || got[0].MNC != "260" {
		t.Errorf("unexpected first: %+v", got[0])
	}
}
