// Package rm520 implements modem.Modem for the Quectel RM520N-GL.
//
// Methods are split across files by concern: sim.go, signal.go, cell.go,
// settings.go, scan.go, sms.go. Each file contains its handler plus a
// pure parser exposed for test. This file holds the skeleton: type,
// constructor, registration, and the trivial Info/Capabilities accessors.
package rm520

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

func init() {
	modem.Register("RM520N-GL", New)
}

type Modem struct {
	at                 atserver.ATServer
	info               modem.Info
	supportedLTEBands  []int
	supportedNR5GBands []int
}

// Datasheet fallback if the modem refuses AT+QNWPREFCFG="ue_capability_band"
// at startup. Bands 7 and 38 listed here per Quectel rev 1.4, but the chip's
// own report is authoritative when available.
var (
	fallbackLTEBands  = []int{1, 2, 3, 4, 5, 7, 8, 12, 13, 14, 17, 18, 19, 20, 25, 26, 28, 29, 30, 32, 34, 38, 39, 40, 41, 42, 43, 46, 48, 66, 71}
	fallbackNR5GBands = []int{1, 2, 3, 5, 7, 8, 12, 20, 25, 28, 38, 40, 41, 48, 66, 71, 75, 76, 77, 78, 79}
)

func New(at atserver.ATServer, info modem.Info) (modem.Modem, error) {
	info.Capabilities = []modem.Capability{
		modem.CapBandLockLTE,
		modem.CapBandLock5G,
		modem.CapNetworkScan,
		modem.CapSMS,
		modem.CapNR5GSA,
	}
	lte, nr5g := queryCapabilityBands(at)
	if lte == nil {
		lte = fallbackLTEBands
	}
	if nr5g == nil {
		nr5g = fallbackNR5GBands
	}
	info.AvailableBands = modem.BandSet{LTE: lte, NR5G: nr5g}
	info.USBProtocol = queryUSBProtocol(at)
	info.DataPath = queryDataPath(at)
	return &Modem{
		at:                 at,
		info:               info,
		supportedLTEBands:  lte,
		supportedNR5GBands: nr5g,
	}, nil
}

// queryCapabilityBands asks the modem what bands the SIM/carrier policy
// actually permits. Returns (lte, nr5g) as nil on any failure so the caller
// can fall back to a hardcoded list.
//
// We use policy_band, NOT ue_capability_band: on Quectel R03 firmware
// ue_capability_band mirrors the *current* nr5g_band/lte_band preferences,
// so as soon as the user restricts bands the picker would shrink to those.
// policy_band is stable and reflects what the modem can actually attempt.
func queryCapabilityBands(at atserver.ATServer) (lte, nr5g []int) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := at.SendCMD(ctx, `AT+QNWPREFCFG="policy_band"`, 3*time.Second)
	if err != nil || resp.Status != atserver.ATStatusOK {
		return nil, nil
	}
	for _, line := range resp.Response {
		switch {
		case strings.Contains(line, `"lte_band"`):
			lte = parseBandList(line)
		case strings.Contains(line, `"nr5g_band"`) && !strings.Contains(line, `"nsa_nr5g_band"`):
			nr5g = parseBandList(line)
		}
	}
	return lte, nr5g
}

// queryUSBProtocol reads AT+QCFG="usbnet" and maps the integer to a label.
// Returns "" on failure. Note: this is just the USB-wire format — irrelevant
// if the actual data path is PCIe (see queryDataPath).
//
//	+QCFG: "usbnet",0     QMI (default, qmi_wwan)
//	+QCFG: "usbnet",1     ECM
//	+QCFG: "usbnet",2     MBIM
//	+QCFG: "usbnet",3     RNDIS
//	+QCFG: "usbnet",5     NCM
func queryUSBProtocol(at atserver.ATServer) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := at.SendCMD(ctx, `AT+QCFG="usbnet"`, 3*time.Second)
	if err != nil || resp.Status != atserver.ATStatusOK {
		return ""
	}
	for _, line := range resp.Response {
		if !strings.Contains(line, `"usbnet"`) {
			continue
		}
		idx := strings.LastIndex(line, ",")
		if idx < 0 {
			return ""
		}
		v := strings.TrimSpace(line[idx+1:])
		switch v {
		case "0":
			return "QMI"
		case "1":
			return "ECM"
		case "2":
			return "MBIM"
		case "3":
			return "RNDIS"
		case "5":
			return "NCM"
		}
	}
	return ""
}

// queryDataPath reads AT+QCFG="data_interface" and reports which physical
// interface carries the data plane. Format: `+QCFG: "data_interface",<primary>,<aux>`
// where 0=USB, 1=PCIe. We report the primary.
func queryDataPath(at atserver.ATServer) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := at.SendCMD(ctx, `AT+QCFG="data_interface"`, 3*time.Second)
	if err != nil || resp.Status != atserver.ATStatusOK {
		return ""
	}
	for _, line := range resp.Response {
		if !strings.Contains(line, `"data_interface"`) {
			continue
		}
		// Split off the prefix, then the comma-separated values after.
		_, val, _ := strings.Cut(line, ":")
		fields := strings.Split(val, ",")
		if len(fields) < 2 {
			return ""
		}
		switch strings.TrimSpace(fields[1]) {
		case "0":
			return "USB"
		case "1":
			return "PCIe"
		}
	}
	return ""
}

// parseBandList extracts the trailing colon-separated decimal list from a
// `+QNWPREFCFG: "<key>",<bands>` line. The list may be wrapped in quotes
// on some firmware revisions.
func parseBandList(line string) []int {
	idx := strings.LastIndex(line, ",")
	if idx < 0 {
		return nil
	}
	v := strings.TrimSpace(line[idx+1:])
	v = strings.Trim(v, `"`)
	parts := strings.Split(v, ":")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (m *Modem) Info() modem.Info                 { return m.info }
func (m *Modem) Capabilities() []modem.Capability { return m.info.Capabilities }
