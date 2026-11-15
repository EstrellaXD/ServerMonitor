import Foundation
import Testing
@testable import ServerMonitor

private final class BundleToken {}

/// Reads a fixture captured from the Go mock backend (`SERVERMONITOR_MOCK_MODE=true`).
func fixture(_ name: String) throws -> Data {
    let url = try #require(Bundle(for: BundleToken.self).url(forResource: name, withExtension: "json"))
    return try Data(contentsOf: url)
}

func decode<T: Decodable>(_ type: T.Type, fixture name: String) throws -> T {
    try WireJSON.decoder().decode(type, from: fixture(name))
}

@Suite("Wire decoding")
struct DecodingTests {
    @Test func systemsFixtureDecodesEveryType() throws {
        let systems = try decode([String: SystemMetrics].self, fixture: "systems")
        #expect(Set(systems.values.map(\.type)) == Set(SystemType.allCases))
        for (key, system) in systems {
            #expect(key == system.id)
            #expect(system.metrics != .none, "\(system.id) has no payload")
            #expect(!system.controls.isEmpty)
        }
    }

    @Test func systemsFixtureKeepsLinuxFields() throws {
        let systems = try decode([String: SystemMetrics].self, fixture: "systems")
        guard case .linux(let linux) = try #require(systems["linux-server-1"]).metrics else {
            Issue.record("linux-server-1 is not linux")
            return
        }
        #expect(!linux.cpu.cores.isEmpty)
        #expect(linux.cpu.loadAvg.count == 3)
        #expect(linux.services.contains { $0.name == "servermonitor" && $0.activeState == "active" })
        let controllable = try #require(linux.fans.first { $0.mode != nil })
        #expect(controllable.pwmPercent != nil)
        #expect(linux.fans.contains { $0.mode == nil && $0.pwmPercent == nil })
    }

    @Test func systemsFixtureKeepsUnifiClients() throws {
        let systems = try decode([String: SystemMetrics].self, fixture: "systems")
        guard case .unifi(let unifi) = try #require(systems["unifi"]).metrics else {
            Issue.record("unifi is not unifi")
            return
        }
        #expect(!unifi.clients.isEmpty)
        #expect(unifi.blockedClients.count == 1)
        #expect(unifi.devices.contains { $0.temperatures == nil })
        #expect(unifi.devices.contains { !($0.temperatures ?? []).isEmpty })
    }

    @Test func webSocketMessageDecodes() throws {
        let update = try decode(MetricsUpdate.self, fixture: "ws-update")
        #expect(update.type == "metrics_update")
        #expect(update.systems.count == 5)
        #expect(update.timestamp > Date(timeIntervalSince1970: 1_700_000_000))
    }

    @Test func dockerLogsDecode() throws {
        let logs = try decode(DockerLogs.self, fixture: "docker-logs")
        #expect(!logs.container.isEmpty)
        #expect(logs.lines.contains { $0.stream == .stdout })
        #expect(logs.lines.contains { $0.stream == .stderr })
    }

    @Test func actionResponsesDecode() throws {
        let ok = try decode(ActionResponse.self, fixture: "action-success")
        #expect(ok.status == "success")
        #expect(ok.output != nil)
        let failure = try decode(APIErrorBody.self, fixture: "action-error")
        #expect(failure.error.contains("MAC"))
    }

    @Test func offlineSystemWithEmptyMetricsDecodesAsNone() throws {
        let systems = try decode([String: SystemMetrics].self, fixture: "offline-system")
        let offline = try #require(systems["nas_backup"])
        #expect(offline.status == .offline)
        #expect(offline.metrics == .none)
        #expect(offline.controls.isEmpty)
        #expect(offline.summary == offline.error)
    }

    @Test func nullArrayDecodesAsEmpty() throws {
        let systems = try decode([String: SystemMetrics].self, fixture: "offline-system")
        guard case .docker(let docker) = try #require(systems["empty-docker"]).metrics else {
            Issue.record("empty-docker is not docker")
            return
        }
        #expect(docker.containers.isEmpty)
    }

    @Test(arguments: [
        "2026-11-15T10:48:21.406421Z",
        "2026-11-15T10:48:21Z",
        "2026-11-15T11:48:21.4+01:00",
    ])
    func rfc3339DatesParse(_ text: String) throws {
        let date = try #require(WireJSON.parseDate(text))
        #expect(abs(date.timeIntervalSince1970 - 1_794_739_701) < 1)
    }
}
