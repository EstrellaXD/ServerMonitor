import LocalAuthentication

/// Device-owner check (Face ID, Touch ID, Optic ID or passcode) for high-risk actions.
enum OwnerAuthentication {
    enum Method: Equatable {
        case faceID, touchID, opticID, passcode
        /// No biometrics and no passcode: the user types the system id instead.
        case unavailable

        var label: String {
            switch self {
            case .faceID: "Face ID"
            case .touchID: "Touch ID"
            case .opticID: "Optic ID"
            case .passcode: "Passcode"
            case .unavailable: ""
            }
        }

        var symbol: String {
            switch self {
            case .faceID: "faceid"
            case .touchID: "touchid"
            case .opticID: "opticid"
            case .passcode, .unavailable: "lock"
            }
        }
    }

    @MainActor
    static func method() -> Method {
        let context = LAContext()
        guard context.canEvaluatePolicy(.deviceOwnerAuthentication, error: nil) else { return .unavailable }
        switch context.biometryType {
        case .faceID: return .faceID
        case .touchID: return .touchID
        case .opticID: return .opticID
        default: return .passcode
        }
    }

    /// True when the owner authenticated. False on cancel or failure.
    @MainActor
    static func authenticate(reason: String) async -> Bool {
        let context = LAContext()
        return (try? await context.evaluatePolicy(.deviceOwnerAuthentication, localizedReason: reason)) ?? false
    }
}
