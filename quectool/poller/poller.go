package poller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/snowzach/golib/log"
	"github.com/snowzach/golib/signal"
	"github.com/snowzach/quectool/quectool/atserver"
)

type Status struct {
	Model    string `json:"model"`
	Firmware string `json:"firmware"`
	APN      string `json:"apn"`
	SimSlot  string `json:"sim_slot"`

	Provider string `json:"provider"`
}

type Poller interface {
	Status() Status
}

type poller struct {
	logger   *slog.Logger
	atserver atserver.ATServer
	status   Status
	sync.Mutex
}

func NewPoller(atserver atserver.ATServer, infoInterval time.Duration, statusInterval time.Duration) (Poller, error) {
	return newPoller(atserver, infoInterval, statusInterval)
}

func newPoller(atserver atserver.ATServer, infoInterval time.Duration, statusInterval time.Duration) (*poller, error) {

	p := &poller{
		logger:   log.Logger.With("context", "poller"),
		atserver: atserver,
	}

	go func() {
		for {
			err := p.infoPoll(signal.Stop.Context())
			if err != nil {
				p.logger.Error("error polling info", "error", err)
			}
			time.Sleep(infoInterval)
		}
	}()

	go func() {
		for {
			err := p.statusPoll(signal.Stop.Context())
			if err != nil {
				p.logger.Error("error polling status", "error", err)
			}
			time.Sleep(statusInterval)
		}
	}()

	return p, nil

}

func (p *poller) infoPoll(ctx context.Context) error {

	r, err := p.atserver.SendCMD(ctx, "AT+CGMM;+QGMR;+CGCONTRDP=1;+QUIMSLOT?")
	if err != nil {
		return fmt.Errorf("error on info poll: %w", err)
	}

	if r.Status != atserver.ATStatusOK {
		return errors.New("error on info poll")
	}

	p.Lock()
	defer p.Unlock()

	p.status.Model = r.Response[0]
	p.status.Firmware = r.Response[1]

	for _, response := range r.Response[2:] {
		if i := strings.IndexByte(response, ':'); i >= 0 {
			switch response[:i] {
			case "+CGCONTRDP":
				p.status.APN = extractFirstQuotedString(response[12:])
			case "+QUIMSLOT":
				p.status.SimSlot = response[11:]
			}
		}
	}

	return nil

}

func (p *poller) statusPoll(ctx context.Context) error {

	r, err := p.atserver.SendCMD(ctx, `AT+QSPN;+CEREG=2;+CEREG?;+CEREG=0;+C5GREG=2;+C5GREG?;+C5GREG=0;+CSQ;+QENG="servingcell";+QRSRP;+QCAINFO;+QNWPREFCFG="mode_pref";+QTEMP`)
	if err != nil {
		return fmt.Errorf("error getting network info: %w", err)
	}

	if r.Status != atserver.ATStatusOK {
		return errors.New("error on info poll")
	}

	p.Lock()
	defer p.Unlock()

	// (*quectool.ATResponse)(0x100d800)({
	// 	Status: (quectool.ATStatus) 1,
	// 	Response: ([]string) (len=27 cap=32) {
	// 	 (string) (len=11) "+CEREG: 2,0",
	// 	 (string) (len=43) "+C5GREG: 2,1,\"B0FB00\",\"106497003\",11,1,\"01\"",
	// 	 (string) (len=11) "+CSQ: 99,99",
	// 	 (string) (len=101) "+QENG: \"servingcell\",\"NOCONN\",\"NR5G-SA\",\"FDD\",310,260,106497003,869,B0FB00,124350,71,3,-84,-11,21,0,-",
	// 	 (string) (len=34) "+QRSRP: -90,-83,-32768,-32768,NR5G",
	// 	 (string) (len=43) "+QCAINFO: \"PCC\",124350,3,\"NR5G BAND 71\",869",
	// 	 (string) (len=52) "+QCAINFO: \"SCC\",520110,12,\"NR5G BAND 41\",1,290,0,-,-",
	// 	 (string) (len=51) "+QCAINFO: \"SCC\",502110,6,\"NR5G BAND 41\",1,290,0,-,-",
	// 	 (string) (len=29) "+QNWPREFCFG: \"mode_pref\",NR5G",
	// 	 (string) (len=32) "+QTEMP:\"modem-lte-sub6-pa1\",\"36\"",
	// 	 (string) (len=27) "+QTEMP:\"modem-sdr0-pa0\",\"0\"",
	// 	 (string) (len=27) "+QTEMP:\"modem-sdr0-pa1\",\"0\"",
	// 	 (string) (len=27) "+QTEMP:\"modem-sdr0-pa2\",\"0\"",
	// 	 (string) (len=27) "+QTEMP:\"modem-sdr1-pa0\",\"0\"",
	// 	 (string) (len=27) "+QTEMP:\"modem-sdr1-pa1\",\"0\"",
	// 	 (string) (len=27) "+QTEMP:\"modem-sdr1-pa2\",\"0\"",
	// 	 (string) (len=26) "+QTEMP:\"modem-mmw0\",\"-273\"",
	// 	 (string) (len=24) "+QTEMP:\"aoss-0-usr\",\"40\"",
	// 	 (string) (len=25) "+QTEMP:\"cpuss-0-usr\",\"39\"",
	// 	 (string) (len=25) "+QTEMP:\"mdmq6-0-usr\",\"39\"",
	// 	 (string) (len=25) "+QTEMP:\"mdmss-0-usr\",\"39\"",
	// 	 (string) (len=25) "+QTEMP:\"mdmss-1-usr\",\"38\"",
	// 	 (string) (len=25) "+QTEMP:\"mdmss-2-usr\",\"39\"",
	// 	 (string) (len=25) "+QTEMP:\"mdmss-3-usr\",\"39\"",
	// 	 (string) (len=32) "+QTEMP:\"modem-lte-sub6-pa2\",\"37\"",
	// 	 (string) (len=31) "+QTEMP:\"modem-ambient-usr\",\"37\""

	// for _, r := range r.Response {

	// 	if strings.HasPrefix(r, "+QSPN:") {
	// 		p.status.Network = r[7:]
	// 	}
	// }

	// s.status.Network = r.Response[0]
	// s.status.Signal = r.Response[1]

	for _, response := range r.Response {
		if i := strings.IndexByte(response, ':'); i >= 0 {
			switch response[:i] {
			case "+QSPN":
				p.status.Provider = extractFirstQuotedString(response[7:])
			case "+CEREG":
			}
		}
	}

	// spew.Dump(r)

	return nil

}

func (p *poller) Status() Status {
	p.Lock()
	defer p.Unlock()
	return p.status
}

// extractFirstQuotedString efficient string hunter
func extractFirstQuotedString(s string) string {
	if start := strings.IndexByte(s, '"'); start >= 0 {
		if end := strings.IndexByte(s[start+1:], '"'); end >= 0 {
			return s[start+1 : start+1+end]
		}
	}
	return ""
}
