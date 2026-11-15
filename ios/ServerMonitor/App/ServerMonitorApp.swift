import SwiftUI

@main
struct ServerMonitorApp: App {
    @State private var store: MetricsStore
    @State private var actions: ActionCenter
    @Environment(\.scenePhase) private var scenePhase

    init() {
        let store = MetricsStore()
        _store = State(initialValue: store)
        _actions = State(initialValue: ActionCenter(store: store))
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(store)
                .environment(actions)
        }
        .onChange(of: scenePhase, initial: true) { _, phase in
            switch phase {
            case .active: store.resume()
            case .background: store.suspend()
            default: break
            }
        }
    }
}

struct RootView: View {
    @Environment(MetricsStore.self) private var store
    @Environment(ActionCenter.self) private var actions

    var body: some View {
        if store.needsSetup {
            SetupView()
        } else {
            NavigationStack {
                ServersView()
                    .navigationDestination(for: String.self) { id in
                        SystemDetailView(systemID: id)
                    }
            }
            .actionConfirmations()
            .overlay(alignment: .bottom) {
                ActionResultToast()
            }
            .animation(.snappy, value: actions.result)
            .sensoryFeedback(trigger: actions.result) { _, result in
                guard let result else { return nil }
                return result.ok ? .success : .error
            }
        }
    }
}
