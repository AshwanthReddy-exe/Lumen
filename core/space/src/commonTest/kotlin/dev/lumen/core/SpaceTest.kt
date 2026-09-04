package dev.lumen.core

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertIs
import kotlin.test.assertTrue

class SpaceTest {
    @Test
    fun `creates one epoch-one Space and pairs owner and host`() {
        val state = Space.createSpace("space", "android", "android")

        assertEquals(1, state.activeHostEpoch)
        assertEquals("android", state.ownerNodeId)
        assertEquals("android", state.activeHostNodeId)
        assertEquals(setOf("android"), state.nodes.keys)
        assertEquals(Operation.CREATE_SPACE, state.auditEvents.single().operation)
        assertEquals(AuditOutcome.ACCEPTED, state.auditEvents.single().outcome)
    }

    @Test
    fun `pairs nodes and rejects duplicate or non-owner pairing`() {
        val initial = Space.createSpace("space", "android", "android")
        val paired = applied(Space.pairNode(initial, PairNodeCommand("pair-mac", "android", "mac")))

        assertEquals(NodeStatus.PAIRED, paired.nodes.getValue("mac").status)
        assertRejected(
            Space.pairNode(paired, PairNodeCommand("pair-mac-duplicate", "android", "mac")),
            RejectionReason.NODE_ALREADY_PAIRED,
        )
        assertRejected(
            Space.pairNode(paired, PairNodeCommand("pair-by-mac", "mac", "iphone")),
            RejectionReason.UNAUTHORIZED_ACTOR,
        )
    }

    @Test
    fun `deny ask allow produce honest lifecycle states`() {
        val denied = readySpace(Grant.DENY)
        assertRejected(Space.submit(denied, command("deny-task")), RejectionReason.GRANT_DENIED)

        val asking = readySpace(Grant.ASK)
        val awaiting = applied(Space.submit(asking, command("ask-task")))
        assertEquals(TaskStatus.AWAITING_PERMISSION, awaiting.tasks.getValue("ask-task").status)

        val queued = applied(Space.approve(awaiting, approval("approve-ask", "approval-ask", "ask-task")))
        assertEquals(TaskStatus.QUEUED, queued.tasks.getValue("ask-task").status)
        val completed = applied(Space.complete(queued, completion("complete-ask", "ask-task", CompletionOutcome.COMPLETED)))
        assertEquals(TaskStatus.COMPLETED, completed.tasks.getValue("ask-task").status)

        val allowed = readySpace(Grant.ALLOW)
        val allowResult = Space.submit(allowed, command("allow-task"))
        assertIs<Transition.Applied>(allowResult)
        assertEquals(TaskStatus.QUEUED, allowResult.state.tasks.getValue("allow-task").status)
    }

    @Test
    fun `submit defaults to deny when capability is absent or ungranted`() {
        val noAdvertisement = pairedSpace()
        assertRejected(Space.submit(noAdvertisement, command("no-advertisement")), RejectionReason.CAPABILITY_NOT_ADVERTISED)

        val advertised = applied(
            Space.advertiseCapability(
                noAdvertisement,
                AdvertiseCapabilityCommand("advertise", "mac", "mac", "coding.run", "apply"),
            ),
        )
        assertRejected(Space.submit(advertised, command("no-grant")), RejectionReason.GRANT_DENIED)
    }

    @Test
    fun `submit rejects invalid origin target and stale host epoch`() {
        val state = readySpace(Grant.ALLOW)

        assertRejected(
            Space.submit(state, command("unknown-origin").copy(originNodeId = "unknown")),
            RejectionReason.NODE_UNKNOWN,
        )
        assertRejected(
            Space.submit(state, command("unknown-target").copy(targetNodeId = "unknown")),
            RejectionReason.NODE_UNKNOWN,
        )
        assertRejected(
            Space.submit(state, command("stale-submit").copy(hostEpoch = 0)),
            RejectionReason.STALE_HOST_EPOCH,
        )
    }

