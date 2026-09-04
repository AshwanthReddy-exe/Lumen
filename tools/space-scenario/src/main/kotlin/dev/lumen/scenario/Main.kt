package dev.lumen.scenario

import dev.lumen.core.ApproveCommand
import dev.lumen.core.CompleteCommand
import dev.lumen.core.CompletionOutcome
import dev.lumen.core.Grant
import dev.lumen.core.PairNodeCommand
import dev.lumen.core.RejectionReason
import dev.lumen.core.SetGrantCommand
import dev.lumen.core.Space
import dev.lumen.core.SpaceState
import dev.lumen.core.SubmitCommand
import dev.lumen.core.TaskStatus
import dev.lumen.core.Transition

private const val SPACE = "space-demo"
private const val ANDROID = "node-android-host"
private const val MAC = "node-mac"
private const val IPHONE = "node-iphone-owner"
private const val CAPABILITY = "coding"
private const val ACTION = "run-tests"
private const val FINGERPRINT = "fp-run-tests"

fun main() {
    var state = Space.createSpace(SPACE, IPHONE, ANDROID)
    state = apply(Space.pairNode(state, PairNodeCommand("pair-mac", IPHONE, MAC))).state

    state = apply(
        Space.advertiseCapability(
            state,
            dev.lumen.core.AdvertiseCapabilityCommand("advertise-mac", MAC, MAC, CAPABILITY, ACTION),
        ),
    ).state

    state = applyGrant(state, "grant-deny", Grant.DENY)
    expectRejected(
        Space.submit(state, submit("submit-deny", "task-deny")),
        RejectionReason.GRANT_DENIED,
    )

    state = applyGrant(state, "grant-ask", Grant.ASK)
    val awaiting = apply(Space.submit(state, submit("submit-ask", "task-ask")))
    check(awaiting.receipt.taskStatus == TaskStatus.AWAITING_PERMISSION) { "ask did not await permission" }
    state = awaiting.state

    val approved = apply(
        Space.approve(
            state,
            ApproveCommand(
                operationId = "approve-ask",
                spaceId = SPACE,
                hostEpoch = state.activeHostEpoch,
                approvalId = "approval-ask",
                taskId = "task-ask",
                actorNodeId = IPHONE,
                targetNodeId = MAC,
                actionFingerprint = FINGERPRINT,
                expiresAt = 2_000L,
                approvedAt = 1_000L,
            ),
        ),
    )
    check(approved.receipt.taskStatus == TaskStatus.QUEUED) { "approval did not queue task" }
    state = approved.state
    state = apply(
        Space.complete(
            state,
            CompleteCommand("complete-ask", SPACE, state.activeHostEpoch, "task-ask", MAC, CompletionOutcome.COMPLETED),
        ),
    ).state
    expectRejected(
        Space.approve(
            state,
            ApproveCommand("approve-ask-retry", SPACE, state.activeHostEpoch, "approval-ask", "task-ask", IPHONE, MAC, FINGERPRINT, 2_000L, 1_000L),
        ),
        RejectionReason.APPROVAL_ALREADY_CONSUMED,
    )

    state = applyGrant(state, "grant-allow", Grant.ALLOW)
    val allowCommand = submit("submit-allow", "task-allow")
    val queued = apply(Space.submit(state, allowCommand))
    check(queued.receipt.taskStatus == TaskStatus.QUEUED) { "allow did not queue task" }
    val replay = apply(Space.submit(queued.state, allowCommand))
    check(replay.receipt == queued.receipt) { "idempotent submit changed its receipt" }
    state = apply(
        Space.complete(
            replay.state,
            CompleteCommand("complete-allow", SPACE, replay.state.activeHostEpoch, "task-allow", MAC, CompletionOutcome.UNKNOWN_OUTCOME),
        ),
    ).state
    check(state.tasks["task-allow"]?.status == TaskStatus.UNKNOWN_OUTCOME) { "uncertain completion was not preserved" }

    println("space scenario passed: deny, ask+approval, allow, idempotency")
}

private fun submit(operationId: String, taskId: String) = SubmitCommand(
    operationId = operationId,
    spaceId = SPACE,
    hostEpoch = 1L,
    taskId = taskId,
    originNodeId = IPHONE,
    targetNodeId = MAC,
    capabilityId = CAPABILITY,
    action = ACTION,
    actionFingerprint = FINGERPRINT,
)

private fun applyGrant(state: SpaceState, operationId: String, grant: Grant): SpaceState = apply(
    Space.setGrant(state, SetGrantCommand(operationId, IPHONE, MAC, CAPABILITY, ACTION, grant)),
).state

private fun apply(transition: Transition): Transition.Applied = when (transition) {
    is Transition.Applied -> transition
    is Transition.Rejected -> error("scenario invariant failed: unexpected rejection ${transition.reason}")
}

private fun expectRejected(transition: Transition, reason: RejectionReason) {
    check(transition is Transition.Rejected && transition.reason == reason) {
        "expected rejection $reason, got $transition"
    }
}
