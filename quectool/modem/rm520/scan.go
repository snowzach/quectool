package rm520

import (
	"context"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

const scanTimeout = 90 * time.Second

func (m *Modem) Scan(ctx context.Context) ([]modem.ScanResult, error) {
	resp, err := m.at.SendCMD(ctx, "AT+COPS=?", scanTimeout)
	if err != nil {
		return nil, err
	}
	if resp.Status != atserver.ATStatusOK {
		return nil, modem.ErrFieldMissing
	}
	return parseScan(resp.Response)
}

func parseScan(lines []string) ([]modem.ScanResult, error) {
	line, ok := modem.FindLine(lines, "+COPS:")
	if !ok {
		return nil, modem.ErrFieldMissing
	}
	_, val, _ := strings.Cut(line, ":")
	val = strings.TrimSpace(val)

	var results []modem.ScanResult
	depth := 0
	start := -1
	for i, c := range val {
		switch c {
		case '(':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')':
			depth--
			if depth == 0 && start >= 0 {
				inner := val[start:i]
				if r, ok := parseScanEntry(inner); ok {
					results = append(results, r)
				}
				start = -1
			}
		}
	}
	return results, nil
}

func parseScanEntry(s string) (modem.ScanResult, bool) {
	fields := modem.SplitCSV(s)
	if len(fields) < 4 {
		return modem.ScanResult{}, false
	}
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	state := fields[0]
	stateStr := map[string]string{"0": "unknown", "1": "available", "2": "current", "3": "forbidden"}[state]
	if stateStr == "" {
		stateStr = "unknown"
	}
	tech := ""
	if len(fields) >= 5 {
		tech = map[string]string{
			"0": "GSM", "2": "UMTS", "3": "GSM-Compact",
			"7": "LTE", "12": "NR5G",
		}[fields[4]]
	}
	mccmnc := modem.ParseQuoted(fields[3])
	mcc, mnc := "", ""
	if len(mccmnc) >= 5 {
		mcc = mccmnc[:3]
		mnc = mccmnc[3:]
	}
	return modem.ScanResult{
		Operator: modem.ParseQuoted(fields[1]),
		MCC:      mcc,
		MNC:      mnc,
		Tech:     tech,
		State:    stateStr,
	}, true
}
