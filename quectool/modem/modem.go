// Package modem provides a backend abstraction over a cellular modem.
// Implementations live in subpackages (e.g. rm520, generic) and register
// themselves with the package-level registry via init().
//
// At startup, cmd/server.go calls Detect(), which runs identification
// AT commands and constructs the impl whose model name matches.
package modem

import (
	"context"
	"errors"
	"sync"

	"github.com/snowzach/quectool/quectool/atserver"
)

var (
	// ErrUnsupported is returned by methods that this modem doesn't support.
	// Callers should also consult Capabilities() to gate UI declaratively.
	ErrUnsupported = errors.New("operation not supported by this modem")

	// ErrFieldMissing is returned when an expected line is absent from an
	// AT response. Parsers must never panic on missing fields.
	ErrFieldMissing = errors.New("expected field missing in AT response")
)

// Capability advertises operations a modem supports. The frontend gates UI
// off Info.Capabilities; the server still enforces support via 405 responses.
type Capability string

const (
	CapBandLockLTE Capability = "band_lock_lte"
	CapBandLock5G  Capability = "band_lock_5g"
	CapNetworkScan Capability = "network_scan"
	CapSMS         Capability = "sms"
	CapNR5GSA      Capability = "nr5g_sa"
)

// Info identifies the modem and is cached at construction.
type Info struct {
	Manufacturer   string       `json:"manufacturer"`
	Model          string       `json:"model"`
	Firmware       string       `json:"firmware"`
	IMEI           string       `json:"imei"`
	Capabilities   []Capability `json:"capabilities"`
	AvailableBands BandSet      `json:"available_bands"`
	// USBProtocol is the wire format used WHEN data flows over USB:
	// "QMI", "ECM", "MBIM", "RNDIS", "NCM", or "" if unknown. Independent
	// of DataPath — irrelevant if the carrier board routes data via PCIe.
	USBProtocol string `json:"usb_protocol,omitempty"`
	// DataPath is which physical interface carries the data path:
	// "USB" or "PCIe" (PCIe is typically wired to a Realtek ethernet
	// bridge chip on M.2 carrier boards). "" if unknown.
	DataPath string `json:"data_path,omitempty"`
}

// BandSet is a pair of LTE/NR5G band number lists.
type BandSet struct {
	LTE  []int `json:"lte"`
	NR5G []int `json:"nr5g"`
}

type SimInfo struct {
	Slot       int    `json:"slot"`
	IMSI       string `json:"imsi"`
	ICCID      string `json:"iccid"`
	Operator   string `json:"operator"`
	APN        string `json:"apn"`
	APNIP      string `json:"apn_ip"`
	Registered bool   `json:"registered"`
	// IPv4, IPv6, Gateway, DNS are the negotiated PDP-context values
	// returned by AT+CGCONTRDP — the modem's view of the connection,
	// which may differ from what the host kernel sees on its data
	// interface (especially under QMI/MBIM where the modem manages IP).
	IPv4    string   `json:"ipv4,omitempty"`
	IPv6    string   `json:"ipv6,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,omitempty"`
}

type Signal struct {
	RSRP int     `json:"rsrp"`
	RSRQ int     `json:"rsrq"`
	SINR float64 `json:"sinr"`
	Tech string  `json:"tech"`
	// State is the modem's QENG state field: CONNECT, NOCONN, SEARCH,
	// LIMSRV, NOSRV. Frontend renders SEARCH/NOSRV as "acquiring".
	State string `json:"state,omitempty"`
}

type CellInfo struct {
	Tech      string     `json:"tech"`
	State     string     `json:"state,omitempty"`
	MCC       string     `json:"mcc"`
	MNC       string     `json:"mnc"`
	CellID    string     `json:"cell_id"`
	PCI       int        `json:"pci"`
	Band      string     `json:"band"`
	Bandwidth string     `json:"bandwidth"`
	Signal    Signal     `json:"signal"`
	// SCells lists additional active cells beyond the primary: the NR5G
	// leg of an NR5G-NSA pairing, and any LTE/NR5G carrier-aggregation
	// secondaries reported by AT+QCAINFO.
	SCells    []CellInfo `json:"scells,omitempty"`
	Neighbors []CellInfo `json:"neighbors,omitempty"`
}

// Settings is the GET response shape (all fields populated).
//
// The frontend speaks in terms of human-meaningful groupings; the per-modem
// implementation owns the mapping to AT commands. NetworkMode is one of:
//
//	"AUTO"           — anything; modem picks best
//	"LTE_ONLY"       — LTE only
//	"NR5G_NSA_ONLY"  — NR5G non-standalone (LTE anchor + NR5G)
//	"NR5G_SA_ONLY"   — NR5G standalone
//	"CUSTOM"         — modem is in a state we can't map; surfaced read-only
type Settings struct {
	APN      APNSettings      `json:"apn"`
	Network  NetworkSettings  `json:"network"`
	Bands    BandSet          `json:"bands"`
	CellLock CellLockSettings `json:"cell_lock"`
}

