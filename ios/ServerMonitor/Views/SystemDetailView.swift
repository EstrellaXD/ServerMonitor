import SwiftUI

struct SystemDetailView: View {
    let systemID: String
    @Environment(MetricsStore.self) private var store
    @State private var logsFor: LogsTarget?

    var body: some View {
        if let system = store.systems[systemID] {
            List {
                HeaderSection(system: system)
                switch system.metrics {
                case .linux(let m): LinuxSections(system: system, metrics: m)
                case .docker(let m): DockerSections(system: system, metrics: m, logsFor: $logsFor)
                case .qbittorrent(let m): QBittorrentSections(system: system, metrics: m)
                case .unifi(let m): UnifiSections(system: system, metrics: m)
                case .unas(let m): UNASSections(system: system, metrics: m)
                case .none: EmptyView()
                }
                PowerSection(system: system)
            }
            .listStyle(.insetGrouped)
            .navigationTitle(system.name)
            .navigationBarTitleDisplayMode(.inline)
            .refreshable { await store.refresh() }
            .sheet(item: $logsFor) { target in
                DockerLogsSheet(systemID: system.id, container: target.container)
            }
        } else {
            ContentUnavailableView("System Not Found", systemImage: "questionmark.circle",
                                   description: Text("The server no longer reports \(systemID)."))
        }
    }
}

private struct HeaderSection: View {
    let system: SystemMetrics

    private var uptime: Int64? {
        if case .linux(let m) = system.metrics { return m.uptime }
        return nil
    }

    var body: some View {
        Section {
            HStack(spacing: 10) {
                StatusSymbol(status: system.status)
                Text(system.status.rawValue.capitalized)
                Spacer()
                Text("\(system.typeLabel) · \(system.id)")
                    .font(.footnote)
                    .fontDesign(.monospaced)
                    .foregroundStyle(.secondary)
            }
            MetricRow(label: "Updated", value: Format.clock(system.lastUpdated))
            if let uptime {
                MetricRow(label: "Uptime", value: Format.uptime(uptime))
            }
            if let error = system.error {
                Text(error)
                    .font(.footnote)
                    .fontDesign(.monospaced)
                    .foregroundStyle(.red)
            }
        }
    }
}

/// Reboot and Power Off for linux and unas systems with the `power` control.
struct PowerSection: View {
    let system: SystemMetrics
    @Environment(ActionCenter.self) private var actions

    var body: some View {
        if system.can("reboot") {
            Section {
                Button("Reboot", systemImage: "arrow.clockwise.circle", role: .destructive) {
                    actions.request("reboot", on: system, title: "Reboot \(system.name)",
                                    message: "The system restarts and is offline for about a minute.")
                }
                .foregroundStyle(.red)
                Button("Power Off", systemImage: "power", role: .destructive) {
                    actions.request("poweroff", on: system, title: "Power Off \(system.name)",
                                    message: "The system shuts down. Start it again by hand.")
                }
                .foregroundStyle(.red)
            } header: {
                Text("Power")
            } footer: {
                Text("Asks for Face ID or the device passcode before it runs.")
            }
            .disabled(actions.isRunning("reboot", target: "", on: system)
                || actions.isRunning("poweroff", target: "", on: system))
        }
    }
}
