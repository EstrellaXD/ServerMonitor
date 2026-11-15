import Foundation

/// Port of frontend/src/lib/systems.ts.
extension SystemMetrics {
    var typeLabel: String {
        switch type {
        case .linux: "linux"
        case .docker: "docker"
        case .qbittorrent: "torrent"
        case .unifi: "router"
        case .unas: "storage"
        }
    }

    func can(_ action: String, params: ActionParams? = nil) -> Bool {
        ActionCatalog.canRun(action, params: params, on: self)
    }

    /// One-line summary, as in the web `status --all` table.
    var summary: String {
        if status == .offline { return error ?? "offline" }
        switch metrics {
        case .linux(let m):
            let up = Format.uptime(m.uptime).split(separator: " ").first.map(String.init) ?? ""
            return "cpu \(Format.fixed(m.cpu.percent, 0))% mem \(Format.fixed(m.memory.percent, 0))% up \(up)"
        case .docker(let m):
            return "\(m.runningCount)/\(m.totalCount) running"
        case .qbittorrent(let m):
            return "↓\(Format.speed(m.downloadSpeed)) ↑\(Format.speed(m.uploadSpeed)) \(m.activeDownloads + m.activeUploads) active"
        case .unifi(let m):
            let online = m.devices.count(where: { $0.status == "online" })
            let hot = hottest(m.devices.flatMap { $0.temperatures ?? [] })
            let max = hot.map { ", max \(Format.fixed($0.current, 0))°C" } ?? ""
            return "\(m.clientsCount) clients, \(online)/\(m.devices.count) devices up\(max)"
        case .unas(let m):
            let pct = m.totalCapacity > 0 ? Double(m.usedCapacity) / Double(m.totalCapacity) * 100 : 0
            let pools = m.pools.count == 1 ? "pool" : "pools"
            return "\(m.pools.count) \(pools) \(Format.fixed(pct, 0))% of \(Format.bytes(m.totalCapacity))"
        case .none:
            return error ?? ""
        }
    }
}

func hottest(_ temps: [TemperatureMetrics]) -> TemperatureMetrics? {
    temps.max { $0.current < $1.current }
}