    @Test
    fun `revocation blocks work and invalidates queued and awaiting tasks`() {
        val queued = applied(Space.submit(readySpace(Grant.ALLOW), command("queued-before-revoke")))
        val revoked = applied(Space.revokeNode(queued, RevokeNodeCommand("revoke-mac", "android", "mac")))

        assertEquals(TaskStatus.FAILED, revoked.tasks.getValue("queued-before-revoke").status)
        assertEquals(RejectionReason.NODE_REVOKED, revoked.tasks.getValue("queued-before-revoke").terminalReason)
        assertRejected(Space.submit(revoked, command("after-revoke")), RejectionReason.NODE_REVOKED)
        assertRejected(
            Space.complete(revoked, completion("complete-revoked", "queued-before-revoke", CompletionOutcome.COMPLETED)),
            RejectionReason.NODE_REVOKED,
        )

        val awaiting = applied(Space.submit(readySpace(Grant.ASK), command("awaiting-before-revoke")))
        val awaitingRevoked = applied(Space.revokeNode(awaiting, RevokeNodeCommand("revoke-awaiting-mac", "android", "mac")))
        assertEquals(TaskStatus.FAILED, awaitingRevoked.tasks.getValue("awaiting-before-revoke").status)
        assertRejected(
            Space.approve(awaitingRevoked, approval("approve-revoked", "approval-revoked", "awaiting-before-revoke")),
            RejectionReason.NODE_REVOKED,
        )
        assertTrue(awaitingRevoked.auditEvents.any { it.idempotencyKey == "revoke-awaiting-mac:awaiting-before-revoke" && it.reason == RejectionReason.NODE_REVOKED })
    }

    @Test
    fun `owner must remain paired and cannot be revoked without a recovery flow`() {
        val separateOwnerAndHost = Space.createSpace("space", "owner", "host")
        assertRejected(
            Space.revokeNode(separateOwnerAndHost, RevokeNodeCommand("revoke-owner", "owner", "owner")),
            RejectionReason.OWNER_REVOCATION_FORBIDDEN,
        )
    }

    @Test
    fun `audit identifies the active Host as authority`() {
        val state = Space.createSpace("space", "owner", "host")
        assertEquals("host", state.auditEvents.single().authorityNodeId)
    }

    @Test
    fun `approval validates current epoch exact binding expiry and once-only use`() {
        val awaiting = applied(Space.submit(readySpace(Grant.ASK), command("approval-task")))

        assertRejected(
            Space.approve(awaiting, approval("wrong-epoch", "approval-wrong-epoch", "approval-task").copy(hostEpoch = 2)),
            RejectionReason.STALE_HOST_EPOCH,
        )
        assertRejected(
            Space.approve(awaiting, approval("wrong-target", "approval-wrong-target", "approval-task").copy(targetNodeId = "iphone")),
            RejectionReason.APPROVAL_MISMATCH,
        )
        assertRejected(
            Space.approve(awaiting, approval("wrong-fingerprint", "approval-wrong-fingerprint", "approval-task").copy(actionFingerprint = "different")),
            RejectionReason.APPROVAL_MISMATCH,
        )
        assertRejected(
            Space.approve(awaiting, approval("expired", "approval-expired", "approval-task").copy(approvedAt = 10, expiresAt = 10)),
            RejectionReason.APPROVAL_EXPIRED,
        )

        val queued = applied(Space.approve(awaiting, approval("approve", "approval", "approval-task")))
        assertEquals(TaskStatus.QUEUED, queued.tasks.getValue("approval-task").status)
        assertRejected(
            Space.approve(queued, approval("approve-again", "approval", "approval-task")),
            RejectionReason.APPROVAL_ALREADY_CONSUMED,
        )
    }

    @Test
    fun `completion checks target epoch and terminal task state`() {
        val queued = applied(Space.submit(readySpace(Grant.ALLOW), command("completion-task")))
        assertRejected(
            Space.complete(queued, completion("wrong-target", "completion-task", CompletionOutcome.COMPLETED).copy(targetNodeId = "iphone")),
            RejectionReason.UNAUTHORIZED_ACTOR,
        )
        assertRejected(
            Space.complete(queued, completion("stale-completion", "completion-task", CompletionOutcome.COMPLETED).copy(hostEpoch = 2)),
            RejectionReason.STALE_HOST_EPOCH,
        )

        val unknown = applied(Space.complete(queued, completion("unknown-outcome", "completion-task", CompletionOutcome.UNKNOWN_OUTCOME)))
        assertEquals(TaskStatus.UNKNOWN_OUTCOME, unknown.tasks.getValue("completion-task").status)
        assertRejected(
            Space.complete(unknown, completion("complete-after-unknown", "completion-task", CompletionOutcome.COMPLETED)),
            RejectionReason.INVALID_TASK_STATE,
        )
    }

