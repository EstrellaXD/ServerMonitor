import SwiftUI

/// Address field with "Test connection". Shared by Settings and the first-launch setup.
struct AddressForm: View {
    @Environment(MetricsStore.self) private var store
    @State private var text = ""
    @State private var test: TestState = .idle

    /// Called after the address is saved.
    var onSave: () -> Void = {}

    private enum TestState: Equatable {
        case idle, running
        case ok(Int)
        case failed(String)
    }

    private var parsed: ServerAddress? { ServerAddress(text) }

    var body: some View {
        Section {
            TextField("host:port", text: $text)
                .fontDesign(.monospaced)
                .keyboardType(.URL)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .submitLabel(.done)
                .onSubmit(save)
                .onChange(of: text) { test = .idle }
            Button {
                Task { await runTest() }
            } label: {
                HStack {
                    Text("Test Connection")
                    Spacer()
                    testStatus
                }
            }
            .disabled(parsed == nil || test == .running)
            if case .failed(let message) = test {
                Text(message).font(.footnote).foregroundStyle(.red)
            }
            Button("Save", action: save)
                .disabled(parsed == nil || parsed == store.address && store.hasSavedAddress)
        } header: {
            Text("Server")
        } footer: {
            Text("The ServerMonitor backend on your local network, for example \(ServerAddress.defaultValue) or 192.168.1.10:8742.")
        }
        .onAppear { text = store.address.hostPort }
    }

    @ViewBuilder private var testStatus: some View {
        switch test {
        case .idle:
            EmptyView()
        case .running:
            ProgressView()
        case .ok(let n):
            Label(countLabel(n, "system"), systemImage: "checkmark.circle.fill")
                .foregroundStyle(.green)
                .font(.footnote.monospaced())
        case .failed:
            Image(systemName: "xmark.circle.fill").foregroundStyle(.red)
        }
    }

    private func runTest() async {
        guard let address = parsed else { return }
        test = .running
        do {
            let systems = try await APIClient(address: address).systems()
            test = .ok(systems.count)
        } catch {
            test = .failed(error.localizedDescription)
        }
    }

    private func save() {
        guard let address = parsed else { return }
        store.setAddress(address)
        onSave()
    }
}

struct SettingsView: View {
    @Environment(\.dismiss) private var dismiss

    private var version: String {
        let info = Bundle.main.infoDictionary
        let short = info?["CFBundleShortVersionString"] as? String ?? "?"
        let build = info?["CFBundleVersion"] as? String ?? "?"
        return "\(short) (\(build))"
    }

    var body: some View {
        NavigationStack {
            Form {
                AddressForm(onSave: { dismiss() })
                Section("About") {
                    LabeledContent("Version") {
                        Text(version).fontDesign(.monospaced)
                    }
                }
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done", systemImage: "checkmark") { dismiss() }
                }
            }
        }
    }
}

/// First launch with no reachable server.
struct SetupView: View {
    @Environment(MetricsStore.self) private var store

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    VStack(alignment: .leading, spacing: 8) {
                        Image(systemName: "server.rack")
                            .font(.largeTitle)
                            .foregroundStyle(.tint)
                        Text("No server found at \(store.address.hostPort).")
                            .font(.headline)
                        if let error = store.loadError {
                            Text(error)
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .padding(.vertical, 4)
                }
                AddressForm()
            }
            .navigationTitle("Set Up")
        }
    }
}
