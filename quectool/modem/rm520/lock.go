package rm520

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

// scsIndexToKHz maps QENG/QSCAN's SCS index to the kHz value AT+QNWLOCK
// requires. Quectel's QENG reports an integer 0..4 representing the spacing
// step; the lock command wants the actual kHz number.
//
//	0 → 15 kHz   (n5, n8, n12, n13, n20, n28, n66, n71, n75, n76, ...)
//	1 → 30 kHz   (n7, n25, n38, n40, n41, n66 alt, n78, n79, ...)
//	2 → 60 kHz   (rare on FR1)
//	3 → 120 kHz  (FR2 mmWave)
//	4 → 240 kHz  (mmWave reference)
func scsIndexToKHz(idx int) int {
	switch idx {
	case 0:
		return 15
	case 1:
		return 30
	case 2:
		return 60
	case 3:
		return 120
	case 4:
		return 240
	default:
		return 15 // safest fallback for unknown low-band cells
	}
}

// LockCurrentCell pins the modem to its current serving cell on the given
// technology. tech is "4g" or "5g". Reads EARFCN/ARFCN and PCI from the
// matching +QENG: "servingcell" line and issues:
//
//	AT+QNWLOCK="common/4g",1,<EARFCN>,<PCI>
//	AT+QNWLOCK="common/5g",<PCI>,<ARFCN>,<SCS_kHz>,<band>
//
// Note the 5G form differs from 4G: no enable flag, PCI before freq, SCS in
// actual kHz (not the index QENG reports), and band number required.
func (m *Modem) LockCurrentCell(ctx context.Context, tech string) error {
	tech = strings.ToLower(tech)
	if tech != "4g" && tech != "5g" {
		return fmt.Errorf("unsupported tech %q (expected 4g or 5g)", tech)
	}

	resp, err := m.at.SendCMD(ctx, `AT+QENG="servingcell"`, 3*time.Second)
	if err != nil {
		return err
	}
	if resp.Status != atserver.ATStatusOK {
		return modem.ErrFieldMissing
	}

	wantTech := map[string]bool{}
	if tech == "4g" {
		wantTech["LTE"] = true
	} else {
		wantTech["NR5G-SA"] = true
		wantTech["NR5G-NSA"] = true
	}

	for _, line := range modem.FindLines(resp.Response, `+QENG: "servingcell"`) {
		_, val, _ := strings.Cut(line, ":")
		f := modem.SplitCSV(val)
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		if len(f) < 10 {
			continue
		}
		t := modem.ParseQuoted(f[2])
		if !wantTech[t] {
			continue
		}
		pci, ok := modem.ParseInt(f[7])
		if !ok {
			return modem.ErrFieldMissing
		}

		var cmd string
		switch t {
		case "LTE":
			// f[8] = EARFCN
			earfcn, ok := modem.ParseInt(f[8])
			if !ok {
				return modem.ErrFieldMissing
			}
			cmd = fmt.Sprintf(`AT+QNWLOCK="common/4g",1,%d,%d`, earfcn, pci)
		case "NR5G-SA", "NR5G-NSA":
			// QENG NR5G-SA layout: f[9]=ARFCN, f[10]=band, f[15]=SCS (index).
			arfcn, ok := modem.ParseInt(f[9])
			if !ok {
				return modem.ErrFieldMissing
			}
			band, ok := modem.ParseInt(f[10])
			if !ok {
				return modem.ErrFieldMissing
			}
			scsIdx := 0
			if len(f) > 15 {
				if v, ok := modem.ParseInt(f[15]); ok {
					scsIdx = v
				}
			}
			scsKHz := scsIndexToKHz(scsIdx)
			cmd = fmt.Sprintf(`AT+QNWLOCK="common/5g",%d,%d,%d,%d`, pci, arfcn, scsKHz, band)
		}

		lockResp, err := m.at.SendCMD(ctx, cmd, 5*time.Second)
		if err != nil {
			return err
		}
		if lockResp.Status != atserver.ATStatusOK {
			return fmt.Errorf("AT command rejected: %s", cmd)
		}
		return nil
	}
	return fmt.Errorf("no serving cell found for %s", tech)
}

// LockCell pins the modem to an explicit cell target. scsIdx is the QENG/
// QSCAN subcarrier-spacing index (0..4); ignored for 4G. band is required
// for 5G — the modem rejects 5G lock without it.
func (m *Modem) LockCell(ctx context.Context, tech string, freq, pci, scsIdx, band int) error {
	tech = strings.ToLower(tech)
	if freq <= 0 || pci < 0 {
		return fmt.Errorf("invalid target freq=%d pci=%d", freq, pci)
	}
	var cmd string
	switch tech {
	case "4g":
		cmd = fmt.Sprintf(`AT+QNWLOCK="common/4g",1,%d,%d`, freq, pci)
	case "5g":
		if band <= 0 {
			return fmt.Errorf("5G lock requires a band number")
		}
		cmd = fmt.Sprintf(`AT+QNWLOCK="common/5g",%d,%d,%d,%d`, pci, freq, scsIndexToKHz(scsIdx), band)
	default:
		return fmt.Errorf("unsupported tech %q (expected 4g or 5g)", tech)
	}
	resp, err := m.at.SendCMD(ctx, cmd, 5*time.Second)
	if err != nil {
		return err
	}
	if resp.Status != atserver.ATStatusOK {
		return fmt.Errorf("AT command rejected: %s", cmd)
	}
	return nil
}
