import Foundation

// Wire types. Mirror of frontend/src/types/metrics.ts and backend/internal/models.
// Keys are mapped with explicit CodingKeys (no key strategy), so dictionary keys
// such as system ids are never rewritten.

enum SystemStatus: String, Codable, Sendable, CaseIterable {
    case healthy, warning, critical, offline
}

enum SystemType: String, Codable, Sendable, CaseIterable {
    case linux, docker, qbittorrent, unifi, unas
}

struct CPUMetrics: Decodable, Sendable, Hashable {
    var percent: Double
    @EmptyIfNull var cores: [Double]
    @EmptyIfNull var loadAvg: [Double]

    enum CodingKeys: String, CodingKey {
        case percent, cores
        case loadAvg = "load_avg"
    }
}

struct MemoryMetrics: Decodable, Sendable, Hashable {
    var total: Int64
    var used: Int64
    var available: Int64
    var percent: Double
}

struct DiskMetrics: Decodable, Sendable, Hashable {
    var mount: String
    var total: Int64
    var used: Int64
    var free: Int64
    var percent: Double
}

struct NetworkMetrics: Decodable, Sendable, Hashable {
    var bytesSent: Int64
    var bytesRecv: Int64
    var uploadRate: Double
    var downloadRate: Double

    enum CodingKeys: String, CodingKey {
        case bytesSent = "bytes_sent"
        case bytesRecv = "bytes_recv"
        case uploadRate = "upload_rate"
        case downloadRate = "download_rate"
    }
}

struct TemperatureMetrics: Decodable, Sendable, Hashable {
    var label: String
    var current: Double
    var high: Double?
    var critical: Double?
}

struct FanMetrics: Decodable, Sendable, Hashable {
    enum Mode: String, Codable, Sendable {
        case auto, manual
    }

    var label: String
    var rpm: Int64
    var pwmPercent: Double?
    var mode: Mode?

    enum CodingKeys: String, CodingKey {
        case label, rpm, mode
        case pwmPercent = "pwm_percent"
    }
}

struct ProcessMetrics: Decodable, Sendable, Hashable {
    var pid: Int
    var name: String
    var cpuPercent: Double
    var memoryUsage: Int64
    var memoryPercent: Double

    enum CodingKeys: String, CodingKey {
        case pid, name
        case cpuPercent = "cpu_percent"
        case memoryUsage = "memory_usage"
        case memoryPercent = "memory_percent"
    }
}

struct ServiceStatus: Decodable, Sendable, Hashable {
    var name: String
    var activeState: String
    var subState: String

    enum CodingKeys: String, CodingKey {
        case name
        case activeState = "active_state"
        case subState = "sub_state"
    }
}

struct LinuxMetrics: Decodable, Sendable, Hashable {
    var cpu: CPUMetrics
    var memory: MemoryMetrics
    @EmptyIfNull var disks: [DiskMetrics]
    var network: NetworkMetrics
    @EmptyIfNull var temperatures: [TemperatureMetrics]
    @EmptyIfNull var fans: [FanMetrics]
    @EmptyIfNull var processes: [ProcessMetrics]
    @EmptyIfNull var services: [ServiceStatus]
    var uptime: Int64
}

struct ContainerMetrics: Decodable, Sendable, Hashable {
    var id: String
    var name: String
    var image: String
    var status: String
    var state: String
    var cpuPercent: Double
    var memoryUsage: Int64
    var memoryLimit: Int64
    var memoryPercent: Double

    enum CodingKeys: String, CodingKey {
        case id, name, image, status, state
        case cpuPercent = "cpu_percent"
        case memoryUsage = "memory_usage"
        case memoryLimit = "memory_limit"
        case memoryPercent = "memory_percent"
    }
}

struct DockerMetrics: Decodable, Sendable, Hashable {
    @EmptyIfNull var containers: [ContainerMetrics]
    var runningCount: Int
    var stoppedCount: Int
    var totalCount: Int

    enum CodingKeys: String, CodingKey {
        case containers
        case runningCount = "running_count"
        case stoppedCount = "stopped_count"
        case totalCount = "total_count"
    }
}

struct TorrentInfo: Decodable, Sendable, Hashable {
    var hash: String
    var name: String
    var size: Int64
    /// Percent, 0–100.
    var progress: Double
    var downloadSpeed: Int64
    var uploadSpeed: Int64
    var state: String
    var eta: Int64?

    enum CodingKeys: String, CodingKey {
        case hash, name, size, progress, state, eta
        case downloadSpeed = "download_speed"
        case uploadSpeed = "upload_speed"
    }
}

struct QBittorrentMetrics: Decodable, Sendable, Hashable {
    var downloadSpeed: Int64
    var uploadSpeed: Int64
    var activeDownloads: Int
    var activeUploads: Int
    var totalTorrents: Int
    @EmptyIfNull var torrents: [TorrentInfo]

