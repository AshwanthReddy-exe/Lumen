import Foundation
import XCTest
@testable import NativeSchemaSpike

final class NativeSchemaSpikeTests: XCTestCase {
    private let evaluatedAt = ISO8601DateFormatter().date(from: "2026-09-04T12:00:00Z")!
    private let fixtureDirectory = URL(fileURLWithPath: #filePath).deletingLastPathComponent().appendingPathComponent("../../../fixtures")

    private func fixture(_ name: String) throws -> Data { try Data(contentsOf: fixtureDirectory.appendingPathComponent(name)) }

    func testValidPairingRequestIsAccepted() throws {
        let request = try PairingRequest(json: fixture("pairing-request.valid.json"), evaluatedAt: evaluatedAt)
        XCTAssertEqual(request.taskId, "task_demo0001")
    }

    func testExpiredRequestIsRejected() throws { XCTAssertThrowsError(try PairingRequest(json: fixture("pairing-request.expired.json"), evaluatedAt: evaluatedAt)) }
    func testUnsupportedVersionIsRejected() throws { XCTAssertThrowsError(try PairingRequest(json: fixture("pairing-request.unsupported-version.json"), evaluatedAt: evaluatedAt)) }
    func testUnknownFieldIsRejected() throws { XCTAssertThrowsError(try PairingRequest(json: fixture("pairing-request.unknown-field.json"), evaluatedAt: evaluatedAt)) }

    func testUnknownPayloadFieldIsRejected() throws {
        var json = String(data: try fixture("pairing-request.valid.json"), encoding: .utf8)!
        json = json.replacingOccurrences(of: "\"displayName\":", with: "\"untrustedHint\": true, \"displayName\":")
        XCTAssertThrowsError(try PairingRequest(json: Data(json.utf8), evaluatedAt: evaluatedAt))
    }

    func testNonIncreasingTimestampsAreRejected() throws {
        var json = String(data: try fixture("pairing-request.valid.json"), encoding: .utf8)!
        json = json.replacingOccurrences(of: "2026-09-04T11:59:00Z", with: "2026-09-04T12:01:00Z")
        XCTAssertThrowsError(try PairingRequest(json: Data(json.utf8), evaluatedAt: evaluatedAt))
    }

    func testSSEPreservesIdsTypesAndData() throws {
        let text = String(data: try fixture("task-events.sse"), encoding: .utf8)!
        let events = SSEParser.parse(text)
        XCTAssertEqual(events.count, 3)
        XCTAssertEqual(events.map(\.id), ["event_demo0001", "event_demo0002", "event_demo0003"])
        XCTAssertEqual(events.map(\.type), ["task.status", "task.status", "task.status"])
        XCTAssertEqual(events.map(\.data), ["{\"taskId\":\"task_demo0001\",\"state\":\"running\"}", "{\"taskId\":\"task_demo0001\",\"state\":\"awaiting_permission\"}", "{\"taskId\":\"task_demo0001\",\n\"state\":\"completed\"}"])
    }
}
