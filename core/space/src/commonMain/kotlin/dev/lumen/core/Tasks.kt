package dev.lumen.core

internal object SpaceTasks {
    fun submit(state: SpaceState, command: SubmitCommand): Transition {
        val content = CommandContent.Submit(
            command.spaceId, command.hostEpoch, command.taskId, command.originNodeId,
            command.targetNodeId, command.capabilityId, command.action, command.actionFingerprint,
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
                else -> {
                    val status = if (current.grants[key] == Grant.ASK) TaskStatus.AWAITING_PERMISSION else TaskStatus.QUEUED
                    val task = taskFor(command, status)
                    accept(current.copy(tasks = current.tasks + (task.id to task)), task.id, task.status)
                }
            }
        }
    }

    fun approve(state: SpaceState, command: ApproveCommand): Transition {
        val content = CommandContent.Approve(
            command.spaceId, command.hostEpoch, command.approvalId, command.taskId, command.actorNodeId,
            command.targetNodeId, command.actionFingerprint, command.expiresAt, command.approvedAt,
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
                        command.approvalId, task.id, command.actorNodeId, command.targetNodeId,
                        command.actionFingerprint, command.expiresAt, command.approvedAt,
                    )
                    val queued = task.copy(status = TaskStatus.QUEUED)
                    accept(
                        current.copy(approvals = current.approvals + (approval.id to approval), tasks = current.tasks + (queued.id to queued)),
                        queued.id,
                        queued.status,
                    )
                }
            }
        }
    }

    fun complete(state: SpaceState, command: CompleteCommand): Transition {
        val content = CommandContent.Complete(command.spaceId, command.hostEpoch, command.taskId, command.targetNodeId, command.outcome)
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
                    accept(current.copy(tasks = current.tasks + (completed.id to completed)), completed.id, completed.status)
                }
            }
        }
    }
}
