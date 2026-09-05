package dev.lumen.core

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

class RecoveryTest {
    @Test
    fun `restart marks only queued work unknown and preserves authority and approvals`() {
        var before = readySpace()
        listOf("queued", "completed", "failed", "unknown").forEach { id ->
            before = applied(Space.submit(before, submit(id)))
        }
        before = applied(Space.complete(before, complete("completed", CompletionOutcome.COMPLETED)))
        before = applied(Space.complete(before, complete("failed", CompletionOutcome.FAILED)))
        before = applied(Space.complete(before, complete("unknown", CompletionOutcome.UNKNOWN_OUTCOME)))
        before = applied(Space.setGrant(before, SetGrantCommand("ask", "owner", "worker", "coding.run", "apply", Grant.ASK)))
        before = applied(Space.submit(before, submit("approved")))
        before = applied(Space.approve(before, ApproveCommand(
            operationId = "approve",
            spaceId = "space",
            hostEpoch = 1,
            approvalId = "approval",
            taskId = "approved",
            actorNodeId = "owner",
            targetNodeId = "worker",
            actionFingerprint = "digest",
            expiresAt = 10,
            approvedAt = 1,
        )))
        before = applied(Space.submit(before, submit("awaiting")))

        val after = applied(Space.recoverAfterRestart(before, recovery()))

        listOf("queued", "approved").forEach { id ->
            assertEquals(
                before.tasks.getValue(id).copy(status = TaskStatus.UNKNOWN_OUTCOME, terminalReason = RejectionReason.HOST_RESTARTED),
                after.tasks.getValue(id),
            )
        }
        listOf("completed", "failed", "unknown", "awaiting").forEach { id ->
            assertEquals(before.tasks.getValue(id), after.tasks.getValue(id))
        }
        assertEquals(before.nodes, after.nodes)
        assertEquals(before.grants, after.grants)
        assertEquals(before.advertisements, after.advertisements)
        assertEquals(before.approvals, after.approvals)
        assertEquals(before.activeHostEpoch, after.activeHostEpoch)
        assertEquals(before.activeHostNodeId, after.activeHostNodeId)
        assertEquals(before.ownerNodeId, after.ownerNodeId)
        assertEquals(before.spaceId, after.spaceId)
        before.commands.forEach { (id, recorded) -> assertEquals(recorded, after.commands[id]) }

        val events = after.auditEvents.drop(before.auditEvents.size)
        assertEquals(setOf("restart:queued", "restart:approved", "restart"), events.map { it.idempotencyKey }.toSet())
        assertEquals(3, events.size)
        events.forEach { event ->
            assertEquals(Operation.RECOVER_AFTER_RESTART, event.operation)
            assertEquals(AuditOutcome.ACCEPTED, event.outcome)
            assertEquals("host", event.actorNodeId)
            assertEquals("host", event.authorityNodeId)
            assertEquals(1, event.hostEpoch)
            if (event.idempotencyKey != "restart") assertEquals(RejectionReason.HOST_RESTARTED, event.reason)
        }
    }

    @Test
    fun `restart requires the current Host and matching Space and epoch`() {
        val before = applied(Space.submit(readySpace(), submit("queued")))
        val invalid = listOf(
            recovery().copy(spaceId = "another-space") to RejectionReason.INVALID_SPACE,
            recovery().copy(hostEpoch = 0) to RejectionReason.STALE_HOST_EPOCH,
            recovery().copy(hostEpoch = 2) to RejectionReason.STALE_HOST_EPOCH,
            recovery().copy(actorNodeId = "owner") to RejectionReason.UNAUTHORIZED_ACTOR,
            recovery().copy(actorNodeId = "worker") to RejectionReason.UNAUTHORIZED_ACTOR,
            recovery().copy(actorNodeId = "unknown") to RejectionReason.UNAUTHORIZED_ACTOR,
        )
        invalid.forEach { (command, reason) ->
            val result = assertIs<Transition.Rejected>(Space.recoverAfterRestart(before, command))
            assertEquals(reason, result.reason)
            assertEquals(before.tasks, result.state.tasks)
        }
    }