    @Test
    fun `repeated commands replay first result and altered content collides`() {
        val initial = readySpace(Grant.ALLOW)
        val firstCommand = command("first-task", operationId = "same-command")
        val first = Space.submit(initial, firstCommand)
        val firstState = applied(first)

        val replay = Space.submit(firstState, firstCommand)
        assertIs<Transition.Applied>(replay)
        assertEquals("first-task", replay.receipt.subjectId)
        assertEquals(AuditOutcome.REPLAYED, replay.state.auditEvents.last().outcome)

        val changedCommands = listOf(
            firstCommand.copy(spaceId = "other-space"),
            firstCommand.copy(hostEpoch = 2),
            firstCommand.copy(originNodeId = "mac"),
            firstCommand.copy(targetNodeId = "iphone"),
            firstCommand.copy(capabilityId = "reminder.manage"),
            firstCommand.copy(action = "list"),
            firstCommand.copy(actionFingerprint = "different-digest"),
        )
        changedCommands.forEach { changed ->
            assertRejected(Space.submit(firstState, changed), RejectionReason.IDEMPOTENCY_KEY_REUSED)
        }
    }

    @Test
    fun `audit includes authority metadata but excludes action payload fingerprints`() {
        val fingerprint = "private-artifact-digest-never-in-audit"
        val submitted = Space.submit(readySpace(Grant.ALLOW), command("audit-task", fingerprint = fingerprint))
        val event = submitted.state.auditEvents.last()

        assertEquals("iphone", event.actorNodeId)
        assertEquals("android", event.authorityNodeId)
        assertEquals(1, event.hostEpoch)
        assertEquals(Operation.SUBMIT, event.operation)
        assertEquals(AuditOutcome.ACCEPTED, event.outcome)
        assertEquals("audit-task-command", event.idempotencyKey)
        assertFalse(submitted.state.auditEvents.toString().contains(fingerprint))
        assertTrue(submitted.state.auditEvents.isNotEmpty())
    }

    private fun pairedSpace(): SpaceState {
        val created = Space.createSpace("space", "android", "android")
        val mac = applied(Space.pairNode(created, PairNodeCommand("pair-mac", "android", "mac")))
        return applied(Space.pairNode(mac, PairNodeCommand("pair-iphone", "android", "iphone")))
    }

    private fun readySpace(grant: Grant): SpaceState {
        val advertised = applied(
            Space.advertiseCapability(
                pairedSpace(),
                AdvertiseCapabilityCommand("advertise-mac-coding", "mac", "mac", "coding.run", "apply"),
            ),
        )
        return applied(
            Space.setGrant(
                advertised,
                SetGrantCommand("grant-mac-coding", "android", "mac", "coding.run", "apply", grant),
            ),
        )
    }

    private fun command(taskId: String, fingerprint: String = "digest", operationId: String = "$taskId-command") =
        SubmitCommand(
            operationId = operationId,
            spaceId = "space",
            hostEpoch = 1,
            taskId = taskId,
            originNodeId = "iphone",
            targetNodeId = "mac",
            capabilityId = "coding.run",
            action = "apply",
            actionFingerprint = fingerprint,
        )

    private fun approval(operationId: String, approvalId: String, taskId: String) = ApproveCommand(
        operationId = operationId,
        spaceId = "space",
        hostEpoch = 1,
        approvalId = approvalId,
        taskId = taskId,
        actorNodeId = "android",
        targetNodeId = "mac",
        actionFingerprint = "digest",
        expiresAt = 10,
        approvedAt = 1,
    )

    private fun completion(operationId: String, taskId: String, outcome: CompletionOutcome) = CompleteCommand(
        operationId = operationId,
        spaceId = "space",
        hostEpoch = 1,
        taskId = taskId,
        targetNodeId = "mac",
        outcome = outcome,
    )

    private fun applied(transition: Transition): SpaceState = assertIs<Transition.Applied>(transition).state

    private fun assertRejected(transition: Transition, expected: RejectionReason) {
        assertEquals(expected, assertIs<Transition.Rejected>(transition).reason)
    }
}
