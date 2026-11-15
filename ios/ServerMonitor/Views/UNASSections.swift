import SwiftUI

struct UNASSections: View {
    let system: SystemMetrics
    let metrics: UNASMetrics

    private var usedPercent: Double {
        metrics.totalCapacity > 0 ? Double(metrics.usedCapacity) / Double(metrics.totalCapacity) * 100 : 0
    }

    // Same set as the web UI (UnasDetail.vue).
    private static func isHealthy(_ status: String) -> Bool {
        ["healthy", "online", "normal", "ok"].contains(status.lowercased())
    }

    var body: some View {
        Section("Capacity") {
            MetricRow(label: "Used", value: Format.percent(usedPercent),
                      detail: "\(Format.bytes(metrics.usedCapacity)) of \(Format.bytes(metrics.totalCapacity))",
                      bar: usedPercent, level: .usage(usedPercent))
        }
        if !metrics.pools.isEmpty {
            Section("Pools") {
                ForEach(metrics.pools, id: \.name) { p in
                    MetricRow(label: p.name, value: Format.percent(p.percent),
                              detail: "\(p.status) · \(Format.bytes(p.used)) of \(Format.bytes(p.total)) · \(Format.bytes(p.available)) free",
                              bar: p.percent, level: .usage(p.percent))
                }
            }
        }
        if !metrics.disks.isEmpty {
            Section("Disks") {
                ForEach(metrics.disks, id: \.name) { d in
                    HStack(alignment: .firstTextBaseline) {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(d.name)
                                .fontDesign(.monospaced)
                            Text("\(d.model) · \(d.serial)")
                                .font(.caption)
                                .fontDesign(.monospaced)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        VStack(alignment: .trailing, spacing: 2) {
                            Text(d.status)
                                .font(.caption)
                                .foregroundStyle(Self.isHealthy(d.status) ? Color.green : Color.orange)
                            if let t = d.temperature {
                                Text(Format.temperature(t))
                                    .fontDesign(.monospaced)
                                    .foregroundStyle(Level.temperature(TemperatureMetrics(label: d.name, current: t, high: 45, critical: 55)).color)
                            }
                        }
                    }
                }
            }
        }
    }
}
