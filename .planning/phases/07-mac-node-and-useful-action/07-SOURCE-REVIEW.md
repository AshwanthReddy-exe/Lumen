# Source-grounded plan review — Phase 07

Status: **OPEN; not cross-AI converged.** Claude CLI produced no result in a bounded attempt; this is an inline source review, not `gsd-review` output.

## 07-01

- **HIGH:** The selected-root file path promises race-safe reads, but names no exact macOS file descriptor procedure or representation of a persistent selected root (`07-01-PLAN.md:20`). Specify root selection/bookmark persistence, `openat`-style traversal, inode checks, and permission-loss handling before a basic executor implements it. The node package does not yet exist, so no code supports this claim.
- **MEDIUM:** Local grants are mentioned on the node (`07-01-PLAN.md:13,20`) while pairing explicitly creates no grant (`07-01-PLAN.md:19`). Define the node-local grant record and its relationship to the Host grant; a signed Host invocation alone must not widen it.

## 07-02

- **HIGH:** The invocation record lists an argument digest but not a canonical argument encoding or digest algorithm (`07-02-PLAN.md:19`). Define both in `protocol/fixtures/invocation-v1.json`, including rejection of noncanonical equivalents, or duplicate detection can disagree across Go and Swift.
- **MEDIUM:** “Authenticated user picks target” is stated (`07-02-PLAN.md:19`), but the plan does not identify the authenticated client request field or its ownership. Tie target selection to a concrete command contract and ensure the Hermes parser cannot supply it (`07-02-PLAN.md:20`).

## 07-03

- **HIGH:** Target receipt durability is promised (`07-03-PLAN.md:19`) without naming the Mac durable store, transaction boundaries, or retention/replay window. Add receipt schema and storage file ownership before dispatch; otherwise restart recovery is underspecified.
- **MEDIUM:** `test/scenario/node_disconnect_test.go` is also owned by `09-01-PLAN.md:7`. State that Phase 09 extends the Phase 07 test and retains its assertions to avoid an executor replacing the earlier matrix.

CYCLE_SUMMARY: unresolved HIGH 3; actionable MEDIUM 3; LOW 0. Revise plans, then run independent `gsd-review` again.
