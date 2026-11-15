import Foundation

/// `host:port` of the backend, as the user types it.
struct ServerAddress: Hashable, Sendable {
    static let defaultValue = "192.168.31.170:8742"

    let hostPort: String
    let httpURL: URL
    let webSocketURL: URL

    /// Accepts `host`, `host:port`, or a pasted `http://host:port/`. Returns nil for anything else.
    init?(_ text: String) {
        var value = text.trimmingCharacters(in: .whitespacesAndNewlines)
        if value.lowercased().hasPrefix("http://") { value.removeFirst(7) }
        while value.hasSuffix("/") { value.removeLast() }
        guard !value.isEmpty, !value.contains("/"),
              let http = URL(string: "http://\(value)"), let host = http.host(), !host.isEmpty,
              let ws = URL(string: "ws://\(value)/ws")
        else { return nil }
        hostPort = value
        httpURL = http
        webSocketURL = ws
    }
}

enum APIError: LocalizedError, Equatable {
    /// The server answered with an error status; `message` is its `{error}` text.
    case server(status: Int, message: String)
    case invalidResponse

    var errorDescription: String? {
        switch self {
        case .server(_, let message): message
        case .invalidResponse: "The server sent an invalid response."
        }
    }
}

/// REST client for `/api`. Stateless; create one per server address.
struct APIClient: Sendable {
    let address: ServerAddress
    let session: URLSession

    init(address: ServerAddress, session: URLSession = .shared) {
        self.address = address
        self.session = session
    }

    func systems() async throws -> [String: SystemMetrics] {
        var request = URLRequest(url: address.httpURL.appending(path: "api/systems"))
        request.timeoutInterval = 10
        return try await send(request)
    }

    func run(_ body: ActionRequestBody, on systemID: String) async throws -> ActionResponse {
        try await send(Self.actionRequest(baseURL: address.httpURL, systemID: systemID, body: body))
    }

    func dockerLogs(systemID: String, container: String, tail: Int = 200) async throws -> DockerLogs {
        var url = address.httpURL.appending(components: "api", "systems", systemID, "docker", "logs")
        url.append(queryItems: [.init(name: "container", value: container), .init(name: "tail", value: String(tail))])
        var request = URLRequest(url: url)
        request.timeoutInterval = 15
        return try await send(request)
    }

    /// The POST for an action. The backend rejects a body without `Content-Type: application/json` (415).
    static func actionRequest(baseURL: URL, systemID: String, body: ActionRequestBody) throws -> URLRequest {
        var request = URLRequest(url: baseURL.appending(components: "api", "systems", systemID, "actions"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        // Recreate pulls an image; allow it time.
        request.timeoutInterval = 300
        return request
    }

    private func send<T: Decodable>(_ request: URLRequest) async throws -> T {
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }
        let decoder = WireJSON.decoder()
        guard (200..<300).contains(http.statusCode) else {
            let message = (try? decoder.decode(APIErrorBody.self, from: data))?.error ?? "HTTP \(http.statusCode)"
            throw APIError.server(status: http.statusCode, message: message)
        }
        return try decoder.decode(T.self, from: data)
    }
}
