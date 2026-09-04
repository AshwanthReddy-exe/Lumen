import Foundation

private struct DynamicCodingKey: CodingKey {
    let stringValue: String
    init?(stringValue: String) { self.stringValue = stringValue }
    let intValue: Int? = nil
    init?(intValue: Int) { return nil }
}

public struct PairingRequest: Codable, Equatable, Sendable {
    public let schemaVersion: Int
    public let kind: String
    public let spaceId: String
    public let senderNodeId: String
    public let recipientNodeId: String
    public let taskId: String
    public let nonce: String
    public let issuedAt: Date
    public let expiresAt: Date
    public let payload: Payload

    public struct Payload: Codable, Equatable, Sendable {
        public let displayName: String
        public let ephemeralPublicKey: String
        public let confirmationCode: String

        public init(from decoder: Decoder) throws {
            let dynamic = try decoder.container(keyedBy: DynamicCodingKey.self)
            let allowed = Set(Key.allCases.map(\.stringValue))
            if let unknown = dynamic.allKeys.map(\.stringValue).first(where: { !allowed.contains($0) }) { throw ValidationError("unknown field: \(unknown)") }
            let c = try decoder.container(keyedBy: Key.self)
            displayName = try c.decode(String.self, forKey: .displayName)
            ephemeralPublicKey = try c.decode(String.self, forKey: .ephemeralPublicKey)
            confirmationCode = try c.decode(String.self, forKey: .confirmationCode)
            guard (1...80).contains(displayName.count) else { throw ValidationError("displayName length is invalid") }
            guard (32...256).contains(ephemeralPublicKey.count) else { throw ValidationError("ephemeralPublicKey length is invalid") }
            guard confirmationCode.range(of: "^[0-9]{6}$", options: .regularExpression) != nil else { throw ValidationError("confirmationCode is invalid") }
        }

        private enum Key: String, CodingKey, CaseIterable { case displayName, ephemeralPublicKey, confirmationCode }
    }

    public enum ValidationError: Error, LocalizedError, Equatable {
        case invalid(String)
        public init(_ message: String) { self = .invalid(message) }
        public var errorDescription: String? { if case .invalid(let message) = self { return message }; return nil }
    }

    public init(json data: Data, evaluatedAt: Date) throws {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let value = try decoder.singleValueContainer().decode(String.self)
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime]
            guard let date = formatter.date(from: value) else { throw ValidationError("invalid date-time") }
            return date
        }
        let value: PairingRequest
        do { value = try decoder.decode(Self.self, from: data) }
        catch let error as ValidationError { throw error }
        catch { throw ValidationError(error.localizedDescription) }
        guard value.schemaVersion == 1 else { throw ValidationError("unsupported schema version: \(value.schemaVersion)") }
        guard value.kind == "pairing.request" else { throw ValidationError("unsupported kind: \(value.kind)") }
        try Self.validateIdentifier(value.spaceId, prefix: "space_")
        try Self.validateIdentifier(value.senderNodeId, prefix: "node_")
        try Self.validateIdentifier(value.recipientNodeId, prefix: "node_")
        try Self.validateIdentifier(value.taskId, prefix: "task_")
        guard (16...128).contains(value.nonce.count) else { throw ValidationError("nonce length is invalid") }
        guard value.expiresAt > value.issuedAt else { throw ValidationError("expiresAt must be after issuedAt") }
        guard value.expiresAt > evaluatedAt else { throw ValidationError("pairing request is expired") }
        self = value
    }

    private enum Key: String, CodingKey, CaseIterable { case schemaVersion, kind, spaceId, senderNodeId, recipientNodeId, taskId, nonce, issuedAt, expiresAt, payload }
    public init(from decoder: Decoder) throws {
        let dynamic = try decoder.container(keyedBy: DynamicCodingKey.self)
        let allowed = Set(Key.allCases.map(\.stringValue))
        if let unknown = dynamic.allKeys.map(\.stringValue).first(where: { !allowed.contains($0) }) { throw ValidationError("unknown field: \(unknown)") }
        let c = try decoder.container(keyedBy: Key.self)
        schemaVersion = try c.decode(Int.self, forKey: .schemaVersion)
        kind = try c.decode(String.self, forKey: .kind)
        spaceId = try c.decode(String.self, forKey: .spaceId)
        senderNodeId = try c.decode(String.self, forKey: .senderNodeId)
        recipientNodeId = try c.decode(String.self, forKey: .recipientNodeId)
        taskId = try c.decode(String.self, forKey: .taskId)
        nonce = try c.decode(String.self, forKey: .nonce)
        issuedAt = try c.decode(Date.self, forKey: .issuedAt)
        expiresAt = try c.decode(Date.self, forKey: .expiresAt)
        payload = try c.decode(Payload.self, forKey: .payload)
    }

    private static func validateIdentifier(_ value: String, prefix: String) throws {
        guard value.range(of: "^\(prefix)[a-z0-9]{8}$", options: .regularExpression) != nil else { throw ValidationError("invalid identifier: \(value)") }
    }
}

public struct ServerSentEvent: Equatable, Sendable {
    public let id: String?
    public let type: String?
    public let data: String
}

public enum SSEParser {
    public static func parse(_ text: String) -> [ServerSentEvent] {
        text.replacingOccurrences(of: "\r\n", with: "\n").components(separatedBy: "\n\n").compactMap { block in
            let lines = block.split(whereSeparator: { $0 == "\n" || $0 == "\r" }).map(String.init)
            guard !lines.isEmpty else { return nil }
            var id: String?, type: String?, data: [String] = []
            for line in lines {
                if line.hasPrefix("id:") { id = String(line.dropFirst(3)).trimmingCharacters(in: .whitespaces) }
                else if line.hasPrefix("event:") { type = String(line.dropFirst(6)).trimmingCharacters(in: .whitespaces) }
                else if line.hasPrefix("data:") { data.append(String(line.dropFirst(5)).trimmingCharacters(in: .whitespaces)) }
            }
            guard !data.isEmpty else { return nil }
            return ServerSentEvent(id: id, type: type, data: data.joined(separator: "\n"))
        }
    }
}
