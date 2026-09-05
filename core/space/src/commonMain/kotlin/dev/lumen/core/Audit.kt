package dev.lumen.core

/**
 * Redacted audit record. It intentionally contains no action arguments, artifact values, or
 * action fingerprints. The [idempotencyKey] is retained so an operator can correlate retries.
 */
data class AuditEvent(
    val sequence: Long,
    val actorNodeId: String?,
    val authorityNodeId: String,
    val hostEpoch: Long,
    val operation: Operation,
    val outcome: AuditOutcome,
    val idempotencyKey: String,
    val reason: RejectionReason? = null,
)

enum class AuditOutcome {
    ACCEPTED,
    REJECTED,
    REPLAYED,
}

internal fun SpaceState.appendAudit(
    actorNodeId: String?,
    operation: Operation,
    outcome: AuditOutcome,
    idempotencyKey: String,
    reason: RejectionReason? = null,
): SpaceState = copy(
    auditEvents = auditEvents + AuditEvent(
        sequence = auditEvents.size.toLong() + 1,
        actorNodeId = actorNodeId,
        authorityNodeId = activeHostNodeId,
        hostEpoch = activeHostEpoch,
        operation = operation,
        outcome = outcome,
        idempotencyKey = idempotencyKey,
        reason = reason,
    ),
)
