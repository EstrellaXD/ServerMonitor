import Foundation

/// Text formats. Port of frontend/src/composables/useFormatters.ts.
enum Format {
    static func bytes(_ value: Int64) -> String { bytes(Double(value)) }

    static func bytes(_ value: Double) -> String {
        let units = ["B", "KB", "MB", "GB", "TB", "PB"]
        var size = value
        var i = 0
        while size >= 1024, i < units.count - 1 {
            size /= 1024
            i += 1
        }
        return "\(fixed(size, 1)) \(units[i])"
    }

    static func speed(_ bytesPerSecond: Int64) -> String { speed(Double(bytesPerSecond)) }

    static func speed(_ bytesPerSecond: Double) -> String {
        if bytesPerSecond < 1024 { return "\(fixed(bytesPerSecond, 0)) B/s" }
        if bytesPerSecond < 1024 * 1024 { return "\(fixed(bytesPerSecond / 1024, 1)) KB/s" }
        return "\(fixed(bytesPerSecond / 1024 / 1024, 2)) MB/s"
    }

    static func uptime(_ seconds: Int64) -> String {
        let days = seconds / 86400
        let hours = (seconds % 86400) / 3600
        let minutes = (seconds % 3600) / 60
        if days > 0 { return "\(days)d \(hours)h \(minutes)m" }
        if hours > 0 { return "\(hours)h \(minutes)m" }
        return "\(minutes)m"
    }

    static func eta(_ seconds: Int64?) -> String {
        guard let seconds, seconds > 0 else { return "–" }
        if seconds > 86400 { return "\(seconds / 86400)d" }
        if seconds > 3600 { return "\(seconds / 3600)h \((seconds % 3600) / 60)m" }
        if seconds > 60 { return "\(seconds / 60)m \(seconds % 60)s" }
        return "\(seconds)s"
    }

    static func percent(_ value: Double, _ decimals: Int = 1) -> String { "\(fixed(value, decimals))%" }

    static func temperature(_ celsius: Double, _ decimals: Int = 0) -> String { "\(fixed(celsius, decimals))°C" }

    /// 24-hour HH:MM:SS in every locale (same as the web `hour12: false`).
    static func clock(_ date: Date?) -> String {
        guard let date else { return "--:--:--" }
        let style = Date.VerbatimFormatStyle(
            format: "\(hour: .twoDigits(clock: .twentyFourHour, hourCycle: .zeroBased)):\(minute: .twoDigits):\(second: .twoDigits)",
            timeZone: .current, calendar: Calendar(identifier: .gregorian)
        )
        return date.formatted(style)
    }

    /// Fixed decimals, same as JavaScript `toFixed`: rounds the exact binary value, and an exact tie away from zero.
    static func fixed(_ value: Double, _ decimals: Int) -> String {
        guard value != 0 else { return String(format: "%.\(decimals)f", 0.0) }
        let scale = pow(10, Double(decimals))
        let scaled = value.magnitude * scale
        // printf rounds an exact tie to even; JavaScript rounds it up.
        // The fused multiply-add residual is 0 only when the product is exact (12.25, not 12.35 = 12.3499…).
        let tieUp = scaled - scaled.rounded(.down) == 0.5 && (-scaled).addingProduct(value.magnitude, scale) == 0
        let rounded = tieUp ? Double(signOf: value, magnitudeOf: scaled.rounded(.up) / scale) : value
        return String(format: "%.\(decimals)f", rounded)
    }
}

enum Level {
    case ok, warn, crit

    /// Usage percent → level (defaults 75 / 90).
    static func usage(_ percent: Double, warn: Double = 75, crit: Double = 90) -> Level {
        if percent >= crit { return .crit }
        if percent >= warn { return .warn }
        return .ok
    }

    /// Temperature level: crit at the sensor's critical point, warn at its high point (defaults 85 / 70 °C).
    static func temperature(_ t: TemperatureMetrics) -> Level {
        if t.current >= (t.critical ?? 85) { return .crit }
        if t.current >= (t.high ?? 70) { return .warn }
        return .ok
    }
}
