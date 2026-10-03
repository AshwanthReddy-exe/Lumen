---
phase: 03
reviewers: []
review_status: manual-source-review-external-pending
---

# Phase 03 plan review

External GSD reviewer did not complete; this is a source-grounded manual review, not cross-AI convergence. `claude -p 'Reply READY only.' --model haiku --tools ''` returned no output after 30 seconds and was interrupted. The Codex lane is the current agent's own CLI and cannot count as independent under the GSD review workflow.

## 03-01

- **HIGH:** E4 is a conditional architecture gate, yet the plan lets the same task choose between keeping encrypted snapshots and migrating the whole authority store. `internal/store/store.go:180` already exposes a single `Update` transaction boundary; define measured volume, latency, fault thresholds, and an explicit decision artifact before a basic executor may alter persistence. The current task does not tell the executor what result constitutes pass or fail.
- **MEDIUM:** `internal/space/conversation.go` is asked to define several state types and a transition, but no field-level schema or transition table is provided. Pin allowed states, sequence allocation, duplicate-key retention, and active-turn release on uncertain/cancelled outcomes.

## 03-02

- **HIGH:** Reconciliation says to query Hermes by idempotency key, but the plan gives no pinned endpoint or fallback when the pinned runtime cannot do that. Require a capability check and an explicit `uncertain` transition without retry when the query is unavailable; record mapping before dispatch. This is central to FR-30 (`.planning/REQUIREMENTS.md:31`).
- **MEDIUM:** `internal/conversation/service.go`, `projection.go`, and `recovery.go` all say they provide the same outcome. Assign exact functions, input/output shapes, and persistence owner so a small model can implement without rediscovering boundaries.

## 03-03

- **HIGH:** Existing operator authentication is via a local credential and Unix socket (`internal/host/service.go:49`, `cmd/lumen-host/main.go:137`). The browser API plan says to reuse operator credential validation but does not define a safe browser session, CSRF policy, origin policy, or transport exposure. Decide these before implementing HTTP routes; never place the operator credential in browser storage.
- **MEDIUM:** Replay names fixed watermark and SSE, but the handoff from snapshot to subscription is unspecified. Pin event IDs, cursor-too-old response, subscription registration order, and a race test across the watermark boundary.

## 03-04

- **HIGH:** Both UI tasks verify only `go test ./internal/host`; this cannot test `apps/web/src/conversation.test.tsx`. Define an actual web test/build command after selecting the scaffold and make it the task verification.
- **MEDIUM:** The browser scaffold introduces dependencies without naming a supported runtime/package manager. Use the actual repository toolchain decision and lockfile, or add a decision gate before this task.

## 03-05

- **MEDIUM:** The final journey must record real provider/Host/browser evidence separately from Go contract tests. Do not mark user-visible chat complete from test doubles or direct Hermes calls, consistent with the Phase 02 limitations in `docs/PLAN.md:408`.

## CYCLE_SUMMARY

Status: **not converged**. Unresolved actionable findings: 4 HIGH, 4 MEDIUM. Replan 03-01 through 03-04 and run an independent GSD external review afterward. This artifact is not a successful GSD reviewer result.
