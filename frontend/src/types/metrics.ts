export type SystemStatus = 'healthy' | 'warning' | 'critical' | 'offline'
export type SystemType = 'linux' | 'docker' | 'qbittorrent' | 'unifi' | 'unas'

export interface CpuMetrics {
  percent: number
  cores: number[]
  load_avg: number[]
}

export interface MemoryMetrics {
  total: number
  used: number
  available: number
  percent: number
}

export interface DiskMetrics {
  mount: string
  total: number
  used: number
  free: number
  percent: number
}

export interface NetworkMetrics {
  bytes_sent: number
  bytes_recv: number
  upload_rate: number
  download_rate: number
}

export interface TemperatureMetrics {
  label: string
  current: number
  high?: number
  critical?: number
}

export interface LinuxMetrics {
  cpu: CpuMetrics
  memory: MemoryMetrics
  disks: DiskMetrics[]
  network: NetworkMetrics
  temperatures: TemperatureMetrics[]
  uptime: number
}

export interface ContainerMetrics {
  id: string
  name: string
  image: string
  status: string
  state: string
  cpu_percent: number
  memory_usage: number
  memory_limit: number
  memory_percent: number
}

export interface DockerMetrics {
  containers: ContainerMetrics[]
  running_count: number
  stopped_count: number
  total_count: number
}

export interface TorrentInfo {
  name: string
  size: number
  progress: number
  download_speed: number
  upload_speed: number
  state: string
  eta?: number
}

export interface QBittorrentMetrics {
  download_speed: number
  upload_speed: number
  active_downloads: number
  active_uploads: number
  total_torrents: number
  torrents: TorrentInfo[]
}

export interface UnifiDevice {
  name: string
  mac: string
  model: string
  status: string
  uptime: number
}

export interface UnifiMetrics {
  devices: UnifiDevice[]
  clients_count: number
  guests_count: number
  wan_download: number
  wan_upload: number
}

export interface StoragePool {
  name: string
  status: string
  total: number
  used: number
  available: number
  percent: number
}

export interface StorageDisk {
  name: string
  model: string
  serial: string
  status: string
  temperature?: number
}

export interface UNASMetrics {
  pools: StoragePool[]
  disks: StorageDisk[]
  total_capacity: number
  used_capacity: number
}

export type MetricsData = LinuxMetrics | DockerMetrics | QBittorrentMetrics | UnifiMetrics | UNASMetrics

export interface SystemMetrics {
  id: string
  name: string
  type: SystemType
  status: SystemStatus
  last_updated: string
  error?: string
  metrics: MetricsData
}

export interface MetricsUpdate {
  type: string
  timestamp: string
  systems: Record<string, SystemMetrics>
}
