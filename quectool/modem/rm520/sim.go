package rm520

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

// simInputs aggregates raw lines from the AT calls used by GetSim. Split
// from the handler for testability.
//
// Registration is queried across all three +xREG variants because the
// "right" one depends on what tech the modem is on:
//   - +CGREG  : GPRS/2G/3G packet domain
//   - +CEREG  : EPS / LTE
//   - +C5GREG : 5G-SA
// The handler issues all three and considers the modem registered if any
// reports stat 1 (home) or 5 (roaming).
type simInputs struct {
	slot      []string
	imsi      []string
	iccid     []string
	cops      []string
	cgcontrdp []string
	cgdcont   []string
	cgreg     []string
	cereg     []string
	c5greg    []string
}

func (m *Modem) GetSim(ctx context.Context) (*modem.SimInfo, error) {
	send := func(cmd string) ([]string, error) {
		resp, err := m.at.SendCMD(ctx, cmd, 3*time.Second)
		if err != nil {
			return nil, err
		}
		if resp.Status == atserver.ATStatusError {
			return nil, nil // best-effort: caller treats absent fields as missing
		}
		return resp.Response, nil
	}
	in := simInputs{}
	var err error
	if in.slot, err = send("AT+QUIMSLOT?"); err != nil {
		return nil, err
	}
	if in.imsi, err = send("AT+CIMI"); err != nil {
		return nil, err
	}
	if in.iccid, err = send(`AT+QCCID`); err != nil {
		return nil, err
	}
	if in.cops, err = send(`AT+COPS?`); err != nil {
		return nil, err
	}
	if in.cgcontrdp, err = send(`AT+CGCONTRDP=1`); err != nil {
		return nil, err
	}
	if in.cgdcont, err = send(`AT+CGDCONT?`); err != nil {
		return nil, err
	}
	if in.cgreg, err = send(`AT+CGREG?`); err != nil {
		return nil, err
	}
	if in.cereg, err = send(`AT+CEREG?`); err != nil {
		return nil, err
	}
	if in.c5greg, err = send(`AT+C5GREG?`); err != nil {
		return nil, err
	}
	return parseSim(in)
}

func parseSim(in simInputs) (*modem.SimInfo, error) {
	out := &modem.SimInfo{}

	// Slot
	if line, ok := modem.FindLine(in.slot, "+QUIMSLOT:"); ok {
		_, val, _ := strings.Cut(line, ":")
		if n, ok := modem.ParseInt(strings.TrimSpace(val)); ok {
			out.Slot = n
		}
	}
	// IMSI is a bare numeric line on its own.
	for _, l := range in.imsi {
		if t := strings.TrimSpace(l); t != "" && t[0] >= '0' && t[0] <= '9' {
			out.IMSI = t
			break
		}
	}
	// ICCID
	if line, ok := modem.FindLine(in.iccid, "+QCCID:"); ok {
		_, val, _ := strings.Cut(line, ":")
		out.ICCID = strings.TrimSpace(val)
	}
	// Operator from +COPS: <mode>,<format>,"<oper>",<act>
	if line, ok := modem.FindLine(in.cops, "+COPS:"); ok {
		_, val, _ := strings.Cut(line, ":")
		fields := modem.SplitCSV(val)
		if len(fields) >= 3 {
			out.Operator = modem.ParseQuoted(strings.TrimSpace(fields[2]))
		}
	}
	// CGCONTRDP carries the assigned PDP context — APN, IP/mask, gateway,
	// DNS. In dual-stack (IPv4v6) the modem emits two lines per cid, one
	// per family; we walk all of them and merge into the single output.
	// Format:
	//
	//	+CGCONTRDP: <cid>,<bid>,"<apn>","<ip+mask>","<gw>","<dns1>","<dns2>"[,...,<MTU>]
	for _, line := range modem.FindLines(in.cgcontrdp, "+CGCONTRDP:") {
		_, val, _ := strings.Cut(line, ":")
		fields := modem.SplitCSV(val)
		if len(fields) >= 3 && out.APN == "" {
			out.APN = modem.ParseQuoted(strings.TrimSpace(fields[2]))
		}
		if len(fields) >= 4 {
			ipv4, ipv6 := parseCGContrdpAddr(modem.ParseQuoted(strings.TrimSpace(fields[3])))
			if ipv4 != "" {
				out.IPv4 = ipv4
			}
			if ipv6 != "" {
				out.IPv6 = ipv6
			}
		}
		if len(fields) >= 5 {
			gw4, gw6 := parseCGContrdpAddr(modem.ParseQuoted(strings.TrimSpace(fields[4])))
			if gw := pickNonEmpty(gw4, gw6); gw != "" && out.Gateway == "" {
				out.Gateway = gw
			}
		}
		for i := 5; i < len(fields) && i <= 6; i++ {
			d4, d6 := parseCGContrdpAddr(modem.ParseQuoted(strings.TrimSpace(fields[i])))
			if d := pickNonEmpty(d4, d6); d != "" {
				out.DNS = append(out.DNS, d)
			}
		}
	}
	// APN IP type from +CGDCONT: 1,"<type>","<apn>",...
	if line, ok := modem.FindLine(in.cgdcont, "+CGDCONT: 1,"); ok {
		_, val, _ := strings.Cut(line, ":")
		fields := modem.SplitCSV(val)
		if len(fields) >= 2 {
			out.APNIP = modem.ParseQuoted(strings.TrimSpace(fields[1]))
		}
	}
	// Registration: any of +CGREG / +CEREG / +C5GREG with stat 1 (home) or
	// 5 (roaming) means the modem is registered. Different tech generations
	// advertise registration in different commands; we accept any match.
	for _, lines := range [][]string{in.cgreg, in.cereg, in.c5greg} {
		for _, prefix := range []string{"+CGREG:", "+CEREG:", "+C5GREG:"} {
			line, ok := modem.FindLine(lines, prefix)
			if !ok {
				continue
			}
			_, val, _ := strings.Cut(line, ":")
			fields := modem.SplitCSV(val)
			if len(fields) >= 2 {
				if n, ok := modem.ParseInt(fields[1]); ok && (n == 1 || n == 5) {
					out.Registered = true
				}
			}
		}
	}

	if out.IMSI == "" && out.Operator == "" && out.APN == "" {
		return nil, modem.ErrFieldMissing
	}
	return out, nil
}

// parseCGContrdpAddr decodes the dotted-octet form Quectel uses for IPs in
// CGCONTRDP responses. Octets are decimal-encoded bytes:
//
//	4 octets   = bare IPv4
//	8 octets   = IPv4 + subnet mask (we discard the mask)
//	16 octets  = bare IPv6
//	32 octets  = IPv6 + prefix mask (we discard the mask)
//
// Returns ipv4, ipv6 — exactly one is set on success, both empty on
// unrecognized input. The empty input "0.0.0.0.0.0.0.0..." (modem reports
// the placeholder when no address has been negotiated yet) returns empty.
func parseCGContrdpAddr(s string) (ipv4, ipv6 string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	parts := strings.Split(s, ".")
	nums := make([]byte, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 0 || n > 255 {
			return "", ""
		}
		nums = append(nums, byte(n))
	}
	allZero := true
	for _, b := range nums {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return "", ""
	}
	switch len(nums) {
	case 4, 8:
		return fmt.Sprintf("%d.%d.%d.%d", nums[0], nums[1], nums[2], nums[3]), ""
	case 16, 32:
		return "", net.IP(nums[:16]).String()
	}
	return "", ""
}

func pickNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
