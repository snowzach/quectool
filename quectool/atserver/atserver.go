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
)

func (s ATStatus) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
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
	Close() error
}

type atServer struct {
	logger   *slog.Logger
	port     io.ReadWriteCloser
	response chan []byte
	timeout  time.Duration
	mu       sync.Mutex
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
		response: make(chan []byte),
		timeout:  timeout,
	}

	go func() {
		buffer := make([]byte, 8192)
		var (
			pos               int
			emptyMessageCount int
		)

		for {
			n, err := ats.port.Read(buffer[pos:])
			if err != nil {
				log.Error("unable to read response", "error", err)
				continue
			}
			pos += n

			log.Debug("Got port data", slog.String("data", string(buffer[:pos])))

			// If the port keeps streaming empty data, it needs reset
			if pos == 0 {
				emptyMessageCount++
				if emptyMessageCount > 20 {
					// Reset the port
					ats.mu.Lock()
					if err := ats.port.Close(); err != nil {
						log.Error("Unable to close port", "error", err)
					}
					// Wait for modem to come around
					for retries := 10; retries > 0; retries-- {
						ats.port, err = NewPort(portName)
						if err != nil {
							log.Info("Port not ready. Sleeping.", "error", err)
							time.Sleep(5 * time.Second)
						} else {
							break
						}
					}
					if ats.port == nil {
						log.Fatal("Unable to reopen port after error")
					} else {
						log.Info("Port reopened")
					}
					ats.mu.Unlock()
					emptyMessageCount = 0
				}
			} else {
				emptyMessageCount = 0
			}

			// Read until we get an OK or ERROR
			if bytes.Contains(buffer[:pos], []byte("\r\nOK\r\n")) || bytes.Contains(buffer[:pos], []byte("ERROR\r\n")) || bytes.Contains(buffer[:pos], []byte("ERROR:")) {
				ret := make([]byte, pos)
				copy(ret, buffer[:pos])
				select {
				case ats.response <- ret:
					// Send response
				case <-time.After(1 * time.Second):
					// No one is listening
				}
				pos = 0
			}
		}
	}()

	return ats, nil

}

func (ats *atServer) SendCMD(ctx context.Context, cmd string, timeout time.Duration) (*ATResponse, error) {

	ats.mu.Lock()
	defer ats.mu.Unlock()

	log.Debug("Sent port data", "data", string(cmd))

	// Write command
	if n, err := ats.port.Write([]byte(cmd + "\r\n")); err != nil {
		return nil, fmt.Errorf("unable to send command %d: %v", n, err)
	}

	if timeout == 0 {
		timeout = ats.timeout
	}

	// Wait for response/cancel/timeout
	var response []byte
	select {
	case response = <-ats.response:
	case <-ctx.Done():
		return nil, context.Canceled
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response")
	}

	var (
		ret = &ATResponse{
			Command: cmd,
			Status:  ATStatusUnknown,
		}
		header, trailer int
	)

	// If it has the command in the output, mark it for removal
	if bytes.HasPrefix(response, []byte(cmd+"\r")) {
		header = len(cmd) + 1
	}

	// Parse the OK/ERROR strings
	if bytes.HasSuffix(response, []byte("\r\nOK\r\n")) {
		ret.Status = ATStatusOK
		trailer = 6
	} else if bytes.HasSuffix(response, []byte("\r\nERROR\r\n")) {
		ret.Status = ATStatusError
		trailer = 9
	}

	for _, line := range bytes.Split(response[header:len(response)-trailer], []byte("\r\n")) {
		if len(line) == 0 {
			continue
		}
		ret.Response = append(ret.Response, string(line))
	}

	return ret, nil

}

func (ats *atServer) Close() error {
	return ats.port.Close()
}
