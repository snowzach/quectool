package sysinfo

type SysInfo struct {
	Uptime    int64     `json:"uptime"`
	Loads     [3]uint64 `json:"loads"`
	TotalRam  uint64    `json:"total_ram"`
	FreeRam   uint64    `json:"free_ram"`
	SharedRam uint64    `json:"shared_ram"`
	BufferRam uint64    `json:"buffer_ram"`
	Procs     uint16    `json:"procs"`
}
