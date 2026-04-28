package rm520

import (
	"context"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

func (m *Modem) GetSignal(ctx context.Context) (*modem.Signal, error) {
	resp, err := m.at.SendCMD(ctx, `AT+QENG="servingcell"`, 3*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.Status != atserver.ATStatusOK {
		return nil, modem.ErrFieldMissing
	}
	return parseSignal(resp.Response)
}

// parseSignal reads RSRP/RSRQ/SINR/Tech from a +QENG: "servingcell",...
// response. Field positions are tech-specific per the Quectel RM520N-GL
// AT command spec; the layouts differ enough between LTE and NR5G that
// tech-conditional indexing is more reliable than counting from the end.
//
// LTE layout:
//   +QENG: "servingcell",<state>,"LTE",<is_tdd>,<MCC>,<MNC>,<cellID>,
//          <PCID>,<earfcn>,<freq_band>,<UL_bw>,<DL_bw>,<TAC>,
//          <RSRP>,<RSRQ>,<RSSI>,<SINR>,<srxlev>
//
// NR5G-SA / NR5G-NSA NR5G-line layout:
//   +QENG: "servingcell",<state>,"NR5G-SA",<duplex>,<MCC>,<MNC>,
//          <cellID>,<PCID>,<TAC>,<ARFCN>,<band>,<DL_bw>,
//          <RSRP>,<RSRQ>,<SINR>,<scs>,<srxlev>
func parseSignal(lines []string) (*modem.Signal, error) {
	line, ok := modem.FindLine(lines, `+QENG: "servingcell"`)
	if !ok {
		return nil, modem.ErrFieldMissing
	}
	return parseSignalLine(line)
}

// parseSignalLine extracts a Signal from a single +QENG: "servingcell" line.
// Useful for iterating multiple servingcell entries (NR5G-NSA reports two).
func parseSignalLine(line string) (*modem.Signal, error) {
	_, val, _ := strings.Cut(line, ":")
	fields := modem.SplitCSV(val)
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	out := &modem.Signal{}
	// f[1] holds the state — present in every QENG response, including
	// short ones like +QENG: "servingcell","SEARCH" where there's no tech
	// or signal data to follow.
	if len(fields) >= 2 {
		out.State = modem.ParseQuoted(fields[1])
	}
	if len(fields) < 5 {
		return out, nil
	}
	out.Tech = modem.ParseQuoted(fields[2])

	var rsrpIdx, rsrqIdx, sinrIdx int
	switch out.Tech {
	case "LTE":
		rsrpIdx, rsrqIdx, sinrIdx = 13, 14, 16
	case "NR5G-SA", "NR5G-NSA":
		rsrpIdx, rsrqIdx, sinrIdx = 12, 13, 14
	default:
		// Unknown tech: return what we have (Tech only) without panicking.
		return out, nil
	}
	if len(fields) > rsrpIdx {
		if v, ok := modem.ParseInt(fields[rsrpIdx]); ok {
			out.RSRP = v
		}
	}
	if len(fields) > rsrqIdx {
		if v, ok := modem.ParseInt(fields[rsrqIdx]); ok {
			out.RSRQ = v
		}
	}
	if len(fields) > sinrIdx {
		if v, ok := modem.ParseInt(fields[sinrIdx]); ok {
			out.SINR = float64(v)
		}
	}
	return out, nil
}