    enum CodingKeys: String, CodingKey {
        case torrents
        case downloadSpeed = "download_speed"
        case uploadSpeed = "upload_speed"
        case activeDownloads = "active_downloads"
        case activeUploads = "active_uploads"
        case totalTorrents = "total_torrents"
    }
}

struct UnifiDevice: Decodable, Sendable, Hashable {
    var name: String
    var mac: String
    var model: String
    var status: String
    var uptime: Int64
    /// Only gateways report sensors; null or empty for other devices.
    var temperatures: [TemperatureMetrics]?
}

struct UnifiClient: Decodable, Sendable, Hashable {
    var mac: String
    var name: String
    var ip: String
    var isGuest: Bool

    enum CodingKeys: String, CodingKey {
        case mac, name, ip
        case isGuest = "is_guest"
    }
}

struct UnifiMetrics: Decodable, Sendable, Hashable {
    @EmptyIfNull var devices: [UnifiDevice]
    @EmptyIfNull var clients: [UnifiClient]
    @EmptyIfNull var blockedClients: [UnifiClient]
    var clientsCount: Int
    var guestsCount: Int
    /// Mbps.
    var wanDownload: Double
    /// Mbps.
    var wanUpload: Double

    enum CodingKeys: String, CodingKey {
        case devices, clients
        case blockedClients = "blocked_clients"
        case clientsCount = "clients_count"
        case guestsCount = "guests_count"
        case wanDownload = "wan_download"
        case wanUpload = "wan_upload"
    }
}

struct StoragePool: Decodable, Sendable, Hashable {
    var name: String
    var status: String
    var total: Int64
    var used: Int64
    var available: Int64
    var percent: Double
}

struct StorageDisk: Decodable, Sendable, Hashable {
    var name: String
    var model: String
    var serial: String
    var status: String
    var temperature: Double?
}

struct UNASMetrics: Decodable, Sendable, Hashable {
    @EmptyIfNull var pools: [StoragePool]
    @EmptyIfNull var disks: [StorageDisk]
    var totalCapacity: Int64
    var usedCapacity: Int64

    enum CodingKeys: String, CodingKey {
        case pools, disks
        case totalCapacity = "total_capacity"
        case usedCapacity = "used_capacity"
    }
}

/// `SystemMetrics.metrics`, discriminated by `SystemMetrics.type`.
/// An offline system sends `metrics: {}`, which decodes as `.none`.
enum MetricsPayload: Sendable, Hashable {
    case linux(LinuxMetrics)
    case docker(DockerMetrics)
    case qbittorrent(QBittorrentMetrics)
    case unifi(UnifiMetrics)
    case unas(UNASMetrics)
    case none
}

struct SystemMetrics: Decodable, Sendable, Hashable, Identifiable {
    var id: String
    var name: String
    var type: SystemType
    var status: SystemStatus
    var lastUpdated: Date
    var error: String?
    @EmptyIfNull var controls: [String]
    var metrics: MetricsPayload

    enum CodingKeys: String, CodingKey {
        case id, name, type, status, error, controls, metrics
        case lastUpdated = "last_updated"
    }

    /// Key that marks a filled payload, the same test as the guards in frontend/src/lib/systems.ts.
    private static let marker: [SystemType: String] = [
        .linux: "cpu", .docker: "containers", .qbittorrent: "torrents", .unifi: "devices", .unas: "pools",
    ]

    init(
        id: String, name: String, type: SystemType, status: SystemStatus, lastUpdated: Date,
        error: String? = nil, controls: [String] = [], metrics: MetricsPayload
    ) {
        self.id = id
        self.name = name
        self.type = type
        self.status = status
        self.lastUpdated = lastUpdated
        self.error = error
        self.controls = controls
        self.metrics = metrics
    }

    init(from decoder: any Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        name = try c.decode(String.self, forKey: .name)
        type = try c.decode(SystemType.self, forKey: .type)
        status = try c.decode(SystemStatus.self, forKey: .status)
        lastUpdated = try c.decode(Date.self, forKey: .lastUpdated)
        error = try c.decodeIfPresent(String.self, forKey: .error)
        _controls = try c.decode(EmptyIfNull<String>.self, forKey: .controls)

        let raw = try c.nestedContainer(keyedBy: AnyKey.self, forKey: .metrics)
        guard let key = Self.marker[type], raw.contains(AnyKey(key)) else {
            metrics = .none
            return
        }
        switch type {
        case .linux: metrics = .linux(try c.decode(LinuxMetrics.self, forKey: .metrics))
        case .docker: metrics = .docker(try c.decode(DockerMetrics.self, forKey: .metrics))
        case .qbittorrent: metrics = .qbittorrent(try c.decode(QBittorrentMetrics.self, forKey: .metrics))
        case .unifi: metrics = .unifi(try c.decode(UnifiMetrics.self, forKey: .metrics))
        case .unas: metrics = .unas(try c.decode(UNASMetrics.self, forKey: .metrics))
        }
    }

}

struct MetricsUpdate: Decodable, Sendable, Hashable {
    var type: String
    var timestamp: Date
    var systems: [String: SystemMetrics]
}
