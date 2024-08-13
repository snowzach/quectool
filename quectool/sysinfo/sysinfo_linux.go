package sysinfo

import (
	"context"

	"golang.org/x/sys/unix"
)

func Get(ctx context.Context) (*SysInfo, error) {
	sysinfo := &unix.Sysinfo_t{}
	if err := unix.Sysinfo(sysinfo); err != nil {
		return nil, err
	}

	return &SysInfo{
		Uptime: int(sysinfo.Uptime),
		Loads: [3]uint{
			uint(sysinfo.Loads[0]),
			uint(sysinfo.Loads[1]),
			uint(sysinfo.Loads[2]),
		},
		TotalRam:  uint(sysinfo.Totalram),
		FreeRam:   uint(sysinfo.Freeram),
		SharedRam: uint(sysinfo.Sharedram),
		BufferRam: uint(sysinfo.Bufferram),
		Procs:     uint(sysinfo.Procs),
	}, nil
}
