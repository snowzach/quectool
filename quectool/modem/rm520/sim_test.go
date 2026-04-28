package rm520

import (
	"reflect"
	"testing"

	"github.com/snowzach/quectool/quectool/modem"
)

func TestParseSim(t *testing.T) {
	// Representative response lines aggregated from multiple AT calls.
	// Replace fixtures with captured hardware output during implementation.
	in := simInputs{
		slot:      []string{`+QUIMSLOT: 1`},
		imsi:      []string{`310260123456789`},
		iccid:     []string{`+QCCID: 8901260012345678901F`},
		cops:      []string{`+COPS: 0,0,"T-Mobile",7`},
		cgcontrdp: []string{`+CGCONTRDP: 1,5,"internet","10.0.0.2.255.255.255.0","10.0.0.1","8.8.8.8","8.8.4.4"`},
		cgdcont:   []string{`+CGDCONT: 1,"IPV4V6","internet","",0,0`},
		cgreg:     []string{`+CGREG: 0,1`},
		cereg:     []string{`+CEREG: 0,0`},
		c5greg:    []string{`+C5GREG: 0,0`},
	}
	got, err := parseSim(in)
	if err != nil {
		t.Fatalf("parseSim: %v", err)
	}
	want := &modem.SimInfo{
		Slot:       1,
		IMSI:       "310260123456789",
		ICCID:      "8901260012345678901F",
		Operator:   "T-Mobile",
		APN:        "internet",
		APNIP:      "IPV4V6",
		Registered: true,
		IPv4:       "10.0.0.2",
		Gateway:    "10.0.0.1",
		DNS:        []string{"8.8.8.8", "8.8.4.4"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

// TestParseSim_IPv6Only covers 5GSA where CGCONTRDP returns just the
// 16-octet IPv6 form with no mask, no gateway, no DNS — captured from
// real RM520N-GL traffic on T-Mobile.
func TestParseSim_IPv6Only(t *testing.T) {
	in := simInputs{
		imsi:      []string{`310260000000001`},
		cops:      []string{`+COPS: 0,0,"T-Mobile",11`},
		cgcontrdp: []string{`+CGCONTRDP: 1,0,"fast.t-mobile.com","38.7.251.145.22.149.209.164.10.210.41.87.105.185.93.227"`},
		c5greg:    []string{`+C5GREG: 0,1`},
	}
	got, err := parseSim(in)
	if err != nil {
		t.Fatalf("parseSim: %v", err)
	}
	if got.IPv6 != "2607:fb91:1695:d1a4:ad2:2957:69b9:5de3" {
		t.Errorf("IPv6: got %q", got.IPv6)
	}
	if got.IPv4 != "" {
		t.Errorf("IPv4 should be empty, got %q", got.IPv4)
	}
}

// TestParseSim_DualStack covers IPv4v6 where the modem emits two
// CGCONTRDP lines for the same cid — one per family — and we merge them.
func TestParseSim_DualStack(t *testing.T) {
	in := simInputs{
		imsi: []string{`310260000000002`},
		cops: []string{`+COPS: 0,0,"T-Mobile",11`},
		cgcontrdp: []string{
			`+CGCONTRDP: 1,5,"internet","100.64.1.10.255.255.255.0","100.64.1.1","8.8.8.8","8.8.4.4"`,
			`+CGCONTRDP: 1,5,"internet","38.7.251.145.22.149.209.164.10.210.41.87.105.185.93.227"`,
		},
		c5greg: []string{`+C5GREG: 0,1`},
	}
	got, err := parseSim(in)
	if err != nil {
		t.Fatalf("parseSim: %v", err)
	}
	if got.IPv4 != "100.64.1.10" {
		t.Errorf("IPv4: got %q", got.IPv4)
	}
	if got.IPv6 != "2607:fb91:1695:d1a4:ad2:2957:69b9:5de3" {
		t.Errorf("IPv6: got %q", got.IPv6)
	}
	if got.Gateway != "100.64.1.1" {
		t.Errorf("Gateway: got %q", got.Gateway)
	}
	if !reflect.DeepEqual(got.DNS, []string{"8.8.8.8", "8.8.4.4"}) {
		t.Errorf("DNS: got %v", got.DNS)
	}
}

func TestParseSim_Missing(t *testing.T) {
	// All inputs empty: function should return ErrFieldMissing for required
	// fields rather than panic.
	_, err := parseSim(simInputs{})
	if err == nil {
		t.Fatal("expected error for empty inputs")
	}
}

// TestParseSim_5GSA covers a real captured RM520N-GL state on 5G-SA where
// only +C5GREG reports registered, +CGREG and +CEREG both say "0,0".
func TestParseSim_5GSA(t *testing.T) {
	in := simInputs{
		slot:      []string{`+QUIMSLOT: 1`},
		imsi:      []string{`310260393092491`},
		iccid:     []string{`+QCCID: 8901260397730924913F`},
		cops:      []string{`+COPS: 0,0,"T-Mobile",11`},
		cgcontrdp: []string{`+CGCONTRDP: 1,0,"fast.t-mobile.com","38.7.251.145.22.149.209.164.10.210.41.87.105.185.93.227"`},
		cgdcont:   []string{`+CGDCONT: 1,"IPV4V6","fast.t-mobile.com","0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0",0,0,0,0,,,,,,,,,"",,,,0`},
		cgreg:     []string{`+CGREG: 0,0`},
		cereg:     []string{`+CEREG: 0,0`},
		c5greg:    []string{`+C5GREG: 0,1`},
	}
	got, err := parseSim(in)
	if err != nil {
		t.Fatalf("parseSim: %v", err)
	}
	if !got.Registered {
		t.Errorf("expected Registered=true (C5GREG stat=1): %+v", got)
	}
	if got.APN != "fast.t-mobile.com" || got.APNIP != "IPV4V6" {
		t.Errorf("unexpected APN/APNIP: %+v", got)
	}
	if got.Operator != "T-Mobile" {
		t.Errorf("unexpected Operator: %q", got.Operator)
	}
}