    @Test
    fun `replaying restart preserves later work and changed content collides`() {
        val first = assertIs<Transition.Applied>(Space.recoverAfterRestart(readySpace(), recovery()))
        val later = applied(Space.submit(first.state, submit("later")))
        val replay = assertIs<Transition.Applied>(Space.recoverAfterRestart(later, recovery()))
        assertEquals(first.receipt, replay.receipt)
        assertEquals(later.tasks, replay.state.tasks)
        assertEquals(later.commands, replay.state.commands)
        assertEquals(AuditOutcome.REPLAYED, replay.state.auditEvents.last().outcome)
        assertEquals(later.auditEvents.size + 1, replay.state.auditEvents.size)

        listOf(
            recovery().copy(spaceId = "other"),
            recovery().copy(hostEpoch = 2),
            recovery().copy(actorNodeId = "owner"),
        ).forEach { changed ->
            val collision = assertIs<Transition.Rejected>(Space.recoverAfterRestart(later, changed))
            assertEquals(RejectionReason.IDEMPOTENCY_KEY_REUSED, collision.reason)
            assertEquals(later.tasks, collision.state.tasks)
        }
        val nextRestart = applied(Space.recoverAfterRestart(later, recovery().copy(operationId = "next-restart")))
        assertEquals(TaskStatus.UNKNOWN_OUTCOME, nextRestart.tasks.getValue("later").status)
    }

    @Test
    fun `historical submit receipt cannot requeue recovered task and late completion is rejected`() {
        val submitted = assertIs<Transition.Applied>(Space.submit(readySpace(), submit("queued")))
        val recovered = applied(Space.recoverAfterRestart(submitted.state, recovery()))
        val replay = assertIs<Transition.Applied>(Space.submit(recovered, submit("queued")))
        assertEquals(submitted.receipt, replay.receipt)
        assertEquals(TaskStatus.QUEUED, replay.receipt.taskStatus)
        assertEquals(TaskStatus.UNKNOWN_OUTCOME, replay.state.tasks.getValue("queued").status)

        val late = assertIs<Transition.Rejected>(Space.complete(replay.state, complete("queued", CompletionOutcome.COMPLETED)))
        assertEquals(RejectionReason.INVALID_TASK_STATE, late.reason)
        assertEquals(recovered.tasks, late.state.tasks)
    }

    private fun readySpace(): SpaceState {
        val created = Space.createSpace("space", "owner", "host")
        val paired = applied(Space.pairNode(created, PairNodeCommand("pair", "owner", "worker")))
        val advertised = applied(Space.advertiseCapability(
            paired, AdvertiseCapabilityCommand("advertise", "worker", "worker", "coding.run", "apply"),
        ))
        return applied(Space.setGrant(
            advertised, SetGrantCommand("allow", "owner", "worker", "coding.run", "apply", Grant.ALLOW),
        ))
    }

    private fun recovery() = RecoverAfterRestartCommand("restart", "space", 1, "host")

    private fun submit(taskId: String) = SubmitCommand(
        operationId = "submit-$taskId",
        spaceId = "space",
        hostEpoch = 1,
        taskId = taskId,
        originNodeId = "owner",
        targetNodeId = "worker",
        capabilityId = "coding.run",
        action = "apply",
        actionFingerprint = "digest",
    )

    private fun complete(taskId: String, outcome: CompletionOutcome) = CompleteCommand(
        operationId = "complete-$taskId",
        spaceId = "space",
        hostEpoch = 1,
        taskId = taskId,
        targetNodeId = "worker",
        outcome = outcome,
    )

    private fun applied(transition: Transition): SpaceState = assertIs<Transition.Applied>(transition).state
}
