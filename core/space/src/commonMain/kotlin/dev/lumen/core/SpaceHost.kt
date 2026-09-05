package dev.lumen.core

/**
 * The serial Host boundary for portable Space transitions.
 *
 * A platform store must atomically create only an empty store, serialize [transact], atomically
 * persist its resulting state, and encrypt the serialized representation. A caller receives a
 * transition only after that commit.
 */
interface SpaceStateStore {
    fun read(): SpaceStateStoreRead
    fun initialize(state: SpaceState): SpaceStateStoreInitialize
    fun transact(operation: (SpaceState) -> Transition): SpaceStateStoreCommit
}

enum class SpaceStateStoreInitialize { CREATED, ALREADY_EXISTS, UNAVAILABLE }

sealed interface SpaceStateStoreRead {
    data class Present(val state: SpaceState) : SpaceStateStoreRead
    data object Missing : SpaceStateStoreRead
    data object Unavailable : SpaceStateStoreRead
}

sealed interface SpaceStateStoreCommit {
    data class Committed(val transition: Transition) : SpaceStateStoreCommit
    data class Unavailable(val lastCommittedState: SpaceState) : SpaceStateStoreCommit
}

sealed interface SpaceHostOpenResult {
    data class Ready(val host: SpaceHost) : SpaceHostOpenResult
    data class Unavailable(val reason: HostUnavailableReason) : SpaceHostOpenResult
}

enum class HostUnavailableReason { STATE_NOT_FOUND, STATE_ALREADY_EXISTS, PERSISTENCE_UNAVAILABLE, RECOVERY_REJECTED }

class SpaceHost private constructor(
    private val store: SpaceStateStore,
) {
    fun snapshot(): SpaceStateStoreRead = store.read()

    fun pairNode(command: PairNodeCommand) = commit { Space.pairNode(it, command) }
    fun advertiseCapability(command: AdvertiseCapabilityCommand) = commit { Space.advertiseCapability(it, command) }
    fun setGrant(command: SetGrantCommand) = commit { Space.setGrant(it, command) }
    fun revokeNode(command: RevokeNodeCommand) = commit { Space.revokeNode(it, command) }
    fun submit(command: SubmitCommand) = commit { Space.submit(it, command) }
    fun approve(command: ApproveCommand) = commit { Space.approve(it, command) }
    fun complete(command: CompleteCommand) = commit { Space.complete(it, command) }

    private fun commit(operation: (SpaceState) -> Transition): Transition = when (val result = store.transact(operation)) {
        is SpaceStateStoreCommit.Committed -> result.transition
        is SpaceStateStoreCommit.Unavailable -> Transition.Rejected(result.lastCommittedState, RejectionReason.PERSISTENCE_UNAVAILABLE)
    }

    companion object {
        fun create(store: SpaceStateStore, spaceId: String, ownerNodeId: String, hostNodeId: String): SpaceHostOpenResult {
            val state = Space.createSpace(spaceId, ownerNodeId, hostNodeId)
            return when (store.initialize(state)) {
                SpaceStateStoreInitialize.CREATED -> SpaceHostOpenResult.Ready(SpaceHost(store))
                SpaceStateStoreInitialize.ALREADY_EXISTS -> SpaceHostOpenResult.Unavailable(HostUnavailableReason.STATE_ALREADY_EXISTS)
                SpaceStateStoreInitialize.UNAVAILABLE -> SpaceHostOpenResult.Unavailable(HostUnavailableReason.PERSISTENCE_UNAVAILABLE)
            }
        }

        fun open(store: SpaceStateStore, recoveryOperationId: String): SpaceHostOpenResult {
            when (store.read()) {
                SpaceStateStoreRead.Missing -> return SpaceHostOpenResult.Unavailable(HostUnavailableReason.STATE_NOT_FOUND)
                SpaceStateStoreRead.Unavailable -> return SpaceHostOpenResult.Unavailable(HostUnavailableReason.PERSISTENCE_UNAVAILABLE)
                is SpaceStateStoreRead.Present -> Unit
            }
            val recovered = store.transact { state ->
                Space.recoverAfterRestart(
                    state,
                    RecoverAfterRestartCommand(recoveryOperationId, state.spaceId, state.activeHostEpoch, state.activeHostNodeId),
                )
            }
            return when (recovered) {
                is SpaceStateStoreCommit.Committed -> if (recovered.transition is Transition.Applied) {
                    SpaceHostOpenResult.Ready(SpaceHost(store))
                } else {
                    SpaceHostOpenResult.Unavailable(HostUnavailableReason.RECOVERY_REJECTED)
                }
                is SpaceStateStoreCommit.Unavailable -> SpaceHostOpenResult.Unavailable(HostUnavailableReason.PERSISTENCE_UNAVAILABLE)
            }
        }
    }
}
