package dev.lumen.core

/**
 * Pure, portable state for one Lumen Space.
 *
 * This module deliberately stores fingerprints rather than action arguments.  Platform,
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

data class Node(
    val id: String,
    val status: NodeStatus = NodeStatus.PAIRED,
)

enum class NodeStatus {
    PAIRED,
    REVOKED,
}

data class CapabilityKey(
    val nodeId: String,
    val capabilityId: String,
    val action: String,
)

data class CapabilityAdvertisement(
    val key: CapabilityKey,
)

enum class Grant {
    DENY,
    ASK,
    ALLOW,
}

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

enum class TaskStatus {
    AWAITING_PERMISSION,
    QUEUED,
    COMPLETED,
    FAILED,
    UNKNOWN_OUTCOME,
}

data class Approval(
    val id: String,
    val taskId: String,
    val actorNodeId: String,
    val targetNodeId: String,
    val actionFingerprint: String,
    val expiresAt: Long,
    val consumedAt: Long,
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

sealed interface Transition {
    val state: SpaceState

    data class Applied(
        override val state: SpaceState,
        val receipt: CommandReceipt,
    ) : Transition

    data class Rejected(
        override val state: SpaceState,
        val reason: RejectionReason,
    ) : Transition
}

data class CommandReceipt(
    val operation: Operation,
    val operationId: String,
    val subjectId: String,
    val taskStatus: TaskStatus? = null,
)

data class RecordedCommand(
    val content: CommandContent,
    val outcome: RecordedOutcome,
)

sealed interface RecordedOutcome {
    data class Applied(val receipt: CommandReceipt) : RecordedOutcome
    data class Rejected(val reason: RejectionReason) : RecordedOutcome
}

sealed interface CommandContent {
    val operation: Operation

    data class Pair(val actorNodeId: String, val nodeId: String) : CommandContent {
        override val operation = Operation.PAIR_NODE
    }

    data class Advertise(
        val actorNodeId: String,
        val nodeId: String,
        val capabilityId: String,
        val action: String,
    ) : CommandContent {
        override val operation = Operation.ADVERTISE_CAPABILITY
    }

    data class SetGrant(
        val actorNodeId: String,
        val key: CapabilityKey,
        val grant: Grant,
    ) : CommandContent {
        override val operation = Operation.SET_GRANT
    }

    data class Revoke(val actorNodeId: String, val nodeId: String) : CommandContent {
        override val operation = Operation.REVOKE_NODE
    }

    data class Submit(
        val spaceId: String,
        val hostEpoch: Long,
        val taskId: String,
        val originNodeId: String,
        val targetNodeId: String,
        val capabilityId: String,
        val action: String,
        val actionFingerprint: String,
    ) : CommandContent {
        override val operation = Operation.SUBMIT
    }

    data class Approve(
        val spaceId: String,
        val hostEpoch: Long,
        val approvalId: String,
        val taskId: String,
        val actorNodeId: String,
        val targetNodeId: String,
        val actionFingerprint: String,
        val expiresAt: Long,
        val approvedAt: Long,
    ) : CommandContent {
        override val operation = Operation.APPROVE
    }

    data class Complete(
        val spaceId: String,
        val hostEpoch: Long,
        val taskId: String,
        val targetNodeId: String,
        val outcome: CompletionOutcome,
    ) : CommandContent {
        override val operation = Operation.COMPLETE
    }
}

enum class Operation {
    CREATE_SPACE,
    PAIR_NODE,
    ADVERTISE_CAPABILITY,
    SET_GRANT,
    REVOKE_NODE,
    SUBMIT,
    APPROVE,
    COMPLETE,
}

enum class RejectionReason {
    INVALID_SPACE,
    INVALID_IDENTIFIER,
    INVALID_COMMAND,
    UNAUTHORIZED_ACTOR,
    NODE_ALREADY_PAIRED,
    NODE_UNKNOWN,
    NODE_REVOKED,
    ACTIVE_HOST_REVOCATION_FORBIDDEN,
    OWNER_REVOCATION_FORBIDDEN,
    CAPABILITY_NOT_ADVERTISED,
    GRANT_DENIED,
    STALE_HOST_EPOCH,
    IDEMPOTENCY_KEY_REUSED,
    TASK_ALREADY_EXISTS,
    TASK_UNKNOWN,
    INVALID_TASK_STATE,
    APPROVAL_ALREADY_CONSUMED,
    APPROVAL_NOT_REQUIRED,
    APPROVAL_MISMATCH,
    APPROVAL_EXPIRED,
    INVALID_COMPLETION_OUTCOME,
}

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

object Space {
    fun createSpace(spaceId: String, ownerNodeId: String, hostNodeId: String): SpaceState {
        requireIdentifier(spaceId)
        requireIdentifier(ownerNodeId)
        requireIdentifier(hostNodeId)
        val nodes = linkedMapOf<String, Node>()
        nodes[ownerNodeId] = Node(ownerNodeId)
        nodes[hostNodeId] = Node(hostNodeId)
        val initial = SpaceState(
            spaceId = spaceId,
            ownerNodeId = ownerNodeId,
            activeHostNodeId = hostNodeId,
            activeHostEpoch = 1,
            nodes = nodes,
        )
        return initial.appendAudit(
            actorNodeId = ownerNodeId,
            operation = Operation.CREATE_SPACE,
            outcome = AuditOutcome.ACCEPTED,
            idempotencyKey = "create:$spaceId",
        )
    }

    fun pairNode(state: SpaceState, command: PairNodeCommand): Transition {
        val content = CommandContent.Pair(command.actorNodeId, command.nodeId)
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            when {
                !isIdentifier(command.nodeId) -> reject(RejectionReason.INVALID_IDENTIFIER)
                !isOwner(current, command.actorNodeId) -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                current.nodes.containsKey(command.nodeId) -> reject(RejectionReason.NODE_ALREADY_PAIRED)
                else -> accept(
                    current.copy(nodes = current.nodes + (command.nodeId to Node(command.nodeId))),
                    command.nodeId,
                )
            }
        }
    }

    fun advertiseCapability(state: SpaceState, command: AdvertiseCapabilityCommand): Transition {
        val key = CapabilityKey(command.nodeId, command.capabilityId, command.action)
        val content = CommandContent.Advertise(
            command.actorNodeId,
            command.nodeId,
            command.capabilityId,
            command.action,
        )
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            when {
                !validCapabilityKey(key) -> reject(RejectionReason.INVALID_IDENTIFIER)
                command.actorNodeId != command.nodeId -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                nodeRejection(current, command.nodeId) != null -> reject(nodeRejection(current, command.nodeId)!!)
                else -> accept(
                    current.copy(advertisements = current.advertisements + CapabilityAdvertisement(key)),
                    command.nodeId,
                )
            }
        }
    }

    fun setGrant(state: SpaceState, command: SetGrantCommand): Transition {
        val key = CapabilityKey(command.nodeId, command.capabilityId, command.action)
        val content = CommandContent.SetGrant(command.actorNodeId, key, command.grant)
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            when {
                !validCapabilityKey(key) -> reject(RejectionReason.INVALID_IDENTIFIER)
                !isOwner(current, command.actorNodeId) -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                nodeRejection(current, command.nodeId) != null -> reject(nodeRejection(current, command.nodeId)!!)
                CapabilityAdvertisement(key) !in current.advertisements -> reject(RejectionReason.CAPABILITY_NOT_ADVERTISED)
                else -> accept(current.copy(grants = current.grants + (key to command.grant)), command.nodeId)
            }
        }
    }

    fun revokeNode(state: SpaceState, command: RevokeNodeCommand): Transition {
        val content = CommandContent.Revoke(command.actorNodeId, command.nodeId)
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            when {
                !isOwner(current, command.actorNodeId) -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                nodeRejection(current, command.nodeId) == RejectionReason.NODE_UNKNOWN -> reject(RejectionReason.NODE_UNKNOWN)
                nodeRejection(current, command.nodeId) == RejectionReason.NODE_REVOKED -> reject(RejectionReason.NODE_REVOKED)
                command.nodeId == current.activeHostNodeId -> reject(RejectionReason.ACTIVE_HOST_REVOCATION_FORBIDDEN)
                command.nodeId == current.ownerNodeId -> reject(RejectionReason.OWNER_REVOCATION_FORBIDDEN)
                else -> {
                    val invalidatedTaskIds = current.tasks.values.filter { task ->
                        if (
                            (task.originNodeId == command.nodeId || task.targetNodeId == command.nodeId) &&
                            task.status in setOf(TaskStatus.AWAITING_PERMISSION, TaskStatus.QUEUED)
                        ) {
                            true
                        } else {
                            false
                        }
                    }.map { it.id }
                    val invalidatedTasks = current.tasks.mapValues { (_, task) ->
                        if (task.id in invalidatedTaskIds) task.copy(status = TaskStatus.FAILED, terminalReason = RejectionReason.NODE_REVOKED) else task
                    }
                    var next = current.copy(
                        nodes = current.nodes + (command.nodeId to Node(command.nodeId, NodeStatus.REVOKED)),
                        tasks = invalidatedTasks,
                    )
                    invalidatedTaskIds.forEach { taskId ->
                        next = next.appendAudit(
                            actorNodeId = command.actorNodeId,
                            operation = Operation.REVOKE_NODE,
                            outcome = AuditOutcome.ACCEPTED,
                            idempotencyKey = "${command.operationId}:$taskId",
                            reason = RejectionReason.NODE_REVOKED,
                        )
                    }
                    accept(
                        next,
                        command.nodeId,
                    )
                }
            }
        }
    }

    fun submit(state: SpaceState, command: SubmitCommand): Transition {
        val content = CommandContent.Submit(
            command.spaceId,
            command.hostEpoch,
            command.taskId,
            command.originNodeId,
            command.targetNodeId,
            command.capabilityId,
            command.action,
            command.actionFingerprint,
        )
        return execute(state, command.operationId, content, command.originNodeId) { current ->
            val key = CapabilityKey(command.targetNodeId, command.capabilityId, command.action)
            when {
                command.spaceId != current.spaceId -> reject(RejectionReason.INVALID_SPACE)
                command.hostEpoch != current.activeHostEpoch -> reject(RejectionReason.STALE_HOST_EPOCH)
                !isIdentifier(command.taskId) || !isIdentifier(command.actionFingerprint) || !validCapabilityKey(key) ->
                    reject(RejectionReason.INVALID_IDENTIFIER)
                nodeRejection(current, command.originNodeId) != null -> reject(nodeRejection(current, command.originNodeId)!!)
                nodeRejection(current, command.targetNodeId) != null -> reject(nodeRejection(current, command.targetNodeId)!!)
                CapabilityAdvertisement(key) !in current.advertisements -> reject(RejectionReason.CAPABILITY_NOT_ADVERTISED)
                command.taskId in current.tasks -> reject(RejectionReason.TASK_ALREADY_EXISTS)
                current.grants[key] != Grant.ALLOW && current.grants[key] != Grant.ASK -> reject(RejectionReason.GRANT_DENIED)
                current.grants[key] == Grant.ASK -> {
                    val task = taskFor(command, TaskStatus.AWAITING_PERMISSION)
                    accept(current.copy(tasks = current.tasks + (task.id to task)), task.id, task.status)
                }
                else -> {
                    val task = taskFor(command, TaskStatus.QUEUED)
                    accept(current.copy(tasks = current.tasks + (task.id to task)), task.id, task.status)
                }
            }
        }
    }

    fun approve(state: SpaceState, command: ApproveCommand): Transition {
        val content = CommandContent.Approve(
            command.spaceId,
            command.hostEpoch,
            command.approvalId,
            command.taskId,
            command.actorNodeId,
            command.targetNodeId,
            command.actionFingerprint,
            command.expiresAt,
            command.approvedAt,
        )
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            val task = current.tasks[command.taskId]
            when {
                command.spaceId != current.spaceId -> reject(RejectionReason.INVALID_SPACE)
                command.hostEpoch != current.activeHostEpoch -> reject(RejectionReason.STALE_HOST_EPOCH)
                !isIdentifier(command.approvalId) || !isIdentifier(command.taskId) || !isIdentifier(command.actionFingerprint) ->
                    reject(RejectionReason.INVALID_IDENTIFIER)
                !isOwner(current, command.actorNodeId) -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                nodeRejection(current, command.actorNodeId) != null -> reject(nodeRejection(current, command.actorNodeId)!!)
                task == null -> reject(RejectionReason.TASK_UNKNOWN)
                nodeRejection(current, task.originNodeId) != null -> reject(nodeRejection(current, task.originNodeId)!!)
                nodeRejection(current, task.targetNodeId) != null -> reject(nodeRejection(current, task.targetNodeId)!!)
                command.approvalId in current.approvals -> reject(RejectionReason.APPROVAL_ALREADY_CONSUMED)
                task.status != TaskStatus.AWAITING_PERMISSION -> reject(
                    if (task.status == TaskStatus.QUEUED) RejectionReason.APPROVAL_NOT_REQUIRED else RejectionReason.INVALID_TASK_STATE,
                )
                command.targetNodeId != task.targetNodeId || command.actionFingerprint != task.actionFingerprint ->
                    reject(RejectionReason.APPROVAL_MISMATCH)
                command.approvedAt >= command.expiresAt -> reject(RejectionReason.APPROVAL_EXPIRED)
                current.grants[CapabilityKey(task.targetNodeId, task.capabilityId, task.action)] != Grant.ASK ->
                    reject(RejectionReason.GRANT_DENIED)
                else -> {
                    val approval = Approval(
                        command.approvalId,
                        task.id,
                        command.actorNodeId,
                        command.targetNodeId,
                        command.actionFingerprint,
                        command.expiresAt,
                        command.approvedAt,
                    )
                    val queued = task.copy(status = TaskStatus.QUEUED)
                    accept(
                        current.copy(
                            approvals = current.approvals + (approval.id to approval),
                            tasks = current.tasks + (queued.id to queued),
                        ),
                        queued.id,
                        queued.status,
                    )
                }
            }
        }
    }

    fun complete(state: SpaceState, command: CompleteCommand): Transition {
        val content = CommandContent.Complete(
            command.spaceId,
            command.hostEpoch,
            command.taskId,
            command.targetNodeId,
            command.outcome,
        )
        return execute(state, command.operationId, content, command.targetNodeId) { current ->
            val task = current.tasks[command.taskId]
            when {
                command.spaceId != current.spaceId -> reject(RejectionReason.INVALID_SPACE)
                command.hostEpoch != current.activeHostEpoch -> reject(RejectionReason.STALE_HOST_EPOCH)
                !isIdentifier(command.taskId) -> reject(RejectionReason.INVALID_IDENTIFIER)
                task == null -> reject(RejectionReason.TASK_UNKNOWN)
                nodeRejection(current, command.targetNodeId) != null -> reject(nodeRejection(current, command.targetNodeId)!!)
                nodeRejection(current, task.originNodeId) != null -> reject(nodeRejection(current, task.originNodeId)!!)
                command.targetNodeId != task.targetNodeId -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                task.status != TaskStatus.QUEUED -> reject(RejectionReason.INVALID_TASK_STATE)
                else -> {
                    val finalState = when (command.outcome) {
                        CompletionOutcome.COMPLETED -> TaskStatus.COMPLETED
                        CompletionOutcome.FAILED -> TaskStatus.FAILED
                        CompletionOutcome.UNKNOWN_OUTCOME -> TaskStatus.UNKNOWN_OUTCOME
                    }
                    val completed = task.copy(status = finalState)
                    accept(
                        current.copy(tasks = current.tasks + (completed.id to completed)),
                        completed.id,
                        completed.status,
                    )
                }
            }
        }
    }

    private fun taskFor(command: SubmitCommand, status: TaskStatus) = Task(
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

    private fun execute(
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
            val replayed = state.appendAudit(
                actorNodeId = actorNodeId,
                operation = content.operation,
                outcome = AuditOutcome.REPLAYED,
                idempotencyKey = operationId,
            )
            return when (val outcome = existing.outcome) {
                is RecordedOutcome.Applied -> Transition.Applied(replayed, outcome.receipt)
                is RecordedOutcome.Rejected -> Transition.Rejected(replayed, outcome.reason)
            }
        }
        return when (val decision = action(state)) {
            is Decision.Accept -> {
                val receipt = CommandReceipt(content.operation, operationId, decision.subjectId, decision.taskStatus)
                val recorded = decision.state.copy(commands = decision.state.commands + (operationId to RecordedCommand(content, RecordedOutcome.Applied(receipt))))
                Transition.Applied(
                    recorded.appendAudit(actorNodeId, content.operation, AuditOutcome.ACCEPTED, operationId),
                    receipt,
                )
            }
            is Decision.Reject -> {
                val recorded = state.copy(commands = state.commands + (operationId to RecordedCommand(content, RecordedOutcome.Rejected(decision.reason))))
                Transition.Rejected(
                    recorded.appendAudit(actorNodeId, content.operation, AuditOutcome.REJECTED, operationId, decision.reason),
                    decision.reason,
                )
            }
        }
    }

    private fun accepted(state: SpaceState, subjectId: String, taskStatus: TaskStatus? = null) =
        Decision.Accept(state, subjectId, taskStatus)

    private fun accept(state: SpaceState, subjectId: String, taskStatus: TaskStatus? = null) =
        accepted(state, subjectId, taskStatus)

    private fun reject(reason: RejectionReason) = Decision.Reject(reason)

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

    private fun isOwner(state: SpaceState, actorNodeId: String) =
        actorNodeId == state.ownerNodeId && state.nodes[actorNodeId]?.status == NodeStatus.PAIRED

    private fun nodeRejection(state: SpaceState, nodeId: String): RejectionReason? = when (state.nodes[nodeId]?.status) {
        NodeStatus.PAIRED -> null
        NodeStatus.REVOKED -> RejectionReason.NODE_REVOKED
        null -> RejectionReason.NODE_UNKNOWN
    }

    private fun validCapabilityKey(key: CapabilityKey) =
        isIdentifier(key.nodeId) && isIdentifier(key.capabilityId) && isIdentifier(key.action)

    private fun isIdentifier(value: String) = value.isNotBlank()

    private fun requireIdentifier(value: String) {
        require(isIdentifier(value)) { "Identifiers must not be blank." }
    }

    private sealed interface Decision {
        data class Accept(val state: SpaceState, val subjectId: String, val taskStatus: TaskStatus? = null) : Decision
        data class Reject(val reason: RejectionReason) : Decision
    }
}

private fun SpaceState.appendAudit(
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
