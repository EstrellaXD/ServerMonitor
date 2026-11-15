import Foundation
import Testing
@testable import ServerMonitor

@Suite("Format")
struct FormatTests {
    @Test(arguments: [
        (12.25, 1, "12.3"),
        (42.5, 0, "43"),
        (2.5, 0, "3"),
        (-2.5, 0, "-3"),
        (12.35, 1, "12.3"),
        (1.005, 2, "1.00"),
        (0.15, 1, "0.1"),
        (12.345, 2, "12.35"),
        (-0.0, 0, "0"),
        (69.6, 1, "69.6"),
    ])
    func fixedMatchesJavaScriptToFixed(value: Double, decimals: Int, expected: String) {
        #expect(Format.fixed(value, decimals) == expected)
    }

    @Test func clockUsesTwentyFourHours() {
        var components = DateComponents()
        (components.year, components.month, components.day) = (2026, 1, 1)
        (components.hour, components.minute, components.second) = (14, 5, 9)
        let date = Calendar.current.date(from: components)!
        #expect(Format.clock(date) == "14:05:09")
    }
}
