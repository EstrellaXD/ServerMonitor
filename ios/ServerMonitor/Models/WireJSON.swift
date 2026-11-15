import Foundation

enum WireJSON {
    /// Decoder for every backend response. Go writes `time.Time` as RFC 3339
    /// with optional fractional seconds ("2026-11-15T10:48:21.406421Z").
    static func decoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let text = try decoder.singleValueContainer().decode(String.self)
            if let date = parseDate(text) { return date }
            throw DecodingError.dataCorrupted(
                .init(codingPath: decoder.codingPath, debugDescription: "Invalid RFC 3339 date: \(text)")
            )
        }
        return decoder
    }

    static func parseDate(_ text: String) -> Date? {
        let fractional = Date.ISO8601FormatStyle(includingFractionalSeconds: true)
        return (try? fractional.parse(text)) ?? (try? Date.ISO8601FormatStyle().parse(text))
    }
}

/// Decodes a JSON array that may be `null` or absent as `[]`.
/// Go encodes a nil slice as `null`, so collectors can send it for empty lists.
@propertyWrapper
struct EmptyIfNull<Element: Decodable & Sendable & Hashable>: Decodable, Sendable, Hashable {
    var wrappedValue: [Element]

    init(wrappedValue: [Element]) {
        self.wrappedValue = wrappedValue
    }

    init(from decoder: any Decoder) throws {
        let c = try decoder.singleValueContainer()
        wrappedValue = c.decodeNil() ? [] : try c.decode([Element].self)
    }
}

extension KeyedDecodingContainer {
    func decode<Element>(_ type: EmptyIfNull<Element>.Type, forKey key: Key) throws -> EmptyIfNull<Element> {
        try decodeIfPresent(type, forKey: key) ?? EmptyIfNull(wrappedValue: [])
    }
}

/// A coding key for any string, used to look into an untyped object.
struct AnyKey: CodingKey {
    var stringValue: String
    var intValue: Int? { nil }

    init(_ string: String) { stringValue = string }
    init?(stringValue: String) { self.stringValue = stringValue }
    init?(intValue: Int) { nil }
}
