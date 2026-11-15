import SwiftUI

extension View {
    /// Mid-risk confirmation dialog and high-risk confirmation sheet for the shared ActionCenter.
    func actionConfirmations() -> some View {
        modifier(ActionConfirmations())
    }
}

private struct ActionConfirmations: ViewModifier {
    @Environment(ActionCenter.self) private var actions

    func body(content: Content) -> some View {
        @Bindable var actions = actions
        content
            .confirmationDialog(
                actions.confirming?.title ?? "",
                isPresented: Binding(
                    get: { actions.confirming != nil },
                    set: { if !$0 { actions.confirming = nil } }
                ),
                titleVisibility: .visible,
                presenting: actions.confirming
            ) { pending in
                Button(pending.title, role: .destructive) { actions.run(pending) }
                Button("Cancel", role: .cancel) {}
            } message: { pending in
                Text(pending.message.isEmpty ? pending.systemName : "\(pending.systemName): \(pending.message)")
            }
            .sheet(item: $actions.authorizing) { pending in
                HighRiskConfirmSheet(pending: pending)
            }
    }
}

/// High-risk confirm: Face ID / Touch ID / passcode, or typing the system id when the device has none.
struct HighRiskConfirmSheet: View {
    let pending: PendingAction
    @Environment(ActionCenter.self) private var actions
    @Environment(\.dismiss) private var dismiss
    @State private var method = OwnerAuthentication.Method.unavailable
    @State private var typedID = ""
    @State private var authFailed = false

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 40))
                .foregroundStyle(.red)
            VStack(spacing: 6) {
                Text("\(pending.title)?")
                    .font(.title3.weight(.semibold))
                    .multilineTextAlignment(.center)
                Text(pending.systemID)
                    .font(.subheadline)
                    .fontDesign(.monospaced)
                    .foregroundStyle(.secondary)
                if !pending.message.isEmpty {
                    Text(pending.message)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }
            }
            if method == .unavailable {
                typeToConfirm
            } else {
                Button {
                    Task { await authenticate() }
                } label: {
                    Label("\(pending.title) with \(method.label)", systemImage: method.symbol)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.glassProminent)
                .controlSize(.large)
                .tint(.red)
                if authFailed {
                    Text("Authentication failed.")
                        .font(.footnote)
                        .foregroundStyle(.red)
                }
            }
            Button("Cancel", role: .cancel) { dismiss() }
                .controlSize(.large)
        }
        .padding(24)
        .presentationDetents([.medium, .large])
        .onAppear { method = OwnerAuthentication.method() }
    }

    private var typeToConfirm: some View {
        VStack(spacing: 10) {
            Text("Type **\(pending.systemID)** to confirm.")
                .font(.callout)
            TextField(pending.systemID, text: $typedID)
                .fontDesign(.monospaced)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .textFieldStyle(.roundedBorder)
                .submitLabel(.go)
                .onSubmit(confirmTyped)
            Button(pending.title, role: .destructive, action: confirmTyped)
                .buttonStyle(.glassProminent)
                .controlSize(.large)
                .tint(.red)
                .disabled(typedID != pending.systemID)
        }
    }

    private func confirmTyped() {
        guard typedID == pending.systemID else { return }
        actions.run(pending)
        dismiss()
    }

    private func authenticate() async {
        if await OwnerAuthentication.authenticate(reason: pending.title) {
            actions.run(pending)
            dismiss()
        } else {
            authFailed = true
        }
    }
}

/// Result of the last action: duration and output, or the server's error text. Tap to dismiss.
struct ActionResultToast: View {
    @Environment(ActionCenter.self) private var actions

    var body: some View {
        if let result = actions.result {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: result.ok ? "checkmark.circle.fill" : "xmark.octagon.fill")
                    .foregroundStyle(result.ok ? .green : .red)
                    .font(.title3)
                VStack(alignment: .leading, spacing: 2) {
                    Text(result.title)
                        .font(.subheadline.weight(.semibold))
                        .lineLimit(2)
                    Text(result.ok ? detail(result) : result.text)
                        .font(.caption)
                        .fontDesign(.monospaced)
                        .foregroundStyle(result.ok ? Color.secondary : Color.red)
                        .lineLimit(4)
                }
                Spacer(minLength: 0)
            }
            .padding(14)
            .glassEffect(.regular, in: .rect(cornerRadius: 20))
            .padding(.horizontal, 16)
            .padding(.bottom, 8)
            .id(result.id)
            .transition(.move(edge: .bottom).combined(with: .opacity))
            .onTapGesture { actions.dismissResult() }
            .accessibilityAddTraits(.isButton)
            .accessibilityHint("Dismisses the result")
        }
    }

    private func detail(_ result: ActionResult) -> String {
        result.text.isEmpty ? "done \(result.durationText)" : "done \(result.durationText) · \(result.text)"
    }
}
