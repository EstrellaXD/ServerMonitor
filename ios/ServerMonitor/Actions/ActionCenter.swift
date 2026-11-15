import Foundation
import Observation

/// An action the user asked for, waiting for its confirm step.
struct PendingAction: Identifiable, Hashable, Sendable {
    let id = UUID()
    let systemID: String
    let systemName: String
    let body: ActionRequestBody
    let risk: Risk
    /// Button title and result label, for example "Reboot" or "Kill 1351 traefik".
    let title: String
    /// What happens, shown in the confirm step.
    let message: String

    var key: ActionKey { ActionKey(systemID: systemID, action: body.action, target: body.target, params: body.params) }
}

/// One in-flight request. Params are part of the key, so Kill can follow a hung Terminate on the same PID.
struct ActionKey: Hashable, Sendable {
    let systemID: String
    let action: String
    let target: String
    let params: ActionParams?
}

struct ActionResult: Identifiable, Equatable, Sendable {
    let id = UUID()
    let title: String
    let ok: Bool
    /// Server output on success, the server's error text on failure.
    let text: String
    let duration: Duration

    var durationText: String {
        let ms = Double(duration.components.seconds) * 1000 + Double(duration.components.attoseconds) / 1e15
        return ms < 1000 ? "\(Int(ms.rounded())) ms" : String(format: "%.1f s", ms / 1000)
    }
}

/// Runs actions through their risk tier: low at once, mid after a dialog, high after a sheet with authentication.
@MainActor
@Observable
final class ActionCenter {
    /// Mid-risk action waiting for the confirmation dialog.
    var confirming: PendingAction?
    /// High-risk action waiting for the confirmation sheet.
    var authorizing: PendingAction?
    private(set) var result: ActionResult?
    private(set) var running: Set<ActionKey> = []

    private let store: MetricsStore
    @ObservationIgnored private var clearTask: Task<Void, Never>?

    init(store: MetricsStore) {
        self.store = store
    }

    func isRunning(_ action: String, target: String, on system: SystemMetrics) -> Bool {
        running.contains { $0.systemID == system.id && $0.action == action && $0.target == target }
    }

    func request(
        _ action: String, target: String = "", params: ActionParams? = nil, on system: SystemMetrics,
        title: String, message: String = ""
    ) {
        let pending = PendingAction(
            systemID: system.id,
            systemName: system.name,
            body: ActionRequestBody(action: action, target: target, params: params),
            risk: ActionCatalog.risk(of: action, params: params, type: system.type),
            title: title,
            message: message
        )
        switch pending.risk {
        case .low: run(pending)
        case .mid: confirming = pending
        case .high: authorizing = pending
        }
    }

    func run(_ pending: PendingAction) {
        let key = pending.key
        guard !running.contains(key) else { return }
        running.insert(key)
        let client = store.client
        Task {
            let clock = ContinuousClock()
            let start = clock.now
            let ok: Bool
            let text: String
            do {
                let response = try await client.run(pending.body, on: pending.systemID)
                ok = true
                text = response.output?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            } catch {
                ok = false
                text = error.localizedDescription
            }
            running.remove(key)
            show(ActionResult(title: pending.title, ok: ok, text: text, duration: clock.now - start))
            if ok { await store.refresh() }
        }
    }

    func dismissResult() {
        clearTask?.cancel()
        result = nil
    }

    private func show(_ newResult: ActionResult) {
        result = newResult
        clearTask?.cancel()
        clearTask = Task { [weak self] in
            try? await Task.sleep(for: .seconds(newResult.ok ? 4 : 10))
            guard !Task.isCancelled else { return }
            self?.result = nil
        }
    }
}
