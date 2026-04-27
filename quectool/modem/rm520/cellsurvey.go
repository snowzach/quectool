package rm520

import (
	"context"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

const cellSurveyTimeout = 120 * time.Second

// CellSurvey performs AT+QSCAN=3,1 — full RF sweep across GSM/LTE/NR5G.
// Each visible cell becomes one CellSurveyResult.
func (m *Modem) CellSurvey(ctx context.Context) ([]modem.CellSurveyResult, error) {
	resp, err := m.at.SendCMD(ctx, "AT+QSCAN=3,1", cellSurveyTimeout)
	if err != nil {
		return nil, err
	}
	if resp.Status != atserver.ATStatusOK {
		return nil, modem.ErrFieldMissing
	}
	return parseCellSurvey(resp.Response), nil
}

// parseCellSurvey turns +QSCAN lines into CellSurveyResult entries. Format
// (verified empirically on RM520N-GL A04 firmware):
//
//	+QSCAN: "LTE",MCC,MNC,EARFCN,PCI,RSRP,RSRQ,RSSI,srxlev,cellID,TAC,bandwidth,band
//	+QSCAN: "NR5G",MCC,MNC,ARFCN,PCI,RSRP,RSRQ,SINR,scs,cellID,TAC,srxlev,band,...
//
// Band lives at index 12 in both layouts. NR5G includes SCS at index 8 — we
// capture it because the cell-lock command rejects writes with the wrong
// SCS, so locking from a survey result requires we pass through the right one.
// Lines we can't parse are skipped silently — partial results beat failure.
func parseCellSurvey(lines []string) []modem.CellSurveyResult {
	var out []modem.CellSurveyResult
	for _, line := range modem.FindLines(lines, "+QSCAN:") {
		_, val, _ := strings.Cut(line, ":")
		f := modem.SplitCSV(val)
		if len(f) < 13 {
			continue
		}
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		c := modem.CellSurveyResult{
			Tech:   modem.ParseQuoted(f[0]),
			MCC:    f[1],
			MNC:    f[2],
			CellID: f[9],
		}
		if v, ok := modem.ParseInt(f[3]); ok {
			c.Freq = v
		}
		if v, ok := modem.ParseInt(f[4]); ok {
			c.PCI = v
		}
		if v, ok := modem.ParseInt(f[5]); ok {
			c.RSRP = v
		}
		if v, ok := modem.ParseInt(f[6]); ok {
			c.RSRQ = v
		}
		if v, ok := modem.ParseInt(f[12]); ok {
			c.Band = v
		}
		// SCS only meaningful for NR5G. 0 is a valid value (15 kHz), so we
		// always parse — caller looks at Tech to decide whether to use it.
		if c.Tech == "NR5G" {
			if v, ok := modem.ParseInt(f[8]); ok {
				c.SCS = v
			}
		}
		out = append(out, c)
	}
	return out
}
