package rm520

import (
	"context"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

func (m *Modem) GetCell(ctx context.Context) (*modem.CellInfo, error) {
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
	serving, err := send(`AT+QENG="servingcell"`)
	if err != nil {
		return nil, err
	}
	neighbours, err := send(`AT+QENG="neighbourcell"`)
	if err != nil {
		return nil, err
	}
	// QCAINFO is best-effort — modems without CA reply ERROR. Errors are
	// already swallowed inside send (returns nil, nil on non-OK status).
	cainfo, err := send(`AT+QCAINFO`)
	if err != nil {
		return nil, err
	}
	return parseCell(serving, neighbours, cainfo)
}

func parseCell(serving, neighbours, cainfo []string) (*modem.CellInfo, error) {
	servingLines := modem.FindLines(serving, `+QENG: "servingcell"`)
	if len(servingLines) == 0 {
		return nil, modem.ErrFieldMissing
	}
	out, err := parseServingLine(servingLines[0])
	if err != nil {
		return nil, err
	}

	// Additional servingcell lines (NR5G-NSA pairs the LTE anchor with an
	// NR5G secondary) become SCells.
	for _, line := range servingLines[1:] {
		if c, err := parseServingLine(line); err == nil && c != nil {
			out.SCells = append(out.SCells, *c)
		}
	}

	// Carrier-aggregation SCells reported by AT+QCAINFO.
	out.SCells = append(out.SCells, parseQCAInfo(cainfo)...)

	for _, nl := range modem.FindLines(neighbours, `+QENG: "neighbourcell`) {
		_, nval, _ := strings.Cut(nl, ":")
		nf := modem.SplitCSV(nval)
		for i := range nf {
			nf[i] = strings.TrimSpace(nf[i])
		}
		if len(nf) < 4 {
			continue
		}
		nb := modem.CellInfo{Tech: modem.ParseQuoted(nf[1])}
		if pci, ok := modem.ParseInt(nf[3]); ok {
			nb.PCI = pci
		}
		out.Neighbors = append(out.Neighbors, nb)
	}
	return out, nil
}

// parseServingLine parses a single +QENG: "servingcell" line into a CellInfo.
// Field positions are tech-conditional per the RM520N-GL AT command spec.
// Short responses like +QENG: "servingcell","SEARCH" return a CellInfo with
// only State populated — the modem isn't on a cell yet, but that's not an
// error worth surfacing.
func parseServingLine(line string) (*modem.CellInfo, error) {
	_, val, _ := strings.Cut(line, ":")
	f := modem.SplitCSV(val)
	for i := range f {
		f[i] = strings.TrimSpace(f[i])
	}
	out := &modem.CellInfo{}
	if len(f) >= 2 {
		out.State = modem.ParseQuoted(f[1])
	}
	if len(f) < 8 {
		return out, nil
	}
	out.Tech = modem.ParseQuoted(f[2])
	out.MCC = f[4]
	out.MNC = f[5]
	out.CellID = f[6]
	if pci, ok := modem.ParseInt(f[7]); ok {
		out.PCI = pci
	}
	switch out.Tech {
	case "LTE":
		if len(f) > 11 {
			out.Band = f[9]
			out.Bandwidth = f[11]
		}
	case "NR5G-SA", "NR5G-NSA":
		if len(f) > 11 {
			out.Band = f[10]
			out.Bandwidth = f[11]
		}
	}
	if sig, err := parseSignalLine(line); err == nil {
		out.Signal = *sig
	}
	return out, nil
}

// parseQCAInfo extracts secondary carrier cells from an AT+QCAINFO response.
// Format: +QCAINFO: "<type>",<EARFCN>,<bandwidth>,"<band>",<state>[,<PCI>,<RSRP>,<RSRQ>]
// The PCC entry is the primary carrier (already represented by the +QENG
// servingcell), so we skip it; SCC entries become SCells.
func parseQCAInfo(lines []string) []modem.CellInfo {
	var out []modem.CellInfo
	for _, line := range modem.FindLines(lines, `+QCAINFO:`) {
		_, val, _ := strings.Cut(line, ":")
		f := modem.SplitCSV(val)
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		if len(f) < 5 {
			continue
		}
		kind := modem.ParseQuoted(f[0])
		if kind == "PCC" {
			continue
		}
		band := modem.ParseQuoted(f[3])
		// QCAINFO bands are reported like "LTE BAND 2" or "NR5G BAND n41".
		// Pick a reasonable Tech and trim the band to the suffix.
		tech := "LTE"
		if strings.HasPrefix(band, "NR5G") {
			tech = "NR5G"
		}
		if i := strings.LastIndex(band, " "); i > 0 {
			band = band[i+1:]
		}
		c := modem.CellInfo{
			Tech:      tech,
			Band:      band,
			Bandwidth: f[2],
		}
		if len(f) > 5 {
			if pci, ok := modem.ParseInt(f[5]); ok {
				c.PCI = pci
			}
		}
		if len(f) > 7 {
			if rsrp, ok := modem.ParseInt(f[6]); ok {
				c.Signal.RSRP = rsrp
			}
			if rsrq, ok := modem.ParseInt(f[7]); ok {
				c.Signal.RSRQ = rsrq
			}
			c.Signal.Tech = tech
		}
		out = append(out, c)
	}
	return out
}
