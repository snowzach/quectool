//go:build windows
// +build windows

package atserver

import (
	"fmt"

	"github.com/albenik/go-serial/v2"
)

func NewPort(portName string) (*serial.Port, error) {
	port, err := serial.Open(portName,
		serial.WithBaudrate(115200),
		serial.WithDataBits(8),
		serial.WithParity(serial.NoParity),
		serial.WithStopBits(serial.OneStopBit),
		serial.WithReadTimeout(1000),
		serial.WithWriteTimeout(1000),
		serial.WithHUPCL(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open port: %w", err)
	}

	_, _ = port.Write([]byte("\r\n"))

	return port, nil

}
