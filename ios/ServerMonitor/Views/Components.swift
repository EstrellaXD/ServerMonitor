import SwiftUI

extension SystemStatus {
    var symbol: String {
        switch self {
        case .healthy: "checkmark.circle.fill"
        case .warning: "exclamationmark.triangle.fill"
        case .critical: "xmark.octagon.fill"
        case .offline: "circle.dashed"
        }
    }

    var color: Color {
        switch self {
        case .healthy: .green
        case .warning: .orange
        case .critical: .red
        case .offline: .gray
        }
    }
}

extension Level {
    var color: Color {
        switch self {
        case .ok: .green
        case .warn: .orange
        case .crit: .red
        }
    }
}

struct StatusSymbol: View {
    let status: SystemStatus

    var body: some View {
        Image(systemName: status.symbol)
            .foregroundStyle(status.color)
            .accessibilityLabel(status.rawValue)
    }
}

/// Linear bar for a 0–100 value, colored by level.
struct UsageBar: View {
    let percent: Double
    var level: Level = .ok

    var body: some View {
        ProgressView(value: min(max(percent, 0), 100), total: 100)
            .tint(level.color)
            .accessibilityHidden(true)
    }
}

/// Label on the left, monospaced value on the right, optional bar below.
struct MetricRow: View {
    let label: String
    let value: String
    var detail: String?
    var bar: Double?
    var level: Level = .ok
    var valueColor: Color?

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline) {
                Text(label)
                Spacer(minLength: 12)
                Text(value)
                    .monospacedDigit()
                    .fontDesign(.monospaced)
                    .foregroundStyle(valueColor ?? .primary)
            }
            if let bar {
                UsageBar(percent: bar, level: level)
            }
            if let detail {
                Text(detail)
                    .font(.caption)
                    .fontDesign(.monospaced)
                    .foregroundStyle(.secondary)
            }
        }
        .accessibilityElement(children: .combine)
    }
}

/// Small spinner shown in a row while its action runs.
struct RunningIndicator: View {
    let isRunning: Bool

    var body: some View {
        if isRunning {
            ProgressView().controlSize(.small)
        }
    }
}

/// "3 rows" style count for a section footer.
func countLabel(_ n: Int, _ noun: String) -> String {
    "\(n) \(noun)\(n == 1 ? "" : "s")"
}
