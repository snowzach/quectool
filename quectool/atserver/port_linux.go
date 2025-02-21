//go:build linux
// +build linux

package atserver

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/sys/unix"
)

type Port struct {
	fd int
}

// Create a new port... unix style (Thanks https://github.com/pkg/term)
func NewPort(portName string) (*Port, error) {

	fd, err := unix.Open(portName, unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NDELAY|unix.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("unable to open port %s: %w", portName, err)
	}

	// Get the attributes
	var attr unix.Termios
	err = unix.IoctlSetTermios(fd, unix.TCGETS, &attr)
	if err == nil {
		// Set to raw mode
		attr.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
		attr.Oflag &^= unix.OPOST
		attr.Cflag &^= unix.CSIZE | unix.PARENB
		attr.Cflag |= unix.CS8
		attr.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG
		attr.Cc[unix.VMIN] = 1
		attr.Cc[unix.VTIME] = 0
		if err := unix.IoctlSetTermios(fd, unix.TCSETS, &attr); err != nil {
			return nil, fmt.Errorf("unable to set port to raw mode: %w", err)
		}

	} else if err != nil && !strings.Contains(err.Error(), "inappropriate ioctl") {
		return nil, fmt.Errorf("unable to get port attributes %s: %w", portName, err)
	}

	// Set non-blocking
	if err := unix.SetNonblock(fd, false); err != nil {
		return nil, fmt.Errorf("unable to set port to non-block mode: %w", err)
	}

	return &Port{
		fd: fd,
	}, nil
}

func (p *Port) Write(data []byte) (int, error) {
	n, err := unix.Write(p.fd, data)
	if n < 0 {
		n = 0
	}
	if n != len(data) {
		return n, io.ErrShortWrite
	}
	return n, err
}

func (p *Port) Read(data []byte) (int, error) {
	n, err := unix.Read(p.fd, data)
	if n < 0 {
		n = 0
	}
	if n == 0 && len(data) > 0 && err == nil {
		return 0, io.EOF
	}
	return n, err
}

func (p *Port) Close() error {
	err := unix.Close(p.fd)
	p.fd = -1
	return err
}
