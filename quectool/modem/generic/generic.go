// Package generic provides a fallback Modem implementation for unknown
// modem models. It returns Info derived from identification and
// ErrUnsupported for every operation.
package generic

import (
	"context"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/modem"
)

func init() {
	modem.Register("__generic__", New)
}

type Generic struct{ info modem.Info }

func New(_ atserver.ATServer, info modem.Info) (modem.Modem, error) {
	return &Generic{info: info}, nil
}

func (g *Generic) Info() modem.Info                 { return g.info }
func (g *Generic) Capabilities() []modem.Capability { return nil }

func (g *Generic) GetSim(context.Context) (*modem.SimInfo, error)       { return nil, modem.ErrUnsupported }
func (g *Generic) GetCell(context.Context) (*modem.CellInfo, error)     { return nil, modem.ErrUnsupported }
func (g *Generic) GetSignal(context.Context) (*modem.Signal, error)     { return nil, modem.ErrUnsupported }
func (g *Generic) GetSettings(context.Context) (*modem.Settings, error) { return nil, modem.ErrUnsupported }
func (g *Generic) SetSettings(context.Context, *modem.SettingsUpdate) (*modem.Settings, error) {
	return nil, modem.ErrUnsupported
}
func (g *Generic) LockCurrentCell(context.Context, string) error      { return modem.ErrUnsupported }
func (g *Generic) LockCell(context.Context, string, int, int, int, int) error {
	return modem.ErrUnsupported
}
func (g *Generic) Reboot(context.Context) error                   { return modem.ErrUnsupported }
func (g *Generic) SetDataPath(context.Context, string) error      { return modem.ErrUnsupported }
func (g *Generic) Scan(context.Context) ([]modem.ScanResult, error) { return nil, modem.ErrUnsupported }
func (g *Generic) CellSurvey(context.Context) ([]modem.CellSurveyResult, error) {
	return nil, modem.ErrUnsupported
}
func (g *Generic) ListSMS(context.Context) ([]modem.SMS, error)     { return nil, modem.ErrUnsupported }
func (g *Generic) DeleteSMS(context.Context, int) error             { return modem.ErrUnsupported }
func (g *Generic) DeleteSMSBulk(context.Context, string) error      { return modem.ErrUnsupported }
func (g *Generic) MarkAllRead(context.Context) error                { return modem.ErrUnsupported }
func (g *Generic) SendSMS(context.Context, string, string) error { return modem.ErrUnsupported }
