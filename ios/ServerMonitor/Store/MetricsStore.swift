import Foundation
import Observation

enum ConnectionState: Equatable, Sendable {
    case idle
    case connecting
    case live
    case retrying(after: Int, reason: String)

    var label: String {
        switch self {
        case .idle: "paused"
        case .connecting: "connecting"
        case .live: "live"
        case .retrying(let seconds, _): "retry in \(seconds) s"
        }
    }
}

/// App-wide state: the server address, the latest metrics, and the WebSocket life cycle.
@MainActor
@Observable
final class MetricsStore {
    static let addressKey = "serverAddress"
    /// Seconds to wait before reconnect attempt 1, 2, 3, 4 and later.
    static let backoff = [1, 2, 5, 10]

    private(set) var address: ServerAddress
    /// False until the user saved an address or the app reached a server once.
    private(set) var hasSavedAddress: Bool
    private(set) var systems: [String: SystemMetrics] = [:]
    private(set) var lastUpdate: Date?
    private(set) var connection: ConnectionState = .idle
    /// Error of the last GET /api/systems, nil after a success.
    private(set) var loadError: String?
    private(set) var hasLoaded = false

    private let defaults: UserDefaults
    @ObservationIgnored private var streamTask: Task<Void, Never>?

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        let saved = defaults.string(forKey: Self.addressKey).flatMap(ServerAddress.init)
        address = saved ?? ServerAddress(ServerAddress.defaultValue)!
        hasSavedAddress = saved != nil
    }

    var client: APIClient { APIClient(address: address) }

    /// First launch with no reachable server: show the setup screen.
    var needsSetup: Bool { !hasSavedAddress && hasLoaded && loadError != nil && systems.isEmpty }

    // MARK: Life cycle

    /// Foreground: one GET, then the WebSocket.
    func resume() {
        guard streamTask == nil else { return }
        streamTask = Task { [weak self] in
            await self?.refresh()
            await self?.streamLoop()
        }
    }

    /// Background: close the WebSocket.
    func suspend() {
        streamTask?.cancel()
        streamTask = nil
        connection = .idle
    }

    func setAddress(_ newAddress: ServerAddress) {
        defaults.set(newAddress.hostPort, forKey: Self.addressKey)
        hasSavedAddress = true
        guard newAddress != address else {
            Task { await refresh() }
            return
        }
        address = newAddress
        systems = [:]
        lastUpdate = nil
        loadError = nil
        hasLoaded = false
        suspend()
        resume()
    }

    func refresh() async {
        do {
            let fresh = try await client.systems()
            // suspend() or setAddress() cancelled this task: drop the stale result.
            guard !Task.isCancelled else { return }
            apply(fresh, at: Date())
            loadError = nil
            rememberAddress()
        } catch {
            // URLSession throws URLError(.cancelled), not CancellationError, when the task is cancelled.
            guard !Task.isCancelled else { return }
            loadError = error.localizedDescription
        }
        hasLoaded = true
    }

    private func streamLoop() async {
        var attempt = 0
        while !Task.isCancelled {
            connection = .connecting
            var reason = "connection closed"
            do {
                for try await update in MetricsSocket(url: address.webSocketURL).updates() {
                    attempt = 0
                    connection = .live
                    apply(update.systems, at: update.timestamp)
                    rememberAddress()
                }
            } catch {
                reason = error.localizedDescription
            }
            guard !Task.isCancelled else { return }
            let delay = Self.backoff[min(attempt, Self.backoff.count - 1)]
            attempt += 1
            connection = .retrying(after: delay, reason: reason)
            try? await Task.sleep(for: .seconds(delay))
        }
    }

    private func apply(_ fresh: [String: SystemMetrics], at date: Date) {
        systems = fresh
        lastUpdate = date
        hasLoaded = true
    }

    private func rememberAddress() {
        guard !hasSavedAddress else { return }
        defaults.set(address.hostPort, forKey: Self.addressKey)
        hasSavedAddress = true
    }

    // MARK: Derived lists

    var sortedSystems: [SystemMetrics] {
        systems.values.sorted { $0.name.localizedStandardCompare($1.name) == .orderedAscending }
    }

    /// Systems that are not healthy: critical, warning, offline, then by name.
    var attention: [SystemMetrics] {
        sortedSystems.filter { $0.status != .healthy }.enumerated()
            .sorted { ($0.element.status.severity, $1.offset) > ($1.element.status.severity, $0.offset) }
            .map(\.element)
    }

    func count(_ status: SystemStatus) -> Int {
        systems.values.count(where: { $0.status == status })
    }
}

extension SystemStatus {
    /// Sort rank for the Attention section.
    var severity: Int {
        switch self {
        case .critical: 3
        case .warning: 2
        case .offline: 1
        case .healthy: 0
        }
    }
}
