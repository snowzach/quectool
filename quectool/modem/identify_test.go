package modem

import (
	"context"
	"testing"
	"time"

	"github.com/snowzach/quectool/quectool/atserver"
)

// fakeATServer is a deterministic ATServer for testing identification +
// per-modem parsers. It maps an exact command string to a canned response.
type fakeATServer struct {
	responses map[string]*atserver.ATResponse
}

func (f *fakeATServer) SendCMD(_ context.Context, cmd string, _ time.Duration) (*atserver.ATResponse, error) {
	r, ok := f.responses[cmd]
	if !ok {
		return &atserver.ATResponse{Command: cmd, Status: atserver.ATStatusError}, nil
	}
	return r, nil
}
func (f *fakeATServer) SendMessage(_ context.Context, cmd, _ string, _ time.Duration) (*atserver.ATResponse, error) {
	r, ok := f.responses[cmd]
	if !ok {
		return &atserver.ATResponse{Command: cmd, Status: atserver.ATStatusError}, nil
	}
	return r, nil
}
func (f *fakeATServer) Close() error { return nil }

func TestIdentify(t *testing.T) {
	f := &fakeATServer{responses: map[string]*atserver.ATResponse{
		"AT+CGMI": {Status: atserver.ATStatusOK, Response: []string{"Quectel"}},
		"AT+CGMM": {Status: atserver.ATStatusOK, Response: []string{"RM520N-GL"}},
		"AT+CGMR": {Status: atserver.ATStatusOK, Response: []string{"RM520NGLAAR03A03M4G"}},
		"AT+CGSN": {Status: atserver.ATStatusOK, Response: []string{"869410070123456"}},
	}}
	got, err := identify(context.Background(), f)
	if err != nil {
		t.Fatalf("identify: %v", err)
	}
	if got.Manufacturer != "Quectel" || got.Model != "RM520N-GL" ||
		got.Firmware != "RM520NGLAAR03A03M4G" || got.IMEI != "869410070123456" {
		t.Errorf("unexpected info: %+v", got)
	}
}
