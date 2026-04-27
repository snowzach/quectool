package rm520

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

type settingsInputs struct {
	cgdcont     []string
	modePref    []string
	nr5gMode    []string
	cellLock4G  []string
	cellLock5G  []string
	lteBand     []string
	nr5gBand    []string
	nsaNr5gBand []string
}

func (m *Modem) GetSettings(ctx context.Context) (*modem.Settings, error) {
	send := func(cmd string) ([]string, error) {
		resp, err := m.at.SendCMD(ctx, cmd, 3*time.Second)
		if err != nil {
			return nil, err
		}
		if resp.Status != atserver.ATStatusOK {
			return nil, nil
		}
		return resp.Response, nil
	}
	in := settingsInputs{}
	var err error
	if in.cgdcont, err = send("AT+CGDCONT?"); err != nil {
		return nil, err
	}
	if in.modePref, err = send(`AT+QNWPREFCFG="mode_pref"`); err != nil {
		return nil, err
	}
	if in.nr5gMode, err = send(`AT+QNWPREFCFG="nr5g_disable_mode"`); err != nil {
		return nil, err
	}
	if in.cellLock4G, err = send(`AT+QNWLOCK="common/4g"`); err != nil {
		return nil, err
	}
	if in.cellLock5G, err = send(`AT+QNWLOCK="common/5g"`); err != nil {
		return nil, err
	}
	if in.lteBand, err = send(`AT+QNWPREFCFG="lte_band"`); err != nil {
		return nil, err
	}
	if in.nr5gBand, err = send(`AT+QNWPREFCFG="nr5g_band"`); err != nil {
		return nil, err
	}
	if in.nsaNr5gBand, err = send(`AT+QNWPREFCFG="nsa_nr5g_band"`); err != nil {
		return nil, err
	}
	return parseSettings(in)
}

// trailingValue returns the part of "+TAG: \"k\",VALUE" after the last comma.
func trailingValue(line string) string {
	_, val, _ := strings.Cut(line, ":")
	idx := strings.LastIndex(val, ",")
	if idx < 0 {
		return strings.TrimSpace(val)
	}
	return strings.TrimSpace(val[idx+1:])
}

// secondField returns the field at index 1 of "+TAG: A,B[,...]" (zero-indexed
// after the colon). Used for QNWLOCK responses where the state lives there.
func secondField(line string) string {
	_, val, _ := strings.Cut(line, ":")
	parts := modem.SplitCSV(val)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// parseBands turns a colon-joined band list into a slice of band numbers.
// Empirically on RM520N-GL, the modem reports `0` when no bands are enabled
// (which leaves the radio stuck in SEARCH) — there is no "0 = all" sentinel.
// We still drop the literal 0 here because writing it back rejects, but the
// caller should treat an empty result as "no bands selected" rather than
// "all bands enabled".
func parseBands(line string) []int {
	v := trailingValue(line)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ":")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n != 0 {
			out = append(out, n)
		}
	}
	return out
}

// modeFromAT maps the (mode_pref, nr5g_disable_mode) pair to one of the four
// canonical NetworkMode values. Unrecognized combinations return "CUSTOM" so
// the UI can render the state read-only without crashing.
//
// nr5g_disable_mode semantics on Quectel R03 firmware (verified empirically):
//   0 = nothing disabled (both NSA and SA available)
//   1 = SA disabled (NSA only)
//   2 = NSA disabled (SA only) — also lets nr5g_band actually persist
func modeFromAT(modePref, nr5gDisable string) string {
	mp := strings.ToUpper(strings.TrimSpace(modePref))
	nd := strings.TrimSpace(nr5gDisable)
	switch {
	case mp == "AUTO" && nd == "0":
		return "AUTO"
	case mp == "LTE":
		return "LTE_ONLY"
	case mp == "NR5G" && nd == "2":
		return "NR5G_SA_ONLY"
	case (mp == "AUTO" || mp == "LTE:NR5G" || mp == "NR5G:LTE") && nd == "1":
		return "NR5G_NSA_ONLY"
	default:
		return "CUSTOM"
	}
}

// modeToAT is the inverse: given a canonical mode, return the AT values for
// mode_pref and nr5g_disable_mode that achieve it. Returns ok=false for
// "CUSTOM" or unknown modes (caller must skip writing those).
func modeToAT(mode string) (modePref, nr5gDisable string, ok bool) {
	switch mode {
	case "AUTO":
		return "AUTO", "0", true
	case "LTE_ONLY":
		// nr5g_disable_mode is irrelevant when LTE-only, but Quectel
		// rejects setting mode_pref alone if a stale NR mode lingers;
		// always reset it to 0 here for predictability.
		return "LTE", "0", true
	case "NR5G_NSA_ONLY":
		return "AUTO", "1", true
	case "NR5G_SA_ONLY":
		// mode_pref=NR5G forces NR (no LTE fallback); nd=2 disables NSA so
		// this is true SA-only. With this combo nr5g_band restrictions
		// actually persist — the (NR5G, 1) combo silently wipes them to 0.
		return "NR5G", "2", true
	default:
		return "", "", false
	}
}

