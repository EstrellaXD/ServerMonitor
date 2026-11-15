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
    @Environment(\.horizontalSizeClass) private var sizeClass
    /// ID of the system in the detail column. An ID, not a value, so live updates keep it.
    @State private var selection: String?
    /// Distance from the leading edge to the vertical fold line; nil when the display has no vertical fold.
    @State private var foldOffset: CGFloat?

    var body: some View {
        if store.needsSetup {
            SetupView()
        } else {
            // Compact width (outer display): collapses to a push stack.
            // Regular width (inner display): list on the leading side, detail on the trailing side.
            NavigationSplitView {
                ServersView(selection: $selection)
                    .toolbar(removing: .sidebarToggle)
                    .modifier(SidebarWidth(width: foldOffset))
            } detail: {
                if let selection {
                    // A new stack per system: a new selection starts at the top, with its own bar state.
                    NavigationStack {
                        SystemDetailView(systemID: selection)
                    }
                    .id(selection)
                } else {
                    ContentUnavailableView("No System Selected", systemImage: "server.rack")
                }
            }
            .navigationSplitViewStyle(.balanced)
            .onGeometryChange(for: CGFloat?.self, of: Self.verticalFoldOffset) { foldOffset = $0 }
            .onChange(of: sizeClass, initial: true) { selectDefaultIfShown() }
            .onChange(of: store.systems.isEmpty) { selectDefaultIfShown() }
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

    /// Fills an empty detail column with the first system that needs attention, else the first system.
    /// Only in regular width: in compact width a selection pushes the detail over the list.
    private func selectDefaultIfShown() {
        guard sizeClass == .regular, selection == nil else { return }
        selection = (store.attention.first ?? store.sortedSystems.first)?.id
    }

    /// The centre line of the vertical fold (`.division`), active or not, so the column boundary stays
    /// on the fold in the book pose and in the same place when the device is open flat.
    nonisolated private static func verticalFoldOffset(_ proxy: GeometryProxy) -> CGFloat? {
        proxy.reservedRegions(kind: .division, options: .includeInactive)
            .first { $0.frame.height > $0.frame.width }?
            .frame.midX
    }
}

/// Sets the sidebar column width when there is a fold to align with; otherwise keeps the system width.
private struct SidebarWidth: ViewModifier {
    let width: CGFloat?

    func body(content: Content) -> some View {
        if let width {
            content.navigationSplitViewColumnWidth(width)
        } else {
            content
        }
    }
}
