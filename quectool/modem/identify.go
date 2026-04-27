package modem

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
)

const identTimeout = 3 * time.Second

func identify(ctx context.Context, at atserver.ATServer) (Info, error) {
	// Modems flood URCs (RDY, +CPIN, +QIND, +CFUN, +QUSIM, ...) right after
	// boot. A throwaway AT lets the server drain that queue so subsequent
	// ident commands get a clean response.
	_, _ = at.SendCMD(ctx, "AT", identTimeout)

	one := func(cmd string) (string, error) {
		resp, err := at.SendCMD(ctx, cmd, identTimeout)
		if err != nil {
			return "", fmt.Errorf("%s: %w", cmd, err)
		}
		if resp.Status != atserver.ATStatusOK {
			return "", fmt.Errorf("%s: bad response", cmd)
		}
		for _, line := range resp.Response {
			if v := pickIdentLine(line); v != "" {
				return v, nil
			}
		}
		return "", fmt.Errorf("%s: no usable line", cmd)
	}
	manuf, err := one("AT+CGMI")
	if err != nil {
		return Info{}, err
	}
	model, err := one("AT+CGMM")
	if err != nil {
		return Info{}, err
	}
	firmware, err := one("AT+CGMR")
	if err != nil {
		return Info{}, err
	}
	imei, err := one("AT+CGSN")
	if err != nil {
		return Info{}, err
	}
	return Info{
		Manufacturer: manuf,
		Model:        model,
		Firmware:     firmware,
		IMEI:         imei,
	}, nil
}

// pickIdentLine returns the trimmed line if it looks like a real identity
// response, or "" if it's a URC, status keyword, or empty. Identity replies
// (CGMI/CGMM/CGMR/CGSN) are bare strings — never prefixed with `+`.
func pickIdentLine(line string) string {
	v := strings.TrimSpace(line)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "+") {
		return ""
	}
	switch v {
	case "RDY", "OK", "ERROR", "BUSY", "NO CARRIER", "CONNECT":
		return ""
	}
	return v
}

// Detect identifies the connected modem and constructs the matching impl.
// Falls back to a generic impl if no registered constructor matches.
func Detect(ctx context.Context, at atserver.ATServer) (Modem, error) {
	info, err := identify(ctx, at)
	if err != nil {
		return nil, fmt.Errorf("modem identify: %w", err)
	}
	if c, ok := lookup(info.Model); ok {
		return c(at, info)
	}
	gc, ok := lookup("__generic__")
	if !ok {
		return nil, fmt.Errorf("no constructor for model %q and no generic fallback", info.Model)
	}
	return gc(at, info)
}
