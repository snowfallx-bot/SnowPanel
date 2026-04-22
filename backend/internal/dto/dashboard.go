package dto

type DashboardSummary struct {
	Hostname      string  `json:"hostname"`
	SystemVersion string  `json:"system_version"`
	KernelVersion string  `json:"kernel_version"`
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	DiskUsage     float64 `json:"disk_usage"`
	Uptime        string  `json:"uptime"`
}
