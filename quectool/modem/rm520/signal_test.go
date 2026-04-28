package rm520

import (
	"testing"
)

func TestParseSignal_LTE(t *testing.T) {
	// Captured representative LTE response (trailing comma, RSSI between RSRQ
	// and SINR).
	lines := []string{
		`+QENG: "servingcell","NOCONN","LTE","FDD",310,260,1F2A801,193,2200,4,5,5,0xfffe,-95,-12,-65,15,`,
	}
	got, err := parseSignal(lines)
	if err != nil {
		t.Fatalf("parseSignal: %v", err)
	}
	if got.Tech != "LTE" || got.RSRP != -95 || got.RSRQ != -12 || got.SINR != 15 {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseSignal_NR5G_SA(t *testing.T) {
	// Captured from a real RM520N-GL on T-Mobile.
	lines := []string{
		`+QENG: "servingcell","NOCONN","NR5G-SA","FDD",310,260,106497003,792,B0FB00,126270,71,3,-98,-12,7,0,18`,
	}
	got, err := parseSignal(lines)
	if err != nil {
		t.Fatalf("parseSignal: %v", err)
	}
	if got.Tech != "NR5G-SA" || got.RSRP != -98 || got.RSRQ != -12 || got.SINR != 7 {
		t.Errorf("unexpected: %+v", got)
	}
}
