import SwiftUI

struct ServersView: View {
    @Environment(MetricsStore.self) private var store
    @Binding var selection: String?
    @State private var showsSettings = false

    var body: some View {
        List(selection: $selection) {
            Section {
                SummaryRow()
            }
            if !store.attention.isEmpty {
                Section("Attention") {
                    ForEach(store.attention) { SystemRow(system: $0) }
                }
            }
            if !store.systems.isEmpty {
                Section("All") {
                    ForEach(store.sortedSystems) { SystemRow(system: $0) }
                }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle("Servers")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Settings", systemImage: "gearshape") { showsSettings = true }
            }
        }
        .refreshable { await store.refresh() }
        .overlay {
            if store.systems.isEmpty {
                if store.hasLoaded {
                    ContentUnavailableView(
                        "No Data",
                        systemImage: "server.rack",
                        description: Text(store.loadError ?? "The server reports no systems.")
                    )
                } else {
                    ProgressView("Connecting to \(store.address.hostPort)")
                }
            }
        }
        .sheet(isPresented: $showsSettings) {
            SettingsView()
        }
    }
}

private struct SummaryRow: View {
    @Environment(MetricsStore.self) private var store

    private var counts: String {
        let parts = SystemStatus.allCases.compactMap { status -> String? in
            let n = store.count(status)
            return n > 0 ? "\(n) \(status.rawValue)" : nil
        }
        return parts.isEmpty ? "No systems" : parts.joined(separator: " · ")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(counts)
                .font(.headline)
            HStack(spacing: 6) {
                Circle()
                    .fill(store.connection == .live ? Color.green : Color.gray)
                    .frame(width: 8, height: 8)
                Text("Updated \(Format.clock(store.lastUpdate)) · \(store.connection.label)")
                    .monospacedDigit()
            }
            .font(.caption)
            .fontDesign(.monospaced)
            .foregroundStyle(.secondary)
            if store.connection != .live, let error = store.loadError ?? store.connection.reason {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }
        }
        .accessibilityElement(children: .combine)
    }
}

extension ConnectionState {
    var reason: String? {
        if case .retrying(_, let reason) = self { return reason }
        return nil
    }
}

struct SystemRow: View {
    let system: SystemMetrics

    var body: some View {
        NavigationLink(value: system.id) {
            HStack(spacing: 12) {
                StatusSymbol(status: system.status)
                    .font(.title3)
                VStack(alignment: .leading, spacing: 2) {
                    Text(system.name)
                    Text(system.summary)
                        .font(.caption)
                        .fontDesign(.monospaced)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }
        }
    }
}
