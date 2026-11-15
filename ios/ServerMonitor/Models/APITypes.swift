import Foundation

// Wire types for the action and docker-logs endpoints. Mirror of frontend/src/types/api.ts.

enum KillSignal: String, Encodable, Sendable, Hashable {
    case term = "TERM"
    case kill = "KILL"
}

/// `params` of an action request. Nil fields are left out of the JSON.
struct ActionParams: Encodable, Sendable, Hashable {
    var deleteFiles: Bool?
    var signal: KillSignal?
    var percent: Int?

    enum CodingKeys: String, CodingKey {
        case signal, percent
        case deleteFiles = "delete_files"
    }
}

struct ActionRequestBody: Encodable, Sendable, Hashable {
    var action: String
    var target: String
    var params: ActionParams?
}

struct ActionResponse: Decodable, Sendable, Hashable {
    var status: String
    var action: String
    var target: String
    var output: String?
}

struct APIErrorBody: Decodable, Sendable {
    var error: String
}

struct DockerLogLine: Decodable, Sendable, Hashable {
    enum Stream: String, Decodable, Sendable {
        case stdout, stderr
    }

    var stream: Stream
    var text: String
}

struct DockerLogs: Decodable, Sendable, Hashable {
    var container: String
    @EmptyIfNull var lines: [DockerLogLine]
}
