package dev.lumen.core

internal object SpaceLifecycle {
    fun create(spaceId: String, ownerNodeId: String, hostNodeId: String): SpaceState {
        requireIdentifier(spaceId)
        requireIdentifier(ownerNodeId)
        requireIdentifier(hostNodeId)
        val nodes = linkedMapOf<String, Node>()
        nodes[ownerNodeId] = Node(ownerNodeId)
        nodes[hostNodeId] = Node(hostNodeId)
        return SpaceState(
            spaceId = spaceId,
            ownerNodeId = ownerNodeId,
            activeHostNodeId = hostNodeId,
            activeHostEpoch = 1,
            nodes = nodes,
        ).appendAudit(ownerNodeId, Operation.CREATE_SPACE, AuditOutcome.ACCEPTED, "create:$spaceId")
    }
}

internal object SpaceMembership {
    fun pair(state: SpaceState, command: PairNodeCommand): Transition {
        val content = CommandContent.Pair(command.actorNodeId, command.nodeId)
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            when {
                !isIdentifier(command.nodeId) -> reject(RejectionReason.INVALID_IDENTIFIER)
                !isOwner(current, command.actorNodeId) -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                current.nodes.containsKey(command.nodeId) -> reject(RejectionReason.NODE_ALREADY_PAIRED)
                else -> accept(current.copy(nodes = current.nodes + (command.nodeId to Node(command.nodeId))), command.nodeId)
            }
        }
    }

    fun advertise(state: SpaceState, command: AdvertiseCapabilityCommand): Transition {
        val key = CapabilityKey(command.nodeId, command.capabilityId, command.action)
        val content = CommandContent.Advertise(command.actorNodeId, command.nodeId, command.capabilityId, command.action)
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

    fun revoke(state: SpaceState, command: RevokeNodeCommand): Transition {
        val content = CommandContent.Revoke(command.actorNodeId, command.nodeId)
        return execute(state, command.operationId, content, command.actorNodeId) { current ->
            when {
                !isOwner(current, command.actorNodeId) -> reject(RejectionReason.UNAUTHORIZED_ACTOR)
                nodeRejection(current, command.nodeId) == RejectionReason.NODE_UNKNOWN -> reject(RejectionReason.NODE_UNKNOWN)
                nodeRejection(current, command.nodeId) == RejectionReason.NODE_REVOKED -> reject(RejectionReason.NODE_REVOKED)
                command.nodeId == current.activeHostNodeId -> reject(RejectionReason.ACTIVE_HOST_REVOCATION_FORBIDDEN)
                command.nodeId == current.ownerNodeId -> reject(RejectionReason.OWNER_REVOCATION_FORBIDDEN)
                else -> revokeNode(current, command)
            }
        }
    }

    private fun revokeNode(state: SpaceState, command: RevokeNodeCommand): Decision {
        val invalidatedTaskIds = state.tasks.values.filter { task ->
            (task.originNodeId == command.nodeId || task.targetNodeId == command.nodeId) &&
                task.status in setOf(TaskStatus.AWAITING_PERMISSION, TaskStatus.QUEUED)
        }.map { it.id }
        val invalidatedTasks = state.tasks.mapValues { (_, task) ->
            if (task.id in invalidatedTaskIds) task.copy(status = TaskStatus.FAILED, terminalReason = RejectionReason.NODE_REVOKED) else task
        }
        var next = state.copy(
            nodes = state.nodes + (command.nodeId to Node(command.nodeId, NodeStatus.REVOKED)),
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
        return accept(next, command.nodeId)
    }
}