func parseSettings(in settingsInputs) (*modem.Settings, error) {
	out := &modem.Settings{}

	if line, ok := modem.FindLine(in.cgdcont, "+CGDCONT: 1,"); ok {
		_, val, _ := strings.Cut(line, ":")
		fields := modem.SplitCSV(val)
		if len(fields) >= 3 {
			out.APN.IPType = modem.ParseQuoted(strings.TrimSpace(fields[1]))
			out.APN.Name = modem.ParseQuoted(strings.TrimSpace(fields[2]))
		}
	}
	var modePref, nr5gMode string
	if line, ok := modem.FindLine(in.modePref, "+QNWPREFCFG:"); ok {
		modePref = trailingValue(line)
	}
	if line, ok := modem.FindLine(in.nr5gMode, "+QNWPREFCFG:"); ok {
		nr5gMode = trailingValue(line)
	}
	out.Network.Mode = modeFromAT(modePref, nr5gMode)

	// QNWLOCK readback formats:
	//   unlocked: +QNWLOCK: "common/4g",0          /  +QNWLOCK: "common/5g",0
	//   4g lock:  +QNWLOCK: "common/4g",1,<freq>,<pci>
	//   5g lock:  +QNWLOCK: "common/5g",<pci>,<freq>,<scs_kHz>,<band>
	// "Locked" reduces to "field after the key isn't 0" — for 4g it's the
	// enable bit, for 5g it's the PCI (always nonzero on a real cell).
	if line, ok := modem.FindLine(in.cellLock4G, "+QNWLOCK:"); ok {
		out.CellLock.Lock4G = secondField(line) != "0" && secondField(line) != ""
	}
	if line, ok := modem.FindLine(in.cellLock5G, "+QNWLOCK:"); ok {
		out.CellLock.Lock5G = secondField(line) != "0" && secondField(line) != ""
	}
	if line, ok := modem.FindLine(in.lteBand, "+QNWPREFCFG:"); ok {
		out.Bands.LTE = parseBands(line)
	}
	// NR5G bands live in two parallel fields: nr5g_band (used in SA mode)
	// and nsa_nr5g_band (used in NSA mode). The active one depends on
	// nr5g_disable_mode — read whichever applies. Falls back to the other
	// field if the primary one is empty so the UI never shows a blank list
	// just because the user is in a mode that doesn't populate it.
	saLine, _ := modem.FindLine(in.nr5gBand, "+QNWPREFCFG:")
	nsaLine, _ := modem.FindLine(in.nsaNr5gBand, "+QNWPREFCFG:")
	saBands := parseBands(saLine)
	nsaBands := parseBands(nsaLine)
	if out.Network.Mode == "NR5G_NSA_ONLY" {
		out.Bands.NR5G = pickFirstNonEmpty(nsaBands, saBands)
	} else {
		out.Bands.NR5G = pickFirstNonEmpty(saBands, nsaBands)
	}
	return out, nil
}

func pickFirstNonEmpty(a, b []int) []int {
	if len(a) > 0 {
		return a
	}
	return b
}

