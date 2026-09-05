package dev.lumen.core

data class RecoverAfterRestartCommand(
    val operationId: String,
    val spaceId: String,
    val hostEpoch: Long,
    val actorNodeId: String,
)

/** Commands carry no action payload. [actionFingerprint] is a canonical digest supplied by a caller. */
data class SubmitCommand(
    val operationId: String,
    val spaceId: String,
    val hostEpoch: Long,
    val taskId: String,
    val originNodeId: String,
    val targetNodeId: String,
    val capabilityId: String,
    val action: String,
    val actionFingerprint: String,
)

data class PairNodeCommand(
    val operationId: String,
    val actorNodeId: String,
    val nodeId: String,
)

data class AdvertiseCapabilityCommand(
    val operationId: String,
    val actorNodeId: String,
    val nodeId: String,
    val capabilityId: String,
    val action: String,
)

data class SetGrantCommand(
    val operationId: String,
    val actorNodeId: String,
    val nodeId: String,
    val capabilityId: String,
    val action: String,
    val grant: Grant,
)

data class RevokeNodeCommand(
    val operationId: String,
    val actorNodeId: String,
    val nodeId: String,
)

data class ApproveCommand(
    val operationId: String,
    val spaceId: String,
    val hostEpoch: Long,
    val approvalId: String,
    val taskId: String,
    val actorNodeId: String,
    val targetNodeId: String,
    val actionFingerprint: String,
    val expiresAt: Long,
    val approvedAt: Long,
)

data class CompleteCommand(
    val operationId: String,
    val spaceId: String,
    val hostEpoch: Long,
    val taskId: String,
    val targetNodeId: String,
    val outcome: CompletionOutcome,
)

enum class CompletionOutcome {
    COMPLETED,
    FAILED,
    UNKNOWN_OUTCOME,
}
