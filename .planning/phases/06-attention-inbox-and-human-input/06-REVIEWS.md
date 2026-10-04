---
phase: 06
reviewers: []
review_status: manual-source-review-external-pending
---

# Phase 06 plan review

External GSD review remains pending; this is a source-grounded manual review. The external Claude CLI probe hung and was interrupted, and the Codex CLI is not independent of the current agent.

## 06-01

- **HIGH:** The approval transition must tie exact action, args digest, grant revision, actor, node, expiry and consumption to the existing Host authority path. `internal/host/execution.go:733` is the present durable command boundary; the plan names `internal/host/approval.go` without specifying how every command caller must pass through it. Define the insertion point and negative tests at the shared boundary.

## 06-02

- **HIGH:** Pause/resume across restart is underspecified for lost reply acknowledgement. Specify persisted state transitions, idempotency key, reconcile operation and `uncertain` outcome where the Hermes reply API cannot prove acceptance. FR-30 requires honest unknown outcomes (`.planning/REQUIREMENTS.md:31`).

## 06-03

- **HIGH:** Both web and phone tasks verify only Go Host tests (`06-03-PLAN.md:54`, `:60`), leaving the listed TSX tests unrun. Add stack-specific commands and a real device/browser acceptance path.
- **MEDIUM:** Phone card paths inherit the unresolved D-056 choice (`docs/DECISIONS.md:55`). Gate exact file ownership on Phase 05's physical-device decision.

## 06-04

- **HIGH:** The task correctly forbids guessing a Hermes reply endpoint, but has no explicit blocked outcome if the pinned runtime exposes no safe reply-by-ID or status lookup. Define unsupported/uncertain behavior and prohibit enabling approval delivery until the live contract is certified.

## CYCLE_SUMMARY

Status: **not converged**. Unresolved actionable findings: 4 HIGH, 1 MEDIUM. Replan state transitions and actual client checks, then rerun independent GSD review.
