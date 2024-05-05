//go:build linux
// +build linux

package atserver

import (
	"fmt"
	"syscall"
)

type Port struct {
	fd int
}

func NewPort(portName string) (*Port, error) {

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

	return &Port{
		fd: fd,
	}, nil
}

func (p *Port) Write(data []byte) (int, error) {
	return syscall.Write(p.fd, data)
}

func (p *Port) Read(data []byte) (int, error) {
	return syscall.Read(p.fd, data)
}

func (p *Port) Close() error {
	return syscall.Close(p.fd)
}
