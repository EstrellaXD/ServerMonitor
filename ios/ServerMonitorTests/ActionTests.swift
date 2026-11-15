import Foundation
import Testing
@testable import ServerMonitor

private func system(_ type: SystemType, controls: [String]) -> SystemMetrics {
    SystemMetrics(id: "sys", name: "Sys", type: type, status: .healthy, lastUpdated: .now,
                  controls: controls, metrics: .none)
}

@Suite("Risk tiers")
struct RiskTests {
    @Test(arguments: [
        (SystemType.docker, "start", Risk.low),
        (.docker, "stop", .low),
        (.docker, "restart", .low),
        (.docker, "pause", .low),
        (.docker, "unpause", .low),
        (.docker, "kill", .mid),
        (.docker, "recreate", .mid),
        (.qbittorrent, "resume", .low),
        (.qbittorrent, "pause", .low),
        (.qbittorrent, "recheck", .low),
        (.qbittorrent, "top_prio", .low),
        (.qbittorrent, "bottom_prio", .low),
        (.qbittorrent, "increase_prio", .low),
        (.qbittorrent, "decrease_prio", .low),
        (.qbittorrent, "delete", .mid),
        (.linux, "reboot", .high),
        (.linux, "poweroff", .high),
        (.linux, "kill", .mid),
        (.linux, "service_start", .mid),
        (.linux, "service_stop", .mid),
        (.linux, "service_restart", .mid),
        (.linux, "fan_set", .mid),
        (.linux, "fan_auto", .low),
        (.unas, "reboot", .high),
        (.unas, "poweroff", .high),
        (.unifi, "restart_device", .mid),
        (.unifi, "kick", .mid),
        (.unifi, "block", .mid),
        (.unifi, "unblock", .mid),
    ])
    func baseRiskMatchesContract(type: SystemType, action: String, risk: Risk) {
        #expect(ActionCatalog.risk(of: action, type: type) == risk)
    }

    @Test func paramsRaiseRisk() {
        #expect(ActionCatalog.risk(of: "delete", params: ActionParams(deleteFiles: true), type: .qbittorrent) == .high)
        #expect(ActionCatalog.risk(of: "delete", params: ActionParams(deleteFiles: false), type: .qbittorrent) == .mid)
        #expect(ActionCatalog.risk(of: "kill", params: ActionParams(signal: .kill), type: .linux) == .high)
        #expect(ActionCatalog.risk(of: "kill", params: ActionParams(signal: .term), type: .linux) == .mid)
        #expect(ActionCatalog.risk(of: "fan_set", params: ActionParams(percent: 0), type: .linux) == .high)
        #expect(ActionCatalog.risk(of: "fan_set", params: ActionParams(percent: 25), type: .linux) == .mid)
    }

    @Test func unknownActionIsHigh() {
        #expect(ActionCatalog.risk(of: "format_disk", type: .linux) == .high)
        #expect(ActionCatalog.risk(of: "kick", type: .docker) == .high)
    }
}

@Suite("Control gating")
struct GatingTests {
    @Test func actionNeedsItsControl() {
        let docker = system(.docker, controls: ["container"])
        #expect(ActionCatalog.canRun("restart", on: docker))
        #expect(ActionCatalog.canRun("kill", on: docker))
        #expect(!ActionCatalog.canRun("recreate", on: docker))
    }

    @Test func noControlsAllowsNothing() {
        let linux = system(.linux, controls: [])
        for action in ActionCatalog.specs[.linux, default: [:]].keys {
            #expect(!ActionCatalog.canRun(action, on: linux), "\(action)")
        }
    }

    @Test func actionOfAnotherTypeIsRefused() {
        #expect(!ActionCatalog.canRun("reboot", on: system(.docker, controls: ["container", "recreate"])))
        #expect(!ActionCatalog.canRun("kick", on: system(.unas, controls: ["power"])))
    }

    @Test func deleteWithFilesNeedsDeleteFilesControl() {
        let plain = system(.qbittorrent, controls: ["torrent"])
        let full = system(.qbittorrent, controls: ["torrent", "delete_files"])
        let withFiles = ActionParams(deleteFiles: true)
        #expect(ActionCatalog.canRun("delete", params: ActionParams(deleteFiles: false), on: plain))
        #expect(!ActionCatalog.canRun("delete", params: withFiles, on: plain))
        #expect(ActionCatalog.canRun("delete", params: withFiles, on: full))
        #expect(!ActionCatalog.canRun("delete", params: withFiles, on: system(.qbittorrent, controls: ["delete_files"])))
    }

