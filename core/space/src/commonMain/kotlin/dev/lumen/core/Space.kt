package dev.lumen.core

/**
 * Pure, portable state transitions for one Lumen Space.
 *
 * Platform, transport, cryptographic, and persistence concerns remain outside this boundary.
 */
object Space {
    fun recoverAfterRestart(state: SpaceState, command: RecoverAfterRestartCommand): Transition =
        SpaceRecovery.recover(state, command)

    fun createSpace(spaceId: String, ownerNodeId: String, hostNodeId: String): SpaceState =
        SpaceLifecycle.create(spaceId, ownerNodeId, hostNodeId)

    fun pairNode(state: SpaceState, command: PairNodeCommand): Transition =
        SpaceMembership.pair(state, command)

    fun advertiseCapability(state: SpaceState, command: AdvertiseCapabilityCommand): Transition =
        SpaceMembership.advertise(state, command)

    fun setGrant(state: SpaceState, command: SetGrantCommand): Transition =
        SpaceMembership.setGrant(state, command)

    fun revokeNode(state: SpaceState, command: RevokeNodeCommand): Transition =
        SpaceMembership.revoke(state, command)

    fun submit(state: SpaceState, command: SubmitCommand): Transition =
        SpaceTasks.submit(state, command)

    fun approve(state: SpaceState, command: ApproveCommand): Transition =
        SpaceTasks.approve(state, command)

    fun complete(state: SpaceState, command: CompleteCommand): Transition =
        SpaceTasks.complete(state, command)
}
