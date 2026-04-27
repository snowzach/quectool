package rm520

import (
	"testing"
)

func TestParseCell_LTE(t *testing.T) {
	serving := []string{
		`+QENG: "servingcell","NOCONN","LTE","FDD",310,260,1F2A801,193,2200,4,5,5,0xfffe,-95,-12,-65,15,`,
	}
	neighbors := []string{
		`+QENG: "neighbourcell intra","LTE",2200,193,-,-,-95,-12,-65,15,7,-,-,-,-,-`,
		`+QENG: "neighbourcell intra","LTE",2200,194,-,-,-100,-15,-70,10,7,-,-,-,-,-`,
	}
	got, err := parseCell(serving, neighbors, nil)
	if err != nil {
		t.Fatalf("parseCell: %v", err)
	}
	if got.Tech != "LTE" || got.MCC != "310" || got.MNC != "260" || got.PCI != 193 {
		t.Errorf("unexpected serving: %+v", got)
	}
	if len(got.Neighbors) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(got.Neighbors))
	}
}

// NR5G-NSA reports two servingcell lines: an LTE anchor and an NR5G
// secondary. The first becomes the primary; the second goes in SCells.
func TestParseCell_NSA_TwoServingCells(t *testing.T) {
	serving := []string{
		`+QENG: "servingcell","NOCONN","LTE","FDD",310,260,1F2A801,193,2200,4,5,5,0xfffe,-95,-12,-65,15,`,
		`+QENG: "servingcell","NOCONN","NR5G-NSA","FDD",310,260,106497003,792,B0FB00,126270,71,3,-98,-12,7,0,18`,
	}
	got, err := parseCell(serving, nil, nil)
	if err != nil {
		t.Fatalf("parseCell: %v", err)
	}
	if got.Tech != "LTE" {
		t.Errorf("primary tech = %q, want LTE", got.Tech)
	}
	if len(got.SCells) != 1 {
		t.Fatalf("expected 1 SCell, got %d", len(got.SCells))
	}
	if got.SCells[0].Tech != "NR5G-NSA" || got.SCells[0].Band != "71" {
		t.Errorf("SCell wrong: %+v", got.SCells[0])
	}
}

// AT+QCAINFO reports the primary plus secondary carriers. PCC is dropped
// (already covered by +QENG); SCC entries become SCells.
func TestParseCell_QCAInfo(t *testing.T) {
	serving := []string{
		`+QENG: "servingcell","NOCONN","LTE","FDD",310,260,1F2A801,193,2200,4,5,5,0xfffe,-95,-12,-65,15,`,
	}
	cainfo := []string{
		`+QCAINFO: "PCC",2200,5,"LTE BAND 4",1`,
		`+QCAINFO: "SCC",1825,5,"LTE BAND 2",1,255,-95,-12`,
		`+QCAINFO: "SCC",6300,5,"LTE BAND 41",1,42,-100,-15`,
	}
	got, err := parseCell(serving, nil, cainfo)
	if err != nil {
		t.Fatalf("parseCell: %v", err)
	}
	if len(got.SCells) != 2 {
		t.Fatalf("expected 2 SCells (PCC dropped), got %d", len(got.SCells))
	}
	if got.SCells[0].Band != "2" || got.SCells[1].Band != "41" {
		t.Errorf("SCell bands wrong: %+v / %+v", got.SCells[0], got.SCells[1])
	}
	if got.SCells[1].PCI != 42 {
		t.Errorf("SCC PCI not parsed: %d", got.SCells[1].PCI)
	}
}
