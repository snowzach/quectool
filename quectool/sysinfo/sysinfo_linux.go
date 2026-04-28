package sysinfo

import (
	"context"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func Get(ctx context.Context) (*SysInfo, error) {
	sysinfo := &unix.Sysinfo_t{}
	if err := unix.Sysinfo(sysinfo); err != nil {
		return nil, err
	}
	host, _ := os.Hostname()

	return &SysInfo{
		Hostname:      host,
		Uptime:        int(sysinfo.Uptime),
		ProcessUptime: int(time.Since(startTime).Seconds()),
		Loads: [3]uint{
			uint(sysinfo.Loads[0]),
			uint(sysinfo.Loads[1]),
			uint(sysinfo.Loads[2]),
		},
		TotalRam:   uint(sysinfo.Totalram),
		FreeRam:    uint(sysinfo.Freeram),
		SharedRam:  uint(sysinfo.Sharedram),
		BufferRam:  uint(sysinfo.Bufferram),
		Procs:      uint(sysinfo.Procs),
		Interfaces: collectInterfaces(),
	}, nil
}

// collectInterfaces enumerates every up, non-loopback interface with at
// least one assigned address. Errors are swallowed — a partial list is more
// useful than failing the whole sysinfo call over a flaky NIC.
func collectInterfaces() []NetIface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]NetIface, 0, len(ifaces))
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		ni := NetIface{Name: ifc.Name, MAC: ifc.HardwareAddr.String()}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			s := ipnet.String()
			if v4 := ipnet.IP.To4(); v4 != nil {
				ni.IPv4 = append(ni.IPv4, s)
			} else {
				// Skip link-local fe80:: — they're noise.
				if strings.HasPrefix(ipnet.IP.String(), "fe80:") {
					continue
				}
				ni.IPv6 = append(ni.IPv6, s)
			}
		}
		if len(ni.IPv4) == 0 && len(ni.IPv6) == 0 {
			continue
		}
		out = append(out, ni)
	}
	return out
}
