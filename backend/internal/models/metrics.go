package models

// --- Linux ---

type CPUMetrics struct {
	Percent float64   `json:"percent"`
	Cores   []float64 `json:"cores"`
	LoadAvg []float64 `json:"load_avg"`
}

type MemoryMetrics struct {
	Total     int64   `json:"total"`
	Used      int64   `json:"used"`
	Available int64   `json:"available"`
	Percent   float64 `json:"percent"`
}

type DiskMetrics struct {
	Mount   string  `json:"mount"`
	Total   int64   `json:"total"`
	Used    int64   `json:"used"`
	Free    int64   `json:"free"`
	Percent float64 `json:"percent"`
}

type NetworkMetrics struct {
	BytesSent    int64   `json:"bytes_sent"`
	BytesRecv    int64   `json:"bytes_recv"`
	UploadRate   float64 `json:"upload_rate"`
	DownloadRate float64 `json:"download_rate"`
}

type TemperatureMetrics struct {
	Label    string   `json:"label"`
	Current  float64  `json:"current"`
	High     *float64 `json:"high,omitempty"`
	Critical *float64 `json:"critical,omitempty"`
}

type FanMetrics struct {
	Label string `json:"label"`
	RPM   int64  `json:"rpm"`
}

type ProcessMetrics struct {
	PID           int     `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   int64   `json:"memory_usage"`
	MemoryPercent float64 `json:"memory_percent"`
}

type LinuxMetrics struct {
	CPU          CPUMetrics           `json:"cpu"`
	Memory       MemoryMetrics        `json:"memory"`
	Disks        []DiskMetrics        `json:"disks"`
	Network      NetworkMetrics       `json:"network"`
	Temperatures []TemperatureMetrics `json:"temperatures"`
	Fans         []FanMetrics         `json:"fans"`
	Processes    []ProcessMetrics     `json:"processes"`
	Uptime       int64                `json:"uptime"`
}

// --- Docker ---

type ContainerMetrics struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Image         string  `json:"image"`
	Status        string  `json:"status"`
	State         string  `json:"state"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   int64   `json:"memory_usage"`
	MemoryLimit   int64   `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
}

type DockerMetrics struct {
	Containers   []ContainerMetrics `json:"containers"`
	RunningCount int32              `json:"running_count"`
	StoppedCount int32              `json:"stopped_count"`
	TotalCount   int32              `json:"total_count"`
}

// --- qBittorrent ---

type TorrentInfo struct {
	Hash          string  `json:"hash"`
	Name          string  `json:"name"`
	Size          int64   `json:"size"`
	Progress      float64 `json:"progress"`
	DownloadSpeed int64   `json:"download_speed"`
	UploadSpeed   int64   `json:"upload_speed"`
	State         string  `json:"state"`
	ETA           *int64  `json:"eta,omitempty"`
}

type QBittorrentMetrics struct {
	DownloadSpeed   int64         `json:"download_speed"`
	UploadSpeed     int64         `json:"upload_speed"`
	ActiveDownloads int32         `json:"active_downloads"`
	ActiveUploads   int32         `json:"active_uploads"`
	TotalTorrents   int32         `json:"total_torrents"`
	Torrents        []TorrentInfo `json:"torrents"`
}

// --- UniFi ---

type UnifiDevice struct {
	Name   string `json:"name"`
	MAC    string `json:"mac"`
	Model  string `json:"model"`
	Status string `json:"status"`
	Uptime int64  `json:"uptime"`
}

type UnifiMetrics struct {
	Devices      []UnifiDevice `json:"devices"`
	ClientsCount int32         `json:"clients_count"`
	GuestsCount  int32         `json:"guests_count"`
	WANDownload  float64       `json:"wan_download"`
	WANUpload    float64       `json:"wan_upload"`
}

// --- UNAS ---

type StoragePool struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Total     int64   `json:"total"`
	Used      int64   `json:"used"`
	Available int64   `json:"available"`
	Percent   float64 `json:"percent"`
}

type StorageDisk struct {
	Name        string   `json:"name"`
	Model       string   `json:"model"`
	Serial      string   `json:"serial"`
	Status      string   `json:"status"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type UNASMetrics struct {
	Pools         []StoragePool `json:"pools"`
	Disks         []StorageDisk `json:"disks"`
	TotalCapacity int64         `json:"total_capacity"`
	UsedCapacity  int64         `json:"used_capacity"`
}
