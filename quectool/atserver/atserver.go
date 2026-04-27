package atserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/snowzach/golib/log"
)

type ATStatus int

const (
	ATStatusUnknown ATStatus = iota
	ATStatusOK
	ATStatusError

	defaultBufferSize = 8192
)

var (
	jsonOK      = []byte(`"OK"`)
	jsonError   = []byte(`"ERROR"`)
	jsonUnknown = []byte(`"UNKNOWN"`)

	tokOK     = []byte("\r\nOK\r\n")
	tokErr    = []byte("\r\nERROR\r\n")
	tokErrCol = []byte("ERROR:")
	crlf      = []byte("\r\n")
)

func (s ATStatus) MarshalJSON() ([]byte, error) {
	switch s {
	case ATStatusOK:
		return jsonOK, nil
	case ATStatusError:
		return jsonError, nil
	default:
		return jsonUnknown, nil
	}
}

func (s ATStatus) String() string {
	switch s {
	case ATStatusOK:
		return "OK"
	case ATStatusError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type ATResponse struct {
	Command  string   `json:"command"`
	Status   ATStatus `json:"status"`
	Response []string `json:"response"`
}

type ATServer interface {
	SendCMD(ctx context.Context, cmd string, timeout time.Duration) (*ATResponse, error)
	// SendMessage handles AT commands with a body-input prompt (notably
	// AT+CMGS for SMS): writes "<cmd>\r", waits briefly for the modem to
	// emit `>`, then writes "<body>\x1a" and waits for OK/ERROR. The
	// readLoop only signals on OK/ERROR so we can't detect `>` directly —
	// a small fixed delay is reliable enough in practice.
	SendMessage(ctx context.Context, cmd, body string, timeout time.Duration) (*ATResponse, error)
	Close() error
}

type atServer struct {
	logger   *slog.Logger
	port     io.ReadWriteCloser
	response chan []byte
	timeout  time.Duration
	mu       sync.Mutex

	// scratchWrite is reused to build "<cmd>\r\n" without allocating per call.
	scratchWrite []byte
}

func NewATServer(portName string, timeout time.Duration) (ATServer, error) {
	return newATServer(portName, timeout)
}

func newATServer(portName string, timeout time.Duration) (*atServer, error) {

	port, err := NewPort(portName)
	if err != nil {
		return nil, err
	}

	ats := &atServer{
		logger:   log.Logger.With("context", "atserver", "port", portName),
		port:     port,
		response: make(chan []byte, 1),
		timeout:  timeout,
	}

	go ats.readLoop(portName)

	return ats, nil
}

func (ats *atServer) readLoop(portName string) {
	// buffer is owned by this goroutine only. It grows on demand and never
	// shrinks back, so steady-state allocation is zero.
	buffer := make([]byte, defaultBufferSize)
	var pos int
	// scanned tracks how far into buffer we have already searched, so each
	// iteration only scans new bytes (with a small overlap to catch a
	// terminator straddling two reads).
	var scanned int

	for {
		// Grow only on demand.
		if pos == len(buffer) {
			nb := make([]byte, len(buffer)*2)
			copy(nb, buffer)
			buffer = nb
		}

		n, err := ats.port.Read(buffer[pos:])
		if err == io.EOF {
			ats.logger.Info("Port EOF. Attempting to re-open", "error", err)
			ats.mu.Lock()
			_ = ats.port.Close()
			ats.port = nil
			var newPort io.ReadWriteCloser
			for retries := 10; retries > 0; retries-- {
				newPort, err = NewPort(portName)
				if err == nil {
					ats.port = newPort
					break
				}
				ats.logger.Info("Port not ready. Sleeping.", "error", err)
				time.Sleep(5 * time.Second)
			}
			if ats.port == nil {
				ats.mu.Unlock()
				log.Fatal("Unable to reopen port after EOF")
			}
			ats.mu.Unlock()
			ats.logger.Info("Port reopened")
			pos = 0
			scanned = 0
			continue
		} else if err != nil {
			ats.logger.Error("unable to read response", "error", err)
			continue
		}

		if ats.logger.Enabled(context.Background(), slog.LevelDebug) {
			ats.logger.Debug("Got port data", slog.String("data", string(buffer[pos:pos+n])))
		}

		pos += n

		// Only scan new bytes; back up by len(tokErr) to catch a token that
		// straddles the previous read boundary.
		start := max(scanned-len(tokErr), 0)
		region := buffer[start:pos]
		if bytes.Contains(region, tokOK) || bytes.Contains(region, tokErr) || bytes.Contains(region, tokErrCol) {
			// One alloc + copy per response is unavoidable with a channel
			// handoff, but it's bounded by the actual response size.
			ret := make([]byte, pos)
			copy(ret, buffer[:pos])
			// Channel is buffered (cap 1). If a previous response wasn't
			// consumed (caller timed out), drop it in favor of the new one.
			select {
			case ats.response <- ret:
			default:
				// Drain stale and try again.
				select {
				case <-ats.response:
				default:
				}
				select {
				case ats.response <- ret:
				default:
				}
			}
			pos = 0
			scanned = 0
			continue
		}
		scanned = pos
	}
}

func (ats *atServer) SendCMD(ctx context.Context, cmd string, timeout time.Duration) (*ATResponse, error) {

	ats.mu.Lock()
	defer ats.mu.Unlock()

	if ats.logger.Enabled(context.Background(), slog.LevelDebug) {
		ats.logger.Debug("Sent port data", "data", cmd)
	}

	// Drain any stale buffered response from a previous timed-out command.
	select {
	case <-ats.response:
	default:
	}

	// Build "<cmd>\r\n" into a reusable scratch buffer to avoid two allocs
	// (one for cmd+"\r\n", one for the []byte conversion).
	need := len(cmd) + 2
	if cap(ats.scratchWrite) < need {
		ats.scratchWrite = make([]byte, 0, need)
	}
	ats.scratchWrite = append(ats.scratchWrite[:0], cmd...)
	ats.scratchWrite = append(ats.scratchWrite, '\r', '\n')

	if n, err := ats.port.Write(ats.scratchWrite); err != nil {
		return nil, fmt.Errorf("unable to send command %d: %v", n, err)
	}

	if timeout == 0 {
		timeout = ats.timeout
	}

	// Use NewTimer so we can Stop() and avoid leaking a runtime timer when
	// the response or context cancellation arrives first.
	t := time.NewTimer(timeout)
	defer t.Stop()

	var response []byte
	select {
	case response = <-ats.response:
	case <-ctx.Done():
		return nil, context.Canceled
	case <-t.C:
		return nil, fmt.Errorf("timeout waiting for response")
	}

	ret := &ATResponse{
		Command: cmd,
		Status:  ATStatusUnknown,
	}
	var header, trailer int

	// If the response echoes the command, strip it.
	if len(response) > len(cmd) && response[len(cmd)] == '\r' && bytes.HasPrefix(response, []byte(cmd)) {
		header = len(cmd) + 1
	}

	if bytes.HasSuffix(response, tokOK) {
		ret.Status = ATStatusOK
		trailer = len(tokOK)
	} else if bytes.HasSuffix(response, tokErr) {
		ret.Status = ATStatusError
		trailer = len(tokErr)
	}

	body := response[header : len(response)-trailer]

	// Pre-size Response slice from a quick line count to avoid append regrowth.
	if n := bytes.Count(body, crlf); n > 0 {
		ret.Response = make([]string, 0, n+1)
	}

	// Manual line iteration; avoids bytes.Split's [][]byte allocation.
	for len(body) > 0 {
		i := bytes.Index(body, crlf)
		var line []byte
		if i < 0 {
			line = body
			body = nil
		} else {
			line = body[:i]
			body = body[i+2:]
		}
		if len(line) == 0 {
			continue
		}
		ret.Response = append(ret.Response, string(line))
	}

	return ret, nil
}

// SendMessage implements the prompt-then-body flow for AT+CMGS-style
// commands. The serial port write is locked end-to-end so a parallel
// SendCMD can't interleave writes mid-prompt.
func (ats *atServer) SendMessage(ctx context.Context, cmd, body string, timeout time.Duration) (*ATResponse, error) {
	ats.mu.Lock()
	defer ats.mu.Unlock()

	// Drain any stale buffered response.
	select {
	case <-ats.response:
	default:
	}

	// Stage 1: write "<cmd>\r" and let the modem emit `>`. We can't detect
	// the prompt directly (readLoop only signals on OK/ERROR), so wait a
	// small fixed delay. 250ms is enough on every Quectel firmware tested.
	if _, err := ats.port.Write([]byte(cmd + "\r")); err != nil {
		return nil, fmt.Errorf("write cmd: %w", err)
	}
	select {
	case <-time.After(250 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Stage 2: body + Ctrl-Z to submit. Ctrl-Z (0x1A) tells the modem to
	// send; ESC (0x1B) would cancel.
	if _, err := ats.port.Write([]byte(body + "\x1a")); err != nil {
		return nil, fmt.Errorf("write body: %w", err)
	}

	if timeout == 0 {
		timeout = ats.timeout
	}
	t := time.NewTimer(timeout)
	defer t.Stop()

	var response []byte
	select {
	case response = <-ats.response:
	case <-ctx.Done():
		return nil, context.Canceled
	case <-t.C:
		return nil, fmt.Errorf("timeout waiting for response")
	}

	ret := &ATResponse{Command: cmd, Status: ATStatusUnknown}
	switch {
	case bytes.HasSuffix(response, tokOK):
		ret.Status = ATStatusOK
		response = response[:len(response)-len(tokOK)]
	case bytes.HasSuffix(response, tokErr):
		ret.Status = ATStatusError
		response = response[:len(response)-len(tokErr)]
	}
	for _, line := range bytes.Split(response, crlf) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		ret.Response = append(ret.Response, string(line))
	}
	return ret, nil
}

func (ats *atServer) Close() error {
	ats.mu.Lock()
	defer ats.mu.Unlock()
	if ats.port == nil {
		return nil
	}
	return ats.port.Close()
}
