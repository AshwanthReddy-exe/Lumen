package dev.lumen.core

internal fun execute(
    state: SpaceState,
    operationId: String,
    content: CommandContent,
    actorNodeId: String?,
    action: (SpaceState) -> Decision,
): Transition {
    if (!isIdentifier(operationId)) {
        return rejected(state, actorNodeId, content.operation, operationId, RejectionReason.INVALID_IDENTIFIER)
    }
    val existing = state.commands[operationId]
    if (existing != null) {
        if (existing.content != content) {
            return rejected(state, actorNodeId, content.operation, operationId, RejectionReason.IDEMPOTENCY_KEY_REUSED)
        }
        val replayed = state.appendAudit(actorNodeId, content.operation, AuditOutcome.REPLAYED, operationId)
        return when (val outcome = existing.outcome) {
            is RecordedOutcome.Applied -> Transition.Applied(replayed, outcome.receipt)
            is RecordedOutcome.Rejected -> Transition.Rejected(replayed, outcome.reason)
        }
    }
    return when (val decision = action(state)) {
        is Decision.Accept -> {
            val receipt = CommandReceipt(content.operation, operationId, decision.subjectId, decision.taskStatus)
            val recorded = decision.state.copy(
                commands = decision.state.commands + (operationId to RecordedCommand(content, RecordedOutcome.Applied(receipt))),
            )
            Transition.Applied(recorded.appendAudit(actorNodeId, content.operation, AuditOutcome.ACCEPTED, operationId), receipt)
        }
        is Decision.Reject -> {
            val recorded = state.copy(
                commands = state.commands + (operationId to RecordedCommand(content, RecordedOutcome.Rejected(decision.reason))),
            )
            Transition.Rejected(
                recorded.appendAudit(actorNodeId, content.operation, AuditOutcome.REJECTED, operationId, decision.reason),
                decision.reason,
            )
        }
    }
}

internal fun accept(state: SpaceState, subjectId: String, taskStatus: TaskStatus? = null) =
    Decision.Accept(state, subjectId, taskStatus)

internal fun reject(reason: RejectionReason) = Decision.Reject(reason)

private fun rejected(
    state: SpaceState,
    actorNodeId: String?,
    operation: Operation,
    operationId: String,
    reason: RejectionReason,
) = Transition.Rejected(
    state.appendAudit(actorNodeId, operation, AuditOutcome.REJECTED, operationId, reason),
    reason,
)

internal fun isOwner(state: SpaceState, actorNodeId: String) =
    actorNodeId == state.ownerNodeId && state.nodes[actorNodeId]?.status == NodeStatus.PAIRED

internal fun nodeRejection(state: SpaceState, nodeId: String): RejectionReason? = when (state.nodes[nodeId]?.status) {
    NodeStatus.PAIRED -> null
    NodeStatus.REVOKED -> RejectionReason.NODE_REVOKED
    null -> RejectionReason.NODE_UNKNOWN
}

internal fun validCapabilityKey(key: CapabilityKey) =
    isIdentifier(key.nodeId) && isIdentifier(key.capabilityId) && isIdentifier(key.action)

internal fun isIdentifier(value: String) = value.isNotBlank()

internal fun requireIdentifier(value: String) {
    require(isIdentifier(value)) { "Identifiers must not be blank." }
}

internal fun taskFor(command: SubmitCommand, status: TaskStatus) = Task(
    id = command.taskId,
    commandId = command.operationId,
    originNodeId = command.originNodeId,
    targetNodeId = command.targetNodeId,
    capabilityId = command.capabilityId,
    action = command.action,
    actionFingerprint = command.actionFingerprint,
    hostEpoch = command.hostEpoch,
    status = status,
)

internal sealed interface Decision {
    data class Accept(val state: SpaceState, val subjectId: String, val taskStatus: TaskStatus? = null) : Decision
    data class Reject(val reason: RejectionReason) : Decision
}
