package rm520

import (
	"reflect"
	"strings"
	"testing"

	"github.com/snowzach/quectool/quectool/modem"
)

func TestParseSettings_AUTO(t *testing.T) {
	in := settingsInputs{
		cgdcont: []string{
			`+CGDCONT: 1,"IPV4V6","internet","",0,0`,
		},
		modePref:   []string{`+QNWPREFCFG: "mode_pref",AUTO`},
		nr5gMode:   []string{`+QNWPREFCFG: "nr5g_disable_mode",0`},
		cellLock4G: []string{`+QNWLOCK: "common/4g",0`},
		cellLock5G: []string{`+QNWLOCK: "common/5g",0`},
		lteBand:    []string{`+QNWPREFCFG: "lte_band",1:3:7:20`},
		nr5gBand:    []string{`+QNWPREFCFG: "nr5g_band",1:3:78`},
		nsaNr5gBand: []string{`+QNWPREFCFG: "nsa_nr5g_band",1:3:78`},
	}
	got, err := parseSettings(in)
	if err != nil {
		t.Fatalf("parseSettings: %v", err)
	}
	want := &modem.Settings{
		APN:      modem.APNSettings{Name: "internet", IPType: "IPV4V6"},
		Network:  modem.NetworkSettings{Mode: "AUTO"},
		Bands:    modem.BandSet{LTE: []int{1, 3, 7, 20}, NR5G: []int{1, 3, 78}},
		CellLock: modem.CellLockSettings{Lock4G: false, Lock5G: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestModeFromAT(t *testing.T) {
	cases := []struct {
		mp, nd, want string
	}{
		{"AUTO", "0", "AUTO"},
		{"LTE", "0", "LTE_ONLY"},
		{"LTE", "2", "LTE_ONLY"},
		{"NR5G", "2", "NR5G_SA_ONLY"},
		{"AUTO", "1", "NR5G_NSA_ONLY"},
		{"LTE:NR5G", "1", "NR5G_NSA_ONLY"},
		{"NR5G", "1", "CUSTOM"}, // SA-disabled + NR5G-forced = no path to a cell
		{"WAT", "0", "CUSTOM"},
	}
	for _, c := range cases {
		if got := modeFromAT(c.mp, c.nd); got != c.want {
			t.Errorf("modeFromAT(%q,%q)=%q, want %q", c.mp, c.nd, got, c.want)
		}
	}
}

func TestBuildSetCommands_APN(t *testing.T) {
	u := &modem.SettingsUpdate{APN: &modem.APNSettings{Name: "newapn", IPType: "IPV4V6"}}
	cmds := buildSetCommands(u, fallbackLTEBands, fallbackNR5GBands)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command (cgdcont), got %d: %v", len(cmds), cmds)
	}
	if !strings.Contains(cmds[0], `"IPV4V6","newapn"`) {
		t.Errorf("expected CGDCONT to set APN+IPType: %q", cmds[0])
	}
}

func TestBuildSetCommands_NetworkMode(t *testing.T) {
	cases := []struct {
		mode      string
		wantCount int
		mustHave  []string
	}{
		{"AUTO", 2, []string{`"mode_pref",AUTO`, `"nr5g_disable_mode",0`}},
		{"LTE_ONLY", 2, []string{`"mode_pref",LTE`, `"nr5g_disable_mode",0`}},
		{"NR5G_SA_ONLY", 2, []string{`"mode_pref",NR5G`, `"nr5g_disable_mode",2`}},
		{"NR5G_NSA_ONLY", 2, []string{`"mode_pref",AUTO`, `"nr5g_disable_mode",1`}},
		{"CUSTOM", 0, nil}, // unrecognized mode → no commands
	}
	for _, c := range cases {
		u := &modem.SettingsUpdate{Network: &modem.NetworkSettings{Mode: c.mode}}
		got := buildSetCommands(u, fallbackLTEBands, fallbackNR5GBands)
		if len(got) != c.wantCount {
			t.Errorf("mode=%q: expected %d commands, got %d: %v", c.mode, c.wantCount, len(got), got)
			continue
		}
		joined := strings.Join(got, " ; ")
		for _, want := range c.mustHave {
			if !strings.Contains(joined, want) {
				t.Errorf("mode=%q: expected output to contain %q; got %q", c.mode, want, joined)
			}
		}
	}
}

func TestBuildSetCommands_Bands(t *testing.T) {
	u := &modem.SettingsUpdate{Bands: &modem.BandSet{LTE: []int{1, 3, 7}, NR5G: []int{41, 71}}}
	cmds := buildSetCommands(u, fallbackLTEBands, fallbackNR5GBands)
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands (lte_band + nr5g_band + nsa_nr5g_band), got %d: %v", len(cmds), cmds)
	}
	joined := strings.Join(cmds, " ; ")
	if !strings.Contains(joined, `"lte_band",1:3:7`) {
		t.Errorf("expected lte_band 1:3:7 in: %q", joined)
	}
	if !strings.Contains(joined, `"nr5g_band",41:71`) {
		t.Errorf("expected nr5g_band 41:71 in: %q", joined)
	}
	if !strings.Contains(joined, `"nsa_nr5g_band",41:71`) {
		t.Errorf("expected nsa_nr5g_band 41:71 in: %q", joined)
	}
}

// Writing the empty string disables every band; callers must expand empty
// to the supported list before invoking joinBands.
func TestJoinBands_Empty(t *testing.T) {
	if got := joinBands(nil); got != "" {
		t.Errorf("joinBands(nil)=%q, want \"\"", got)
	}
}

// An empty Bands update must expand to the supported list, never write 0.
func TestBuildSetCommands_BandsEmpty_ExpandsToSupported(t *testing.T) {
	u := &modem.SettingsUpdate{Bands: &modem.BandSet{}}
	cmds := buildSetCommands(u, []int{1, 3, 41}, []int{71, 78})
	if len(cmds) != 3 {
		t.Fatalf("got %d cmds: %v", len(cmds), cmds)
	}
	joined := strings.Join(cmds, " ; ")
	if !strings.Contains(joined, `"lte_band",1:3:41`) {
		t.Errorf("missing expanded LTE list: %q", joined)
	}
	if !strings.Contains(joined, `"nr5g_band",71:78`) {
		t.Errorf("missing expanded NR5G list: %q", joined)
	}
	if !strings.Contains(joined, `"nsa_nr5g_band",71:78`) {
		t.Errorf("missing expanded NSA NR5G list: %q", joined)
	}
}

// CellLock unlocks are no longer in buildSetCommands — they're emitted by
// SetSettings with error tolerance so an "already unlocked" rejection
// doesn't fail the whole save.
func TestBuildSetCommands_CellLockNoOp(t *testing.T) {
	u := &modem.SettingsUpdate{CellLock: &modem.CellLockSettings{Lock4G: false, Lock5G: false}}
	cmds := buildSetCommands(u, fallbackLTEBands, fallbackNR5GBands)
	if len(cmds) != 0 {
		t.Errorf("expected no commands from buildSetCommands for cell_lock, got %d: %v", len(cmds), cmds)
	}
}
