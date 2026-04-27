package sysinfo

import "time"

// startTime is captured at package load. Subtracting from now() gives the
// "this backend has been running for X" figure shown on the dashboard.
var startTime = time.Now()

type SysInfo struct {
	Hostname      string      `json:"hostname"`
	Uptime        int         `json:"uptime"`         // host kernel uptime, seconds
	ProcessUptime int         `json:"process_uptime"` // seconds since the quectool backend started
	Loads         [3]uint     `json:"loads"`
	TotalRam      uint        `json:"total_ram"`
	FreeRam       uint        `json:"free_ram"`
	SharedRam     uint        `json:"shared_ram"`
	BufferRam     uint        `json:"buffer_ram"`
	Procs         uint        `json:"procs"`
	Interfaces    []NetIface  `json:"interfaces"`
}

// NetIface is one up-and-running network interface with its assigned addresses.
// Loopback is filtered out — boring on a dashboard.
type NetIface struct {
	Name string   `json:"name"`
	MAC  string   `json:"mac,omitempty"`
	IPv4 []string `json:"ipv4,omitempty"`
	IPv6 []string `json:"ipv6,omitempty"`
}