// buildSetCommands turns a partial update into the AT commands needed to
// apply it. CellLock here only honors unlock requests — locking requires
// the EARFCN+PCI of a specific cell and is exposed via LockCurrentCell.
//
// supportedLTE/supportedNR5G are used as the "all bands" expansion when the
// caller sends an empty Bands list — writing 0 disables the band entirely.
func buildSetCommands(u *modem.SettingsUpdate, supportedLTE, supportedNR5G []int) []string {
	var cmds []string
	if u.APN != nil {
		ipt := u.APN.IPType
		if ipt == "" {
			ipt = "IPV4V6"
		}
		cmds = append(cmds, fmt.Sprintf(`AT+CGDCONT=1,"%s","%s"`, ipt, u.APN.Name))
	}
	if u.Network != nil {
		if mp, nd, ok := modeToAT(u.Network.Mode); ok {
			cmds = append(cmds,
				fmt.Sprintf(`AT+QNWPREFCFG="mode_pref",%s`, mp),
				fmt.Sprintf(`AT+QNWPREFCFG="nr5g_disable_mode",%s`, nd),
			)
		}
	}
	if u.Bands != nil {
		// Empty list means "everything supported" — write the full datasheet
		// list explicitly. Writing 0 leaves the modem with no bands enabled
		// and stuck in SEARCH (verified on RM520N-GL firmware RM520NGLAAR03A01M4G).
		lte := u.Bands.LTE
		if len(lte) == 0 {
			lte = supportedLTE
		}
		nr5g := u.Bands.NR5G
		if len(nr5g) == 0 {
			nr5g = supportedNR5G
		}
		cmds = append(cmds, fmt.Sprintf(`AT+QNWPREFCFG="lte_band",%s`, joinBands(lte)))
		// Write the same NR5G list to both fields so the user's selection
		// applies whether the modem ends up on SA (nr5g_band) or NSA
		// (nsa_nr5g_band). The modem ignores the inactive field.
		cmds = append(cmds, fmt.Sprintf(`AT+QNWPREFCFG="nr5g_band",%s`, joinBands(nr5g)))
		cmds = append(cmds, fmt.Sprintf(`AT+QNWPREFCFG="nsa_nr5g_band",%s`, joinBands(nr5g)))
	}
	// CellLock unlock commands are emitted by SetSettings, not here, because
	// the modem rejects "unlock" on a tech that isn't currently locked and
	// the frontend always sends the full {lock_4g, lock_5g} object — we
	// can't tell from u alone what's actually changing. SetSettings runs
	// each unlock with an "already unlocked" tolerance.
	return cmds
}

// joinBands formats a band list for AT+QNWPREFCFG="lte_band"/"nr5g_band".
// Caller is responsible for ensuring the list is non-empty (writing "0"
// disables every band).
func joinBands(bs []int) string {
	parts := make([]string, 0, len(bs))
	for _, b := range bs {
		if b == 0 {
			continue
		}
		parts = append(parts, strconv.Itoa(b))
	}
	return strings.Join(parts, ":")
}

func (m *Modem) SetSettings(ctx context.Context, u *modem.SettingsUpdate) (*modem.Settings, error) {
	for _, c := range buildSetCommands(u, m.supportedLTEBands, m.supportedNR5GBands) {
		resp, err := m.at.SendCMD(ctx, c, 5*time.Second)
		if err != nil {
			return nil, err
		}
		if resp.Status != atserver.ATStatusOK {
			return nil, fmt.Errorf("AT command rejected: %s", c)
		}
	}
	// Cell-lock unlock requests are run separately and tolerantly. The
	// frontend sends the full cell_lock object on every save, so we get
	// {lock_4g:false, lock_5g:false} even when only one is really changing.
	// `AT+QNWLOCK="common/4g",0` on an already-unlocked 4G stack errors —
	// we treat that as success because the requested state is achieved.
	if u.CellLock != nil {
		for _, p := range []struct {
			want bool
			cmd  string
		}{
			{u.CellLock.Lock4G, `AT+QNWLOCK="common/4g",0`},
			{u.CellLock.Lock5G, `AT+QNWLOCK="common/5g",0`},
		} {
			if p.want {
				continue // lock=true is not honored here (use LockCurrentCell/LockCell)
			}
			if _, err := m.at.SendCMD(ctx, p.cmd, 5*time.Second); err != nil {
				return nil, err
			}
			// Don't surface non-OK status — most likely "already unlocked".
		}
	}
	// Mode and band writes apply immediately on RM520N-GL A03 firmware —
	// the modem reattaches on its own. Earlier attempts to "help" with
	// COPS=2/0 backfired because COPS=0 silently resets mode_pref to AUTO,
	// undoing the very change we just made.
	return m.GetSettings(ctx)
}

// Reboot issues AT+CFUN=1,1 — full modem reset including SIM. The modem
// disappears from USB for ~15-20s before it comes back.
func (m *Modem) Reboot(ctx context.Context) error {
	_, err := m.at.SendCMD(ctx, "AT+CFUN=1,1", 5*time.Second)
	return err
}

// SetDataPath writes AT+QCFG="data_interface",<primary>,<aux> with primary
// set to the requested path and aux set to the other (so the inactive
// interface stays available for AT). Doesn't auto-reboot — caller should
// invoke Reboot afterward for the change to take effect.
func (m *Modem) SetDataPath(ctx context.Context, path string) error {
	var primary, aux int
	switch path {
	case "USB":
		primary, aux = 0, 1
	case "PCIe":
		primary, aux = 1, 0
	default:
		return fmt.Errorf("invalid path %q (want USB or PCIe)", path)
	}
	cmd := fmt.Sprintf(`AT+QCFG="data_interface",%d,%d`, primary, aux)
	resp, err := m.at.SendCMD(ctx, cmd, 5*time.Second)
	if err != nil {
		return err
	}
	if resp.Status != atserver.ATStatusOK {
		return fmt.Errorf("data_interface set rejected")
	}
	return nil
}
