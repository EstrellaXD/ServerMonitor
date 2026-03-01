use serde::{Deserialize, Serialize};

// --- Linux ---

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct CpuMetrics {
    pub percent: f64,
    #[serde(default)]
    pub cores: Vec<f64>,
    #[serde(default)]
    pub load_avg: Vec<f64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct MemoryMetrics {
    pub total: i64,
    pub used: i64,
    pub available: i64,
    pub percent: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiskMetrics {
    pub mount: String,
    #[serde(default)]
    pub total: i64,
    #[serde(default)]
    pub used: i64,
    #[serde(default)]
    pub free: i64,
    #[serde(default)]
    pub percent: f64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct NetworkMetrics {
    pub bytes_sent: i64,
    pub bytes_recv: i64,
    pub upload_rate: f64,
    pub download_rate: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemperatureMetrics {
    pub label: String,
    pub current: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub high: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub critical: Option<f64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct LinuxMetrics {
    pub cpu: CpuMetrics,
    pub memory: MemoryMetrics,
    #[serde(default)]
    pub disks: Vec<DiskMetrics>,
    pub network: NetworkMetrics,
    #[serde(default)]
    pub temperatures: Vec<TemperatureMetrics>,
    #[serde(default)]
    pub uptime: i64,
}

// --- Docker ---

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContainerMetrics {
    pub id: String,
    pub name: String,
    pub image: String,
    pub status: String,
    pub state: String,
    #[serde(default)]
    pub cpu_percent: f64,
    #[serde(default)]
    pub memory_usage: i64,
    #[serde(default)]
    pub memory_limit: i64,
    #[serde(default)]
    pub memory_percent: f64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct DockerMetrics {
    #[serde(default)]
    pub containers: Vec<ContainerMetrics>,
    #[serde(default)]
    pub running_count: i32,
    #[serde(default)]
    pub stopped_count: i32,
    #[serde(default)]
    pub total_count: i32,
}

// --- qBittorrent ---

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TorrentInfo {
    pub name: String,
    pub size: i64,
    pub progress: f64,
    pub download_speed: i64,
    pub upload_speed: i64,
    pub state: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub eta: Option<i64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct QBittorrentMetrics {
    #[serde(default)]
    pub download_speed: i64,
    #[serde(default)]
    pub upload_speed: i64,
    #[serde(default)]
    pub active_downloads: i32,
    #[serde(default)]
    pub active_uploads: i32,
    #[serde(default)]
    pub total_torrents: i32,
    #[serde(default)]
    pub torrents: Vec<TorrentInfo>,
}

// --- UniFi ---

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnifiDevice {
    pub name: String,
    pub mac: String,
    pub model: String,
    pub status: String,
    #[serde(default)]
    pub uptime: i64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct UnifiMetrics {
    #[serde(default)]
    pub devices: Vec<UnifiDevice>,
    #[serde(default)]
    pub clients_count: i32,
    #[serde(default)]
    pub guests_count: i32,
    #[serde(default)]
    pub wan_download: f64,
    #[serde(default)]
    pub wan_upload: f64,
}

// --- UNAS ---

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StoragePool {
    pub name: String,
    pub status: String,
    #[serde(default)]
    pub total: i64,
    #[serde(default)]
    pub used: i64,
    #[serde(default)]
    pub available: i64,
    #[serde(default)]
    pub percent: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StorageDisk {
    pub name: String,
    pub model: String,
    pub serial: String,
    pub status: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub temperature: Option<f64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct UNASMetrics {
    #[serde(default)]
    pub pools: Vec<StoragePool>,
    #[serde(default)]
    pub disks: Vec<StorageDisk>,
    #[serde(default)]
    pub total_capacity: i64,
    #[serde(default)]
    pub used_capacity: i64,
}
