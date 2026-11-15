import Foundation

/// One WebSocket connection to `/ws`. The stream yields every `MetricsUpdate`
/// and finishes with an error when the connection drops. Reconnect is the caller's job.
struct MetricsSocket: Sendable {
    let url: URL
    let session: URLSession

    init(url: URL, session: URLSession = .shared) {
        self.url = url
        self.session = session
    }

    func updates() -> AsyncThrowingStream<MetricsUpdate, any Error> {
        let task = session.webSocketTask(with: url)
        return AsyncThrowingStream { continuation in
            let reader = Task {
                task.resume()
                let decoder = WireJSON.decoder()
                do {
                    while true {
                        let message = try await task.receive()
                        let data: Data = switch message {
                        case .data(let data): data
                        case .string(let text): Data(text.utf8)
                        @unknown default: Data()
                        }
                        continuation.yield(try decoder.decode(MetricsUpdate.self, from: data))
                    }
                } catch {
                    continuation.finish(throwing: error)
                }
            }
            continuation.onTermination = { _ in
                reader.cancel()
                task.cancel(with: .goingAway, reason: nil)
            }
        }
    }
}
