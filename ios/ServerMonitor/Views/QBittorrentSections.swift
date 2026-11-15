import SwiftUI

struct QBittorrentSections: View {
    let system: SystemMetrics
    let metrics: QBittorrentMetrics
    @Environment(ActionCenter.self) private var actions

    private static let actionNames = [
        "resume", "pause", "recheck", "top_prio", "bottom_prio", "increase_prio", "decrease_prio", "delete",
    ]

    var body: some View {
        Section("Transfer") {
            MetricRow(label: "Download", value: "↓ \(Format.speed(metrics.downloadSpeed))",
                      detail: "\(metrics.activeDownloads) active")
            MetricRow(label: "Upload", value: "↑ \(Format.speed(metrics.uploadSpeed))",
                      detail: "\(metrics.activeUploads) active")
            MetricRow(label: "Torrents", value: "\(metrics.totalTorrents)")
        }
        if !metrics.torrents.isEmpty {
            Section {
                ForEach(metrics.torrents, id: \.hash) { t in
                    row(t)
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if system.can("resume") {
                                if isPaused(t.state) {
                                    Button("Resume", systemImage: "play.fill") { request("resume", t, "Resume") }
                                        .tint(.green)
                                } else {
                                    Button("Pause", systemImage: "pause.fill") { request("pause", t, "Pause") }
                                        .tint(.orange)
                                }
                            }
                        }
                        .contextMenu {
                            if system.can("resume") { menu(t) }
                        }
                }
            } header: {
                Text("Torrents")
            } footer: {
                if system.can("resume") { Text("Touch and hold a torrent for actions.") }
            }
        }
    }

    private func row(_ t: TorrentInfo) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline) {
                Text(t.name)
                    .lineLimit(1)
                Spacer(minLength: 8)
                RunningIndicator(isRunning: Self.actionNames.contains {
                    actions.isRunning($0, target: t.hash, on: system)
                })
                Text(t.state)
                    .font(.caption)
                    .fontDesign(.monospaced)
                    .foregroundStyle(stateColor(t.state))
            }
            UsageBar(percent: t.progress, level: .ok)
            Text("\(Format.percent(t.progress)) of \(Format.bytes(t.size)) · ↓\(Format.speed(t.downloadSpeed)) ↑\(Format.speed(t.uploadSpeed)) · eta \(Format.eta(t.eta))")
                .font(.caption)
                .fontDesign(.monospaced)
                .monospacedDigit()
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
    }

    @ViewBuilder private func menu(_ t: TorrentInfo) -> some View {
        Button("Resume", systemImage: "play.fill") { request("resume", t, "Resume") }
        Button("Pause", systemImage: "pause.fill") { request("pause", t, "Pause") }
        Button("Recheck", systemImage: "checkmark.arrow.trianglehead.counterclockwise") { request("recheck", t, "Recheck") }
        Menu("Priority", systemImage: "arrow.up.arrow.down") {
            Button("Top", systemImage: "arrow.up.to.line") { request("top_prio", t, "Top priority for") }
            Button("Up", systemImage: "arrow.up") { request("increase_prio", t, "Raise priority of") }
            Button("Down", systemImage: "arrow.down") { request("decrease_prio", t, "Lower priority of") }
            Button("Bottom", systemImage: "arrow.down.to.line") { request("bottom_prio", t, "Bottom priority for") }
        }
        Divider()
        Button("Delete", systemImage: "trash", role: .destructive) {
            actions.request("delete", target: t.hash, params: ActionParams(deleteFiles: false), on: system,
                            title: "Delete \(t.name)", message: "Removes the torrent. The downloaded files stay.")
        }
        if system.can("delete", params: ActionParams(deleteFiles: true)) {
            Button("Delete with Files", systemImage: "trash.fill", role: .destructive) {
                actions.request("delete", target: t.hash, params: ActionParams(deleteFiles: true), on: system,
                                title: "Delete \(t.name) and its files",
                                message: "Removes the torrent and deletes its downloaded files from disk.")
            }
        }
    }

    private func request(_ action: String, _ t: TorrentInfo, _ verb: String) {
        actions.request(action, target: t.hash, on: system, title: "\(verb) \(t.name)")
    }

    private func isPaused(_ state: String) -> Bool {
        state.hasPrefix("paused") || state.hasPrefix("stopped")
    }

    private func stateColor(_ state: String) -> Color {
        if state == "error" || state == "missingFiles" { return .red }
        if isPaused(state) { return .gray }
        if state.hasPrefix("stalled") || state.hasPrefix("queued") || state.hasPrefix("checking") { return .orange }
        return .green
    }
}
