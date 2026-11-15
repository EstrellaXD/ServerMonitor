import SwiftUI

struct UnifiSections: View {
    let system: SystemMetrics
    let metrics: UnifiMetrics
    @Environment(ActionCenter.self) private var actions

    var body: some View {
        Section("WAN") {
            MetricRow(label: "Download", value: "↓ \(Format.fixed(metrics.wanDownload, 1)) Mbps")
            MetricRow(label: "Upload", value: "↑ \(Format.fixed(metrics.wanUpload, 1)) Mbps")
            MetricRow(label: "Clients", value: "\(metrics.clientsCount)", detail: "\(metrics.guestsCount) guests")
        }
        if !metrics.devices.isEmpty {
            Section {
                ForEach(metrics.devices, id: \.mac) { d in
                    deviceRow(d)
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if system.can("restart_device") {
                                Button("Restart", systemImage: "arrow.clockwise.circle") { restart(d) }
                                    .tint(.orange)
                            }
                        }
                        .contextMenu {
                            if system.can("restart_device") {
                                Button("Restart", systemImage: "arrow.clockwise.circle", role: .destructive) { restart(d) }
                            }
                        }
                }
            } header: {
                Text("Devices")
            } footer: {
                if system.can("restart_device") { Text("Swipe a device to restart it.") }
            }
        }
        if !metrics.clients.isEmpty {
            Section {
                ForEach(metrics.clients, id: \.mac) { c in
                    clientRow(c, busy: ["kick", "block"])
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if system.can("kick") {
                                Button("Block", systemImage: "hand.raised.fill") {
                                    client("block", c, title: "Block \(label(c))",
                                           message: "The client cannot connect until you unblock it.")
                                }
                                .tint(.red)
                                Button("Kick", systemImage: "wifi.slash") {
                                    client("kick", c, title: "Kick \(label(c))",
                                           message: "Disconnects the client. It can connect again.")
                                }
                                .tint(.orange)
                            }
                        }
                }
            } header: {
                Text("Clients")
            } footer: {
                if system.can("kick") { Text("Swipe a client to kick or block it.") }
            }
        }
        if !metrics.blockedClients.isEmpty {
            Section {
                ForEach(metrics.blockedClients, id: \.mac) { c in
                    clientRow(c, busy: ["unblock"])
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if system.can("unblock") {
                                Button("Unblock", systemImage: "hand.thumbsup.fill") {
                                    client("unblock", c, title: "Unblock \(label(c))")
                                }
                                .tint(.green)
                            }
                        }
                }
            } header: {
                Text("Blocked Clients")
            } footer: {
                if system.can("unblock") { Text("Swipe a client to unblock it.") }
            }
        }
    }

    private func deviceRow(_ d: UnifiDevice) -> some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: 2) {
                Text(d.name)
                Text("\(d.model) · \(d.status)\(d.status == "online" ? " · up \(Format.uptime(d.uptime))" : "")")
                    .font(.caption)
                    .fontDesign(.monospaced)
                    .foregroundStyle(d.status == "online" ? Color.secondary : Color.orange)
            }
            Spacer()
            RunningIndicator(isRunning: actions.isRunning("restart_device", target: d.mac, on: system))
            if let hot = hottest(d.temperatures ?? []) {
                Text(Format.temperature(hot.current))
                    .fontDesign(.monospaced)
                    .foregroundStyle(Level.temperature(hot).color)
            }
        }
    }

    private func clientRow(_ c: UnifiClient, busy: [String]) -> some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    Text(label(c))
                    if c.isGuest {
                        Text("guest")
                            .font(.caption2)
                            .foregroundStyle(.orange)
                    }
                }
                Text([c.ip, c.mac].filter { !$0.isEmpty }.joined(separator: " · "))
                    .font(.caption)
                    .fontDesign(.monospaced)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            RunningIndicator(isRunning: busy.contains { actions.isRunning($0, target: c.mac, on: system) })
        }
    }

    private func label(_ c: UnifiClient) -> String { c.name.isEmpty ? c.mac : c.name }

    private func restart(_ d: UnifiDevice) {
        actions.request("restart_device", target: d.mac, on: system, title: "Restart \(d.name)",
                        message: "The device is offline while it restarts.")
    }

    private func client(_ action: String, _ c: UnifiClient, title: String, message: String = "") {
        actions.request(action, target: c.mac, on: system, title: title, message: message)
    }
}
