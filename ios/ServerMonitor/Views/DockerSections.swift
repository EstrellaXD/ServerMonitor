import SwiftUI

struct DockerSections: View {
    let system: SystemMetrics
    let metrics: DockerMetrics
    @Environment(ActionCenter.self) private var actions
    @Binding var logsFor: LogsTarget?

    private static let actionNames = ["start", "stop", "restart", "pause", "unpause", "kill", "recreate"]

    var body: some View {
        Section {
            MetricRow(label: "Running", value: "\(metrics.runningCount) / \(metrics.totalCount)",
                      detail: "\(metrics.stoppedCount) stopped")
        }
        Section {
            ForEach(metrics.containers, id: \.id) { c in
                Button {
                    logsFor = LogsTarget(container: c.name)
                } label: {
                    row(c)
                }
                .foregroundStyle(.primary)
                .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                    if system.can("start") { quickActions(c) }
                }
                .swipeActions(edge: .leading) {
                    Button("Logs", systemImage: "text.alignleft") { logsFor = LogsTarget(container: c.name) }
                        .tint(.blue)
                }
                .contextMenu {
                    Button("Logs", systemImage: "text.alignleft") { logsFor = LogsTarget(container: c.name) }
                    if system.can("start") {
                        Divider()
                        menuActions(c)
                    }
                    if system.can("recreate") {
                        Divider()
                        Button("Recreate", systemImage: "arrow.triangle.2.circlepath", role: .destructive) {
                            request("recreate", c, title: "Recreate \(c.name)",
                                    message: "Pulls \(c.image) and replaces the container with the same configuration.")
                        }
                    }
                }
            }
        } header: {
            Text("Containers")
        } footer: {
            Text("Tap a container for its logs. Touch and hold for actions.")
        }
    }

    private func row(_ c: ContainerMetrics) -> some View {
        HStack(spacing: 10) {
            Image(systemName: "shippingbox.fill")
                .foregroundStyle(stateColor(c.state))
            VStack(alignment: .leading, spacing: 2) {
                Text(c.name)
                Text("\(c.state) · \(c.image)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            Spacer(minLength: 8)
            RunningIndicator(isRunning: Self.actionNames.contains {
                actions.isRunning($0, target: c.name, on: system)
            })
            if c.state == "running" {
                VStack(alignment: .trailing, spacing: 1) {
                    Text(Format.percent(c.cpuPercent))
                    Text(Format.bytes(c.memoryUsage))
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                .fontDesign(.monospaced)
                .monospacedDigit()
            }
        }
    }

    @ViewBuilder private func quickActions(_ c: ContainerMetrics) -> some View {
        switch c.state {
        case "running":
            Button("Stop", systemImage: "stop.fill") { request("stop", c, title: "Stop \(c.name)") }
                .tint(.red)
            Button("Restart", systemImage: "arrow.clockwise") { request("restart", c, title: "Restart \(c.name)") }
                .tint(.orange)
        case "paused":
            Button("Unpause", systemImage: "play.fill") { request("unpause", c, title: "Unpause \(c.name)") }
                .tint(.green)
        default:
            Button("Start", systemImage: "play.fill") { request("start", c, title: "Start \(c.name)") }
                .tint(.green)
        }
    }

    @ViewBuilder private func menuActions(_ c: ContainerMetrics) -> some View {
        switch c.state {
        case "running":
            Button("Restart", systemImage: "arrow.clockwise") { request("restart", c, title: "Restart \(c.name)") }
            Button("Stop", systemImage: "stop.fill") { request("stop", c, title: "Stop \(c.name)") }
            Button("Pause", systemImage: "pause.fill") { request("pause", c, title: "Pause \(c.name)") }
            Button("Kill", systemImage: "xmark.octagon", role: .destructive) {
                request("kill", c, title: "Kill \(c.name)", message: "Sends SIGKILL to the container.")
            }
        case "paused":
            Button("Unpause", systemImage: "play.fill") { request("unpause", c, title: "Unpause \(c.name)") }
            Button("Stop", systemImage: "stop.fill") { request("stop", c, title: "Stop \(c.name)") }
        default:
            Button("Start", systemImage: "play.fill") { request("start", c, title: "Start \(c.name)") }
        }
    }

    private func request(_ action: String, _ c: ContainerMetrics, title: String, message: String = "") {
        actions.request(action, target: c.name, on: system, title: title, message: message)
    }

    private func stateColor(_ state: String) -> Color {
        switch state {
        case "running": .green
        case "paused", "restarting": .orange
        case "dead": .red
        default: .gray
        }
    }
}

struct LogsTarget: Identifiable {
    let container: String
    var id: String { container }
}

/// Snapshot of `docker logs --tail 200`, selectable, stderr in red.
struct DockerLogsSheet: View {
    let systemID: String
    let container: String
    @Environment(MetricsStore.self) private var store
    @Environment(\.dismiss) private var dismiss
    @State private var logs: DockerLogs?
    @State private var error: String?
    @State private var isLoading = false

    static let tail = 200

    var body: some View {
        NavigationStack {
            content
                .navigationTitle(container)
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Done", systemImage: "xmark") { dismiss() }
                    }
                    ToolbarItem(placement: .primaryAction) {
                        Button("Refresh", systemImage: "arrow.clockwise") { Task { await load() } }
                            .disabled(isLoading)
                    }
                }
                .task { await load() }
        }
    }

    @ViewBuilder private var content: some View {
        if let error, logs == nil {
            ContentUnavailableView("No Logs", systemImage: "exclamationmark.triangle", description: Text(error))
        } else if let logs {
            if logs.lines.isEmpty {
                ContentUnavailableView("No Output", systemImage: "text.alignleft",
                                       description: Text("The container wrote no log lines."))
            } else {
                // Vertical only: long lines wrap. The container margins keep the text clear of the rail.
                ScrollView {
                    Text(Self.attributed(logs.lines))
                        .font(.caption)
                        .fontDesign(.monospaced)
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .contentMargins(for: .container, edges: .horizontal)
                        .padding(.vertical)
                }
                .defaultScrollAnchor(.bottomLeading)
                .safeAreaInset(edge: .bottom) {
                    Text(error ?? "last \(logs.lines.count) lines")
                        .font(.caption2)
                        .fontDesign(.monospaced)
                        .foregroundStyle(error == nil ? Color.secondary : Color.red)
                        .padding(.bottom, 4)
                }
            }
        } else {
            ProgressView()
        }
    }

    static func attributed(_ lines: [DockerLogLine]) -> AttributedString {
        var out = AttributedString()
        for (i, line) in lines.enumerated() {
            var part = AttributedString(line.text + (i == lines.count - 1 ? "" : "\n"))
            if line.stream == .stderr { part.foregroundColor = .red }
            out += part
        }
        return out
    }

    private func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            logs = try await store.client.dockerLogs(systemID: systemID, container: container, tail: Self.tail)
            error = nil
        } catch {
            self.error = error.localizedDescription
        }
    }
}
