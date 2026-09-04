package dev.lumen.spike

import kotlin.time.Instant
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

private val json = Json { ignoreUnknownKeys = false }
private val rootKeys = setOf("schemaVersion", "kind", "spaceId", "senderNodeId", "recipientNodeId", "taskId", "nonce", "issuedAt", "expiresAt", "payload")
private val payloadKeys = setOf("displayName", "ephemeralPublicKey", "confirmationCode")

data class PairingRequest(val taskId: String, val expiresAt: Instant)

sealed class PairingResult {
    data class Accepted(val request: PairingRequest) : PairingResult()
    data class Rejected(val reason: String) : PairingResult()
}

fun parsePairingRequest(raw: String, evaluatedAt: Instant): PairingResult = try {
    val root = json.parseToJsonElement(raw).jsonObject
    rejectUnknown(root, rootKeys)
    require(root["schemaVersion"]?.jsonPrimitive?.intOrNull == 1) { "unsupported schemaVersion" }
    require(requiredString(root, "kind") == "pairing.request") { "unexpected kind" }
    requirePattern(root, "spaceId", "space_[a-z0-9]{8}")
    requirePattern(root, "senderNodeId", "node_[a-z0-9]{8}")
    requirePattern(root, "recipientNodeId", "node_[a-z0-9]{8}")
    requirePattern(root, "taskId", "task_[a-z0-9]{8}")
    require(requiredString(root, "nonce").length in 16..128) { "invalid nonce" }
    val issued = Instant.parse(requiredString(root, "issuedAt"))
    val expires = Instant.parse(requiredString(root, "expiresAt"))
    require(expires > issued) { "expiresAt must follow issuedAt" }
    require(expires > evaluatedAt) { "request expired" }
    val payload = root["payload"]?.jsonObject ?: error("missing payload")
    rejectUnknown(payload, payloadKeys)
    require(requiredString(payload, "displayName").length in 1..80) { "invalid displayName" }
    require(requiredString(payload, "ephemeralPublicKey").length in 32..256) { "invalid ephemeralPublicKey" }
    require(Regex("^[0-9]{6}$").matches(requiredString(payload, "confirmationCode"))) { "invalid confirmationCode" }
    PairingResult.Accepted(PairingRequest(requiredString(root, "taskId"), expires))
} catch (e: Exception) {
    PairingResult.Rejected(e.message ?: "invalid pairing request")
}

private fun rejectUnknown(value: JsonObject, allowed: Set<String>) {
    require(value.keys.all { it in allowed }) { "unknown field" }
    require(allowed.all { it in value }) { "missing field" }
}

private fun requiredString(value: JsonObject, key: String): String {
    val primitive = value[key]?.jsonPrimitive ?: error("invalid or missing $key")
    require(primitive.isString) { "invalid or missing $key" }
    return primitive.contentOrNull ?: error("invalid or missing $key")
}

private fun requirePattern(value: JsonObject, key: String, pattern: String) {
    require(Regex("^$pattern$").matches(requiredString(value, key))) { "invalid $key" }
}

data class TaskEvent(val id: String, val type: String, val data: String)

fun parseTaskEventsSse(raw: String): List<TaskEvent> {
    val events = mutableListOf<TaskEvent>()
    var id: String? = null
    var type: String? = null
    val data = mutableListOf<String>()
    fun emit() {
        if (id != null || type != null || data.isNotEmpty()) {
            events += TaskEvent(id ?: "", type ?: "message", data.joinToString("\n"))
            id = null; type = null; data.clear()
        }
    }
    raw.replace("\r\n", "\n").split('\n').forEach { line ->
        when {
            line.isEmpty() -> emit()
            line.startsWith("id:") -> id = line.removePrefix("id:").trimStart()
            line.startsWith("event:") -> type = line.removePrefix("event:").trimStart()
            line.startsWith("data:") -> data += line.removePrefix("data:").trimStart()
        }
    }
    emit()
    return events
}
