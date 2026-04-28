package rm520

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

func (m *Modem) ListSMS(ctx context.Context) ([]modem.SMS, error) {
	// Switch to text mode first (idempotent).
	if _, err := m.at.SendCMD(ctx, "AT+CMGF=1", 2*time.Second); err != nil {
		return nil, err
	}
	resp, err := m.at.SendCMD(ctx, `AT+CMGL="ALL"`, 10*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.Status != atserver.ATStatusOK {
		return nil, modem.ErrFieldMissing
	}
	return parseSMSList(resp.Response)
}

func parseSMSList(lines []string) ([]modem.SMS, error) {
	var out []modem.SMS
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if !strings.HasPrefix(strings.TrimLeft(l, " \t"), "+CMGL:") {
			continue
		}
		_, val, _ := strings.Cut(l, ":")
		f := modem.SplitCSV(val)
		for j := range f {
			f[j] = strings.TrimSpace(f[j])
		}
		if len(f) < 5 {
			continue
		}
		s := modem.SMS{}
		if n, ok := modem.ParseInt(f[0]); ok {
			s.Index = n
		}
		stat := modem.ParseQuoted(f[1])
		s.Read = strings.HasPrefix(stat, "REC READ") || strings.HasPrefix(stat, "STO SENT")
		s.From = modem.ParseQuoted(f[2])
		s.Time = modem.ParseQuoted(f[4])
		// Body is the next non-+CMGL line(s) until the next +CMGL or end.
		var bodyParts []string
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(strings.TrimLeft(lines[j], " \t"), "+CMGL:") {
				break
			}
			bodyParts = append(bodyParts, lines[j])
			i = j
		}
		s.Body = strings.TrimSpace(strings.Join(bodyParts, "\n"))
		out = append(out, s)
	}
	return out, nil
}

// normalizeMSISDN strips spaces, dashes, parens, dots — anything not a digit
// or a leading +. AT+CMGS rejects formatted numbers like "+1-614-446-8968".
func normalizeMSISDN(s string) string {
	s = strings.TrimSpace(s)
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			out = append(out, c)
		} else if c == '+' && len(out) == 0 {
			out = append(out, c)
		}
	}
	return string(out)
}

func (m *Modem) SendSMS(ctx context.Context, to, body string) error {
	to = normalizeMSISDN(to)
	if to == "" {
		return fmt.Errorf("empty destination number")
	}
	if _, err := m.at.SendCMD(ctx, "AT+CMGF=1", 2*time.Second); err != nil {
		return err
	}
	// SendMessage handles the AT+CMGS prompt-then-body sequence properly:
	// stage 1 sends the command and waits for `>`, stage 2 writes the body
	// terminated by Ctrl-Z. Trying to cram both into one SendCMD wedges the
	// modem in prompt-input mode when timing slips.
	cmd := fmt.Sprintf(`AT+CMGS="%s"`, to)
	resp, err := m.at.SendMessage(ctx, cmd, body, 30*time.Second)
	if err != nil {
		return err
	}
	if resp.Status != atserver.ATStatusOK {
		return fmt.Errorf("send failed")
	}
	return nil
}

func (m *Modem) DeleteSMS(ctx context.Context, index int) error {
	resp, err := m.at.SendCMD(ctx, fmt.Sprintf("AT+CMGD=%d", index), 5*time.Second)
	if err != nil {
		return err
	}
	if resp.Status != atserver.ATStatusOK {
		return fmt.Errorf("delete failed")
	}
	return nil
}

// DeleteSMSBulk uses Quectel's AT+CMGD=1,<flag> bulk form. flag=1 removes
// only messages with REC READ status; flag=4 removes everything.
func (m *Modem) DeleteSMSBulk(ctx context.Context, scope string) error {
	var flag int
	switch scope {
	case "read":
		flag = 1
	case "all":
		flag = 4
	default:
		return fmt.Errorf("invalid scope %q (want read or all)", scope)
	}
	resp, err := m.at.SendCMD(ctx, fmt.Sprintf("AT+CMGD=1,%d", flag), 10*time.Second)
	if err != nil {
		return err
	}
	if resp.Status != atserver.ATStatusOK {
		return fmt.Errorf("bulk delete failed")
	}
	return nil
}

// MarkAllRead reads each unread message via AT+CMGR, which has the side
// effect of transitioning REC UNREAD → REC READ. There's no bulk form.
func (m *Modem) MarkAllRead(ctx context.Context) error {
	if _, err := m.at.SendCMD(ctx, "AT+CMGF=1", 2*time.Second); err != nil {
		return err
	}
	resp, err := m.at.SendCMD(ctx, `AT+CMGL="REC UNREAD"`, 10*time.Second)
	if err != nil {
		return err
	}
	if resp.Status != atserver.ATStatusOK {
		return fmt.Errorf("list unread failed")
	}
	for _, line := range modem.FindLines(resp.Response, "+CMGL:") {
		_, val, _ := strings.Cut(line, ":")
		f := modem.SplitCSV(val)
		if len(f) == 0 {
			continue
		}
		idx, ok := modem.ParseInt(strings.TrimSpace(f[0]))
		if !ok {
			continue
		}
		// Read each message individually; ignore per-message failures so
		// one bad index doesn't strand the rest still unread.
		_, _ = m.at.SendCMD(ctx, fmt.Sprintf("AT+CMGR=%d", idx), 5*time.Second)
	}
	return nil
}
