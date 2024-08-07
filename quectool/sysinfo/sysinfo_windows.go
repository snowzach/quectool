package sysinfo

import "context"

func Get(ctx context.Context) (*SysInfo, error) {
	// Not implemented for windows
	return &SysInfo{}, nil
}
