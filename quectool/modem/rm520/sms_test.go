package rm520

import (
	"testing"
)

func TestParseSMSList(t *testing.T) {
	// AT+CMGL="ALL" response, text mode:
	// +CMGL: 1,"REC READ","+15551234","","24/04/27,10:30:00-32"
	// Hello world
	// +CMGL: 2,"REC UNREAD","+15555678","","24/04/27,11:00:00-32"
	// Second message
	lines := []string{
		`+CMGL: 1,"REC READ","+15551234","","24/04/27,10:30:00-32"`,
		`Hello world`,
		`+CMGL: 2,"REC UNREAD","+15555678","","24/04/27,11:00:00-32"`,
		`Second message`,
	}
	got, err := parseSMSList(lines)
	if err != nil {
		t.Fatalf("parseSMSList: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Index != 1 || got[0].From != "+15551234" || got[0].Body != "Hello world" || !got[0].Read {
		t.Errorf("unexpected first: %+v", got[0])
	}
	if got[1].Read {
		t.Errorf("expected second to be unread: %+v", got[1])
	}
}
