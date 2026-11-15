import SwiftUI

struct LinuxSections: View {
    let system: SystemMetrics
    let metrics: LinuxMetrics

    var body: some View {
        CPUSection(cpu: metrics.cpu)
        Section("Memory") {
            MetricRow(
                label: "Used",
                value: Format.percent(metrics.memory.percent),
                detail: "\(Format.bytes(metrics.memory.used)) of \(Format.bytes(metrics.memory.total)) · \(Format.bytes(metrics.memory.available)) free",
                bar: metrics.memory.percent,
                level: .usage(metrics.memory.percent)
            )
        }
        if !metrics.disks.isEmpty {
            Section("Disks") {
                ForEach(metrics.disks, id: \.mount) { disk in
                    MetricRow(
                        label: disk.mount,
                        value: Format.percent(disk.percent),
                        detail: "\(Format.bytes(disk.used)) of \(Format.bytes(disk.total)) · \(Format.bytes(disk.free)) free",
                        bar: disk.percent,
                        level: .usage(disk.percent)
                    )
                }
            }
        }
        Section("Network") {
            MetricRow(label: "Download", value: "↓ \(Format.speed(metrics.network.downloadRate))",
                      detail: "\(Format.bytes(metrics.network.bytesRecv)) received")
            MetricRow(label: "Upload", value: "↑ \(Format.speed(metrics.network.uploadRate))",
                      detail: "\(Format.bytes(metrics.network.bytesSent)) sent")
        }
        if !metrics.temperatures.isEmpty {
            Section("Temperatures") {
                // lm-sensors repeats labels such as "temp1" across chips: identify rows by position.
                ForEach(Array(metrics.temperatures.enumerated()), id: \.offset) { _, t in
                    MetricRow(label: t.label, value: Format.temperature(t.current, 1),
                              valueColor: Level.temperature(t).color)
                }
            }
        }
        if !metrics.fans.isEmpty {
            FansSection(system: system, fans: metrics.fans)
        }
        ProcessesSection(system: system, processes: metrics.processes)
        if !metrics.services.isEmpty {
            ServicesSection(system: system, services: metrics.services)
        }
    }
}

private struct CPUSection: View {
    let cpu: CPUMetrics

    @State private var gridWidth: CGFloat = 0

    /// Two or four columns (HIG: an even count divides cleanly at a fold), at least 120 pt each.
    private var columns: [GridItem] {
        let count = gridWidth >= 4 * 120 + 3 * 12 ? 4 : 2
        return Array(repeating: GridItem(.flexible(), spacing: 12), count: count)
    }

    var body: some View {
        Section("CPU") {
            MetricRow(
                label: "Total",
                value: Format.percent(cpu.percent),
                detail: cpu.loadAvg.isEmpty ? nil : "load " + cpu.loadAvg.map { Format.fixed($0, 2) }.joined(separator: " "),
                bar: cpu.percent,
                level: .usage(cpu.percent)
            )
            if !cpu.cores.isEmpty {
                LazyVGrid(columns: columns, alignment: .leading, spacing: 8) {
                    ForEach(Array(cpu.cores.enumerated()), id: \.offset) { index, value in
                        VStack(alignment: .leading, spacing: 3) {
                            HStack {
                                Text("c\(index)").foregroundStyle(.secondary)
                                Spacer()
                                Text(Format.percent(value, 0)).monospacedDigit()
                            }
                            .font(.caption)
                            .fontDesign(.monospaced)
                            UsageBar(percent: value, level: .usage(value))
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
                .onGeometryChange(for: CGFloat.self, of: \.size.width) { gridWidth = $0 }
                .padding(.vertical, 4)
            }
        }
    }
}

private struct FansSection: View {
    let system: SystemMetrics
    let fans: [FanMetrics]
    @Environment(ActionCenter.self) private var actions

    private static let steps = [0, 25, 50, 75, 100]

    var body: some View {
        Section("Fans") {
            ForEach(Array(fans.enumerated()), id: \.offset) { _, fan in
                HStack(alignment: .firstTextBaseline) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(fan.label)
                        Text(detail(fan))
                            .font(.caption)
                            .fontDesign(.monospaced)
                            .foregroundStyle(fan.mode == .manual ? Color.orange : Color.secondary)
                    }
                    Spacer()
                    Text("\(String(fan.rpm)) rpm")
                        .fontDesign(.monospaced)
                        .monospacedDigit()
                    RunningIndicator(isRunning: busy(fan))
                    if fan.mode != nil, system.can("fan_set") {
                        controlMenu(fan)
                    }
                }
            }
        }
    }

    private func detail(_ fan: FanMetrics) -> String {
        let pwm = fan.pwmPercent.map { "pwm \(Format.percent($0, 0))" }
        return [pwm, fan.mode?.rawValue].compactMap(\.self).joined(separator: " · ")
    }

    private func busy(_ fan: FanMetrics) -> Bool {
        actions.isRunning("fan_auto", target: fan.label, on: system)
            || actions.isRunning("fan_set", target: fan.label, on: system)
    }

    private func controlMenu(_ fan: FanMetrics) -> some View {
        let current = fan.mode == .manual ? fan.pwmPercent.map { Int($0.rounded()) } : nil
        return Menu {
            Button {
                actions.request("fan_auto", target: fan.label, on: system, title: "Fan \(fan.label) auto")
            } label: {
                if fan.mode == .auto { Label("Auto", systemImage: "checkmark") } else { Text("Auto") }
            }
            Divider()
            ForEach(Self.steps, id: \.self) { step in
                Button {
                    actions.request(
                        "fan_set", target: fan.label, params: ActionParams(percent: step), on: system,
                        title: "Set fan \(fan.label) to \(step)%",
                        message: step == 0
                            ? "The fan stops and the kernel fan governor is off until you choose Auto."
                            : "The kernel fan governor is off until you choose Auto."
                    )
                } label: {
                    if current == step { Label("\(step) %", systemImage: "checkmark") } else { Text("\(step) %") }
                }
            }
        } label: {
            Text("Set…")
        }
        .disabled(busy(fan))
    }
}

private struct ProcessesSection: View {
    let system: SystemMetrics
    let processes: [ProcessMetrics]
    @Environment(ActionCenter.self) private var actions