    @Test func mockControlsEnableEveryActionOfTheirType() throws {
        let systems = try decode([String: SystemMetrics].self, fixture: "systems")
        for system in systems.values {
            for action in ActionCatalog.specs[system.type, default: [:]].keys {
                #expect(system.can(action), "\(system.id) \(action)")
            }
        }
    }
}

@Suite("Action request encoding")
struct ActionRequestTests {
    private let base = URL(string: "http://raspberrypi.local:8742")!

    private func json(_ request: URLRequest) throws -> [String: Any] {
        let body = try #require(request.httpBody)
        return try #require(try JSONSerialization.jsonObject(with: body) as? [String: Any])
    }

    @Test func postsJSONToTheSystemActionsPath() throws {
        let request = try APIClient.actionRequest(
            baseURL: base, systemID: "docker-host", body: ActionRequestBody(action: "restart", target: "nginx")
        )
        #expect(request.httpMethod == "POST")
        #expect(request.url?.absoluteString == "http://raspberrypi.local:8742/api/systems/docker-host/actions")
        #expect(request.value(forHTTPHeaderField: "Content-Type") == "application/json")
        let object = try json(request)
        #expect(object["action"] as? String == "restart")
        #expect(object["target"] as? String == "nginx")
        #expect(object["params"] == nil)
        #expect(object.count == 2)
    }

    @Test func emptyTargetIsSent() throws {
        let request = try APIClient.actionRequest(
            baseURL: base, systemID: "linux-server-1", body: ActionRequestBody(action: "reboot", target: "")
        )
        #expect(try json(request)["target"] as? String == "")
    }

    @Test func paramsUseWireKeysAndOmitNil() throws {
        let kill = try json(APIClient.actionRequest(
            baseURL: base, systemID: "linux-server-1",
            body: ActionRequestBody(action: "kill", target: "1351", params: ActionParams(signal: .kill))
        ))
        #expect(kill["params"] as? [String: String] == ["signal": "KILL"])

        let fan = try json(APIClient.actionRequest(
            baseURL: base, systemID: "linux-server-1",
            body: ActionRequestBody(action: "fan_set", target: "pwmfan/fan1", params: ActionParams(percent: 50))
        ))
        #expect(fan["params"] as? [String: Int] == ["percent": 50])

        let delete = try json(APIClient.actionRequest(
            baseURL: base, systemID: "qbittorrent",
            body: ActionRequestBody(action: "delete", target: "abc", params: ActionParams(deleteFiles: true))
        ))
        #expect(delete["params"] as? [String: Bool] == ["delete_files": true])
    }

    @Test func systemIDIsOnePathComponent() throws {
        let request = try APIClient.actionRequest(
            baseURL: base, systemID: "a/b", body: ActionRequestBody(action: "start", target: "x")
        )
        #expect(request.url?.path(percentEncoded: true) == "/api/systems/a%2Fb/actions")
    }
}

@Suite("Server address")
struct ServerAddressTests {
    @Test(arguments: ["raspberrypi.local:8742", " raspberrypi.local:8742 ", "http://raspberrypi.local:8742/"])
    func acceptsHostPort(_ text: String) throws {
        let address = try #require(ServerAddress(text))
        #expect(address.hostPort == "raspberrypi.local:8742")
        #expect(address.httpURL.absoluteString == "http://raspberrypi.local:8742")
        #expect(address.webSocketURL.absoluteString == "ws://raspberrypi.local:8742/ws")
    }

    @Test(arguments: ["", "   ", "host/path", "https://host:8742", "http://"])
    func rejectsInvalid(_ text: String) {
        #expect(ServerAddress(text) == nil)
    }
}

@Suite("Summaries")
struct SummaryTests {
    @Test func summariesMatchWebFormat() throws {
        let systems = try decode([String: SystemMetrics].self, fixture: "systems")
        let docker = try #require(systems["docker-host"])
        #expect(docker.summary.hasSuffix("running"))
        let linux = try #require(systems["linux-server-1"])
        #expect(linux.summary.hasPrefix("cpu "))
        #expect(linux.summary.contains(" mem "))
        let unas = try #require(systems["unas"])
        #expect(unas.summary.hasPrefix("1 pool "))
    }

    @Test func formatMatchesJavaScript() {
        #expect(Format.bytes(Int64(1536)) == "1.5 KB")
        #expect(Format.speed(Int64(512)) == "512 B/s")
        #expect(Format.speed(Int64(3 * 1024 * 1024)) == "3.00 MB/s")
        #expect(Format.uptime(90061) == "1d 1h 1m")
        #expect(Format.eta(nil) == "–")
        #expect(Format.eta(125) == "2m 5s")
    }
}
