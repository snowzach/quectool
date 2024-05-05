package atserver

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"syscall"
	"time"

	"github.com/snowzach/golib/log"
	"golang.org/x/sys/unix"
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
	Status   ATStatus `json:"status"`
	Response []string `json:"response"`
}

type ATServer interface {
	SendCMD(ctx context.Context, cmd string) (*ATResponse, error)
	Close() error
}

type atServer struct {
	logger   *slog.Logger
	fd       int
	response chan []byte
	timeout  time.Duration
	mu       sync.Mutex
}

func NewATServer(portName string, timeout time.Duration) (ATServer, error) {
	return newATServer(portName, timeout)
}

func newATServer(portName string, timeout time.Duration) (*atServer, error) {

	// Open and close to clear the port
	fd, err := syscall.Open(portName, syscall.O_RDWR|syscall.O_LARGEFILE|syscall.O_CLOEXEC, 0666)
	if err != nil {
		return nil, fmt.Errorf("unable to open to clear port %s: %v", portName, err)
	}

	if err := syscall.Close(fd); err != nil {
		return nil, fmt.Errorf("unable to close after clear: %v", err)
	}

	// Open the port
	fd, err = syscall.Open(portName, syscall.O_RDWR|syscall.O_LARGEFILE|syscall.O_CLOEXEC, 0666)
	if err != nil {
		return nil, fmt.Errorf("unable to open port %s: %v", "", err)
	}

	ats := &atServer{
		logger:   log.Logger.With("context", "atserver", "port", portName),
		fd:       fd,
		response: make(chan []byte),
		timeout:  timeout,
	}

	go func() {
		buffer := make([]byte, 8192)
		var pos int

		for {
			n, err := unix.Read(fd, buffer[pos:])
			if err != nil {
				log.Error("unable to read response", "error", err)
				continue
			}
			pos += n

			log.Debug("Got port data", slog.String("data", string(buffer[:pos])))

			// Read until we get an OK or ERROR
			if bytes.Contains(buffer[:pos], []byte("\r\nOK\r\n")) || bytes.Contains(buffer[:pos], []byte("ERROR\r\n")) {
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

func (ats *atServer) SendCMD(ctx context.Context, cmd string) (*ATResponse, error) {

	ats.mu.Lock()
	defer ats.mu.Unlock()

	log.Debug("Sent port data", "data", string(cmd))

	// Write command
	if n, err := syscall.Write(ats.fd, []byte(cmd+"\r\n")); err != nil {
		return nil, fmt.Errorf("unable to send command %d: %v", n, err)
	}

	// Wait for response/cancel/timeout
	var response []byte
	select {
	case response = <-ats.response:
	case <-ctx.Done():
		return nil, context.Canceled
	case <-time.After(ats.timeout):
		return nil, fmt.Errorf("timeout waiting for response")
	}

	ret := &ATResponse{
		Status: ATStatusUnknown,
	}

	var header, trailer int

	// It may or may not have our command prefix depending on how it's connected, if it does, remove it.
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
	return syscall.Close(ats.fd)
}