    var body: some View {
        Section {
            ForEach(processes, id: \.pid) { p in
                row(p)
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                        if system.can("kill") {
                            Button("Kill", systemImage: "xmark.octagon") { kill(p, .kill) }
                                .tint(.red)
                            Button("Terminate", systemImage: "stop.circle") { kill(p, .term) }
                                .tint(.orange)
                        }
                    }
                    .contextMenu {
                        if system.can("kill") {
                            Button("Terminate (TERM)", systemImage: "stop.circle") { kill(p, .term) }
                            Button("Kill (KILL)", systemImage: "xmark.octagon", role: .destructive) { kill(p, .kill) }
                        }
                    }
            }
        } header: {
            Text("Processes")
        } footer: {
            if system.can("kill") { Text("Swipe a process to terminate or kill it.") }
        }
    }

    private func row(_ p: ProcessMetrics) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            Text(String(p.pid))
                .foregroundStyle(.secondary)
                .frame(minWidth: 56, alignment: .leading)
            Text(p.name)
                .lineLimit(1)
            Spacer(minLength: 8)
            RunningIndicator(isRunning: actions.isRunning("kill", target: String(p.pid), on: system))
            VStack(alignment: .trailing, spacing: 1) {
                Text(Format.percent(p.cpuPercent))
                Text(Format.bytes(p.memoryUsage))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .fontDesign(.monospaced)
        .monospacedDigit()
    }

    private func kill(_ p: ProcessMetrics, _ signal: KillSignal) {
        let verb = signal == .kill ? "Kill" : "Terminate"
        actions.request(
            "kill", target: String(p.pid), params: ActionParams(signal: signal), on: system,
            title: "\(verb) \(p.pid) \(p.name)",
            message: signal == .kill
                ? "SIGKILL stops the process at once, without cleanup."
                : "Sends SIGTERM. The process can clean up before it exits."
        )
    }
}

private struct ServicesSection: View {
    let system: SystemMetrics
    let services: [ServiceStatus]
    @Environment(ActionCenter.self) private var actions

    private static let commands = [
        ("service_start", "Start", "play"),
        ("service_stop", "Stop", "stop"),
        ("service_restart", "Restart", "arrow.clockwise"),
    ]

    var body: some View {
        Section("Services") {
            ForEach(services, id: \.name) { s in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(s.name)
                        Text("\(s.activeState) (\(s.subState))")
                            .font(.caption)
                            .fontDesign(.monospaced)
                            .foregroundStyle(color(s.activeState))
                    }
                    Spacer()
                    RunningIndicator(isRunning: Self.commands.contains {
                        actions.isRunning($0.0, target: s.name, on: system)
                    })
                    if system.can("service_restart") {
                        Menu("Actions", systemImage: "ellipsis.circle") {
                            ForEach(Self.commands, id: \.0) { action, verb, symbol in
                                Button(verb, systemImage: symbol) {
                                    actions.request(action, target: s.name, on: system,
                                                    title: "\(verb) \(s.name)",
                                                    message: "systemctl \(verb.lowercased()) \(s.name)")
                                }
                            }
                        }
                        .labelStyle(.iconOnly)
                    }
                }
            }
        }
    }

    private func color(_ state: String) -> Color {
        switch state {
        case "active": .green
        case "failed": .red
        case "activating", "deactivating": .orange
        default: .secondary
        }
    }
}