type APNSettings struct {
	Name   string `json:"name"`
	IPType string `json:"ip_type"` // "IPV4" | "IPV6" | "IPV4V6"
}

type NetworkSettings struct {
	Mode string `json:"mode"`
}

type CellLockSettings struct {
	Lock4G bool `json:"lock_4g"`
	Lock5G bool `json:"lock_5g"`
}

// SettingsUpdate is the PUT request shape — pointer fields so absent != zero.
// Only groups with non-nil pointers are applied. Within a group, all fields
// are sent (zero values overwrite — the frontend echoes the unchanged ones
// back when submitting a partial form).
type SettingsUpdate struct {
	APN      *APNSettings      `json:"apn,omitempty"`
	Network  *NetworkSettings  `json:"network,omitempty"`
	Bands    *BandSet          `json:"bands,omitempty"`
	CellLock *CellLockSettings `json:"cell_lock,omitempty"`
}

type ScanResult struct {
	Operator string `json:"operator"`
	MCC      string `json:"mcc"`
	MNC      string `json:"mnc"`
	Tech     string `json:"tech"`
	State    string `json:"state"`
}

// CellSurveyResult is one cell discovered by an active RF scan
// (AT+QSCAN=3,1 on Quectel). Unlike ScanResult, this is per-cell, not
// per-operator — useful for picking a cell to lock to.
type CellSurveyResult struct {
	Tech   string `json:"tech"`     // "LTE" or "NR5G"
	Band   int    `json:"band"`     // numeric band (e.g. 71 for n71/B71)
	Freq   int    `json:"freq"`     // EARFCN (LTE) or ARFCN (NR5G)
	PCI    int    `json:"pci"`
	RSRP   int    `json:"rsrp"`     // dBm
	RSRQ   int    `json:"rsrq"`     // dB
	SCS    int    `json:"scs"`      // NR5G subcarrier spacing index; 0 for LTE
	MCC    string `json:"mcc"`
	MNC    string `json:"mnc"`
	CellID string `json:"cell_id"`
}

type SMS struct {
	Index int    `json:"index"`
	From  string `json:"from"`
	Time  string `json:"time"`
	Body  string `json:"body"`
	Read  bool   `json:"read"`
}

// Modem is the operations surface the frontend consumes via /api/modem/*.
type Modem interface {
	Info() Info
	Capabilities() []Capability

	GetSim(ctx context.Context) (*SimInfo, error)
	GetCell(ctx context.Context) (*CellInfo, error)
	GetSignal(ctx context.Context) (*Signal, error)
	GetSettings(ctx context.Context) (*Settings, error)
	SetSettings(ctx context.Context, u *SettingsUpdate) (*Settings, error)
	// LockCurrentCell pins the modem to its current serving cell on the
	// given technology. tech is "4g" or "5g".
	LockCurrentCell(ctx context.Context, tech string) error
	// LockCell pins the modem to a specific cell. tech is "4g" or "5g";
	// freq is EARFCN (LTE) or ARFCN (NR5G); pci is the cell's physical ID;
	// scsIdx is the NR5G subcarrier-spacing index as reported by QENG/QSCAN
	// (0=15kHz, 1=30kHz, 2=60kHz, 3=120kHz, 4=240kHz), ignored for LTE.
	// band is the numeric band (e.g. 71 for n71) — required for NR5G,
	// ignored for LTE.
	LockCell(ctx context.Context, tech string, freq, pci, scsIdx, band int) error
	// Reboot performs a full modem reset (CFUN=1,1 on Quectel).
	Reboot(ctx context.Context) error
	// SetDataPath switches the primary data interface between "USB" and
	// "PCIe". Takes effect on the next reboot. Returns ErrUnsupported on
	// modems that don't expose this knob.
	SetDataPath(ctx context.Context, path string) error

	Scan(ctx context.Context) ([]ScanResult, error)
	// CellSurvey performs an active RF sweep and returns every visible cell.
	// Slow (30-90s) but produces per-cell signal data the user needs to
	// decide which cell to lock to.
	CellSurvey(ctx context.Context) ([]CellSurveyResult, error)
	ListSMS(ctx context.Context) ([]SMS, error)
	DeleteSMS(ctx context.Context, index int) error
	// DeleteSMSBulk removes all messages matching scope: "read" deletes only
	// messages with read status, "all" deletes everything.
	DeleteSMSBulk(ctx context.Context, scope string) error
	// MarkAllRead reads each unread message in turn (the only way Quectel
	// transitions REC UNREAD → REC READ is via AT+CMGR side effect).
	MarkAllRead(ctx context.Context) error
	SendSMS(ctx context.Context, to, body string) error
}

// Constructor builds a Modem given an AT server and pre-populated Info.
type Constructor func(at atserver.ATServer, info Info) (Modem, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Constructor{}
)

// Register makes a Constructor available under model name. Each impl calls
// Register from its init() so adding a new modem requires no central edit.
func Register(model string, c Constructor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[model] = c
}

// lookup is package-private to avoid leaking the registry shape.
func lookup(model string) (Constructor, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	c, ok := registry[model]
	return c, ok
}
