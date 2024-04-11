package cron

type MachineConfig struct {
	Config Config `json:"config"`
}

type Config struct {
	AutoDestroy bool          `json:"auto_destroy"`
	Guest       GuestConfig   `json:"guest"`
	Image       string        `json:"image"`
	Restart     RestartConfig `json:"restart"`
}

type RestartConfig struct {
	MaxRetries int    `json:"max_retries"`
	Policy     string `json:"policy"`
}

type GuestConfig struct {
	CPUKind  string `json:"cpu_kind"`
	CPUs     int    `json:"cpus"`
	MemoryMB int    `json:"memory_mb"`
}
