package dev.lumen.core

/**
 * Pure, portable state for one Lumen Space.
 *
 * This module deliberately stores fingerprints rather than action arguments. Platform,
 * transport, cryptographic, and persistence concerns remain outside this boundary.
 */
data class SpaceState(
    val spaceId: String,
    val ownerNodeId: String,
    val activeHostNodeId: String,
    val activeHostEpoch: Long,
    val nodes: Map<String, Node>,
    val advertisements: Set<CapabilityAdvertisement> = emptySet(),
    val grants: Map<CapabilityKey, Grant> = emptyMap(),
    val tasks: Map<String, Task> = emptyMap(),
    val approvals: Map<String, Approval> = emptyMap(),
    val commands: Map<String, RecordedCommand> = emptyMap(),
    val auditEvents: List<AuditEvent> = emptyList(),
)

data class Node(val id: String, val status: NodeStatus = NodeStatus.PAIRED)

enum class NodeStatus { PAIRED, REVOKED }

data class CapabilityKey(val nodeId: String, val capabilityId: String, val action: String)

data class CapabilityAdvertisement(val key: CapabilityKey)

enum class Grant { DENY, ASK, ALLOW }

data class Task(
    val id: String,
    val commandId: String,
    val originNodeId: String,
    val targetNodeId: String,
    val capabilityId: String,
    val action: String,
    val actionFingerprint: String,
    val hostEpoch: Long,
    val status: TaskStatus,
    val terminalReason: RejectionReason? = null,
)

enum class TaskStatus { AWAITING_PERMISSION, QUEUED, COMPLETED, FAILED, UNKNOWN_OUTCOME }

data class Approval(
    val id: String,
    val taskId: String,
    val actorNodeId: String,
    val targetNodeId: String,
    val actionFingerprint: String,
    val expiresAt: Long,
    val consumedAt: Long,
)
