import Foundation
import NativeSchemaSpike

@main
struct NativeSchemaSpikeChecker {
    private static let evaluatedAt = ISO8601DateFormatter().date(from: "2026-09-04T12:00:00Z")!

    static func main() throws {
        let directory = URL(fileURLWithPath: CommandLine.arguments.dropFirst().first ?? "../fixtures", isDirectory: true)
        func data(_ name: String) throws -> Data { try Data(contentsOf: directory.appendingPathComponent(name)) }

        let valid = try PairingRequest(json: data("pairing-request.valid.json"), evaluatedAt: evaluatedAt)
        try check(valid.taskId == "task_demo0001", "valid fixture task ID")
        try rejects(data("pairing-request.expired.json"), named: "expired fixture")
        try rejects(data("pairing-request.unsupported-version.json"), named: "unsupported-version fixture")
        try rejects(data("pairing-request.unknown-field.json"), named: "unknown-field fixture")
        let validText = String(data: try data("pairing-request.valid.json"), encoding: .utf8) ?? ""
        let unknownPayload = validText.replacingOccurrences(of: "\"displayName\":", with: "\"untrustedHint\": true, \"displayName\":")
        try rejects(Data(unknownPayload.utf8), named: "unknown payload field")
        let invalidOrdering = validText.replacingOccurrences(of: "2026-09-04T11:59:00Z", with: "2026-09-04T12:06:00Z")
        try rejects(Data(invalidOrdering.utf8), named: "non-increasing timestamps")

        let sse = String(data: try data("task-events.sse"), encoding: .utf8) ?? ""
        let events = SSEParser.parse(sse)
        try check(events.count == 3, "SSE event count")
        try check(events.map(\.id) == ["event_demo0001", "event_demo0002", "event_demo0003"], "SSE event IDs")
        try check(events.map(\.type) == ["task.status", "task.status", "task.status"], "SSE event types")
        try check(events.map(\.data) == [
            "{\"taskId\":\"task_demo0001\",\"state\":\"running\"}",
            "{\"taskId\":\"task_demo0001\",\"state\":\"awaiting_permission\"}",
            "{\"taskId\":\"task_demo0001\",\n\"state\":\"completed\"}"
        ], "SSE event data")
        print("NativeSchemaSpikeChecker: all contract checks passed")
    }

    private static func rejects(_ data: Data, named name: String) throws {
        do {
            _ = try PairingRequest(json: data, evaluatedAt: evaluatedAt)
            throw CheckerError.failure("expected rejection: \(name)")
        } catch is PairingRequest.ValidationError {
            return
        }
    }

    private static func check(_ condition: Bool, _ name: String) throws {
        guard condition else { throw CheckerError.failure("failed check: \(name)") }
    }
}

private enum CheckerError: Error, LocalizedError {
    case failure(String)
    var errorDescription: String? { if case .failure(let message) = self { return message }; return nil }
}
