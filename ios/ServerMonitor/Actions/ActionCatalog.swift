import Foundation

/// Confirm tier of an action. low: run at once. mid: confirmation dialog.
/// high: confirmation sheet plus device-owner authentication.
enum Risk: String, Sendable {
    case low, mid, high
}

/// Mirror of the contract action table (contract.md section 3) and frontend useActions.ts.
enum ActionCatalog {
    struct Spec: Sendable, Equatable {
        let control: String
        let risk: Risk
    }

    private static func low(_ control: String) -> Spec { Spec(control: control, risk: .low) }
    private static func mid(_ control: String) -> Spec { Spec(control: control, risk: .mid) }
    private static func high(_ control: String) -> Spec { Spec(control: control, risk: .high) }

    static let specs: [SystemType: [String: Spec]] = [
        .docker: [
            "start": low("container"),
            "stop": low("container"),
            "restart": low("container"),
            "pause": low("container"),
            "unpause": low("container"),
            "kill": mid("container"),
            "recreate": mid("recreate"),
        ],
        .qbittorrent: [
            "resume": low("torrent"),
            "pause": low("torrent"),
            "recheck": low("torrent"),
            "top_prio": low("torrent"),
            "bottom_prio": low("torrent"),
            "increase_prio": low("torrent"),
            "decrease_prio": low("torrent"),
            "delete": mid("torrent"),
        ],
        .linux: [
            "reboot": high("power"),
            "poweroff": high("power"),
            "kill": mid("process"),
            "service_start": mid("service"),
            "service_stop": mid("service"),
            "service_restart": mid("service"),
            "fan_set": mid("fan"),
            "fan_auto": low("fan"),
        ],
        .unas: [
            "reboot": high("power"),
            "poweroff": high("power"),
        ],
        .unifi: [
            "restart_device": mid("device"),
            "kick": mid("client"),
            "block": mid("client"),
            "unblock": mid("client"),
        ],
    ]

    private static func isDeleteWithFiles(_ type: SystemType, _ action: String, _ params: ActionParams?) -> Bool {
        type == .qbittorrent && action == "delete" && params?.deleteFiles == true
    }

    /// True when `system.controls` enables the action (and `delete_files` for a delete with files).
    static func canRun(_ action: String, params: ActionParams? = nil, on system: SystemMetrics) -> Bool {
        guard let spec = specs[system.type]?[action], system.controls.contains(spec.control) else { return false }
        if isDeleteWithFiles(system.type, action, params) { return system.controls.contains("delete_files") }
        return true
    }

    /// Risk tier. An action that is not in the table is treated as high.
    static func risk(of action: String, params: ActionParams? = nil, type: SystemType) -> Risk {
        if isDeleteWithFiles(type, action, params) { return .high }
        if type == .linux, action == "kill", params?.signal == .kill { return .high }
        // 0 % stops the fan while the kernel governor is disabled.
        if type == .linux, action == "fan_set", params?.percent == 0 { return .high }
        return specs[type]?[action]?.risk ?? .high
    }
}
