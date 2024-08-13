package sysinfo

type SysInfo struct {
	Uptime    int     `json:"uptime"`
	Loads     [3]uint `json:"loads"`
	TotalRam  uint    `json:"total_ram"`
	FreeRam   uint    `json:"free_ram"`
	SharedRam uint    `json:"shared_ram"`
	BufferRam uint    `json:"buffer_ram"`
	Procs     uint    `json:"procs"`
}
