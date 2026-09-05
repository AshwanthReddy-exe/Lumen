package dev.lumen.core

internal object SpaceRecovery {
    fun recover(state: SpaceState, command: RecoverAfterRestartCommand): Transition {
        val content = CommandContent.RecoverAfterRestart(command.spaceId, command.hostEpoch, command.actorNodeId)
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            when {
                command.spaceId != current.spaceId -> reject(RejectionReason.INVALID_SPACE)
                command.hostEpoch != current.activeHostEpoch -> reject(RejectionReason.STALE_HOST_EPOCH)
                command.actorNodeId != current.activeHostNodeId ||
                    nodeRejection(current, command.actorNodeId) != null -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                else -> {
                    // The current slice has no dispatch receipt: queued work may already have run.
                    // Never infer that retrying it is safe merely because the Host restarted.
                    val interrupted = current.tasks.values.filter { it.status == TaskStatus.QUEUED }
                    var next = current.copy(tasks = current.tasks + interrupted.associate { task ->
                        task.id to task.copy(status = TaskStatus.UNKNOWN_OUTCOME, terminalReason = RejectionReason.HOST_RESTARTED)
                    })
                    interrupted.forEach { task ->
                        next = next.appendAudit(
                            command.actorNodeId, Operation.RECOVER_AFTER_RESTART, AuditOutcome.ACCEPTED,
                            "${command.operationId}:${task.id}", RejectionReason.HOST_RESTARTED,
                        )
                    }
                    accept(next, current.spaceId)
                }
            }
        }
    }
}
