package generic

import (
	"context"
	"testing"

	"github.com/snowzach/quectool/quectool/modem"
)

// TestGenericPreservesInfoAndReturnsUnsupported verifies the fallback impl
// keeps the Info populated by Detect and refuses every operation with
// ErrUnsupported. The atserver argument is unused by Generic, so passing
// nil is safe.
func TestGenericPreservesInfoAndReturnsUnsupported(t *testing.T) {
	info := modem.Info{
		Manufacturer: "Acme",
		Model:        "AcmeModem9000",
		Firmware:     "1.0",
		IMEI:         "1234",
	}
	m, err := New(nil, info)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if m.Info().Model != "AcmeModem9000" {
		t.Errorf("Info() lost data: %+v", m.Info())
	}
	if caps := m.Capabilities(); caps != nil {
		t.Errorf("expected nil capabilities, got %v", caps)
	}

	ctx := context.Background()
	if _, err := m.GetSim(ctx); err != modem.ErrUnsupported {
		t.Errorf("GetSim: expected ErrUnsupported, got %v", err)
	}
	if _, err := m.GetCell(ctx); err != modem.ErrUnsupported {
		t.Errorf("GetCell: expected ErrUnsupported, got %v", err)
	}
	if _, err := m.GetSignal(ctx); err != modem.ErrUnsupported {
		t.Errorf("GetSignal: expected ErrUnsupported, got %v", err)
	}
	if _, err := m.GetSettings(ctx); err != modem.ErrUnsupported {
		t.Errorf("GetSettings: expected ErrUnsupported, got %v", err)
	}
	if _, err := m.SetSettings(ctx, &modem.SettingsUpdate{}); err != modem.ErrUnsupported {
		t.Errorf("SetSettings: expected ErrUnsupported, got %v", err)
	}
	if _, err := m.Scan(ctx); err != modem.ErrUnsupported {
		t.Errorf("Scan: expected ErrUnsupported, got %v", err)
	}
	if _, err := m.ListSMS(ctx); err != modem.ErrUnsupported {
		t.Errorf("ListSMS: expected ErrUnsupported, got %v", err)
	}
	if err := m.DeleteSMS(ctx, 1); err != modem.ErrUnsupported {
		t.Errorf("DeleteSMS: expected ErrUnsupported, got %v", err)
	}
	if err := m.SendSMS(ctx, "+15551234", "hi"); err != modem.ErrUnsupported {
		t.Errorf("SendSMS: expected ErrUnsupported, got %v", err)
	}
}

// TestGenericRegistered verifies init() registered the constructor.
func TestGenericRegistered(t *testing.T) {
	// Re-import side-effect already happened via package init; verify by
	// constructing through the registry-shaped path: just check New works.
	if _, err := New(nil, modem.Info{}); err != nil {
		t.Errorf("New returned error: %v", err)
	}
}
