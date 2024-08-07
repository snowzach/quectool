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
		Uptime:    sysinfo.Uptime,
		Loads:     sysinfo.Loads,
		TotalRam:  sysinfo.Totalram,
		FreeRam:   sysinfo.Freeram,
		SharedRam: sysinfo.Sharedram,
		BufferRam: sysinfo.Bufferram,
		Procs:     sysinfo.Procs,
	}, nil
}
