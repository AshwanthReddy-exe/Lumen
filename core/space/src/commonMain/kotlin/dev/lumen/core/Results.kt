package dev.lumen.core

import kotlinx.serialization.Serializable

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

@Serializable data class CommandReceipt(
    val operation: Operation,
    val operationId: String,
    val subjectId: String,
    val taskStatus: TaskStatus? = null,
)

@Serializable data class RecordedCommand(
    val content: CommandContent,
    val outcome: RecordedOutcome,
)

@Serializable sealed interface RecordedOutcome {
    @Serializable data class Applied(val receipt: CommandReceipt) : RecordedOutcome
    @Serializable data class Rejected(val reason: RejectionReason) : RecordedOutcome
}

@Serializable sealed interface CommandContent {
    val operation: Operation

    @Serializable data class RecoverAfterRestart(
        val spaceId: String,
        val hostEpoch: Long,
        val actorNodeId: String,
    ) : CommandContent {
        override val operation = Operation.RECOVER_AFTER_RESTART
    }

    @Serializable data class Pair(val actorNodeId: String, val nodeId: String) : CommandContent {
        override val operation = Operation.PAIR_NODE
    }

    @Serializable data class Advertise(
        val actorNodeId: String,
        val nodeId: String,
        val capabilityId: String,
        val action: String,
    ) : CommandContent {
        override val operation = Operation.ADVERTISE_CAPABILITY
    }

    @Serializable data class SetGrant(
        val actorNodeId: String,
        val key: CapabilityKey,
        val grant: Grant,
    ) : CommandContent {
        override val operation = Operation.SET_GRANT
    }

    @Serializable data class Revoke(val actorNodeId: String, val nodeId: String) : CommandContent {
        override val operation = Operation.REVOKE_NODE
    }

    @Serializable data class Submit(
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

    @Serializable data class Approve(
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

    @Serializable data class Complete(
        val spaceId: String,
        val hostEpoch: Long,
        val taskId: String,
        val targetNodeId: String,
        val outcome: CompletionOutcome,
    ) : CommandContent {
        override val operation = Operation.COMPLETE
    }
}

@Serializable enum class Operation {
    RECOVER_AFTER_RESTART,
    CREATE_SPACE,
    PAIR_NODE,
    ADVERTISE_CAPABILITY,
    SET_GRANT,
    REVOKE_NODE,
    SUBMIT,
    APPROVE,
    COMPLETE,
}

@Serializable enum class RejectionReason {
    PERSISTENCE_UNAVAILABLE,
    HOST_RESTARTED,
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
