# Phase 01 baseline evidence

**Observed:** 2026-09-26

**Base:** `docs/gsd-phase01` at `ebe42cd` (`origin/main` after fetch)

**Purpose:** distinguish merged implementation, draft PR work, automated checks, and live evidence before assigning phase completion.

## Current-state ledger

| Area | State on `main` | Evidence | Remaining proof |
| --- | --- | --- | --- |
| Space authority | Implemented: identity, nodes, grants, approvals, tasks and transitions | `internal/space/model.go`, `internal/space/apply.go`, `internal/space/execution.go` | Cross-device product journey is not present on `main` |
| Durable Host state | Implemented: encrypted whole-state store with locking and atomic replacement | `internal/store/store.go`, `internal/store/envelope.go`, `internal/store/store_test.go` | Conversation-scale storage and restore decision remain experiment-gated |
| Hermes Runs | Implemented: authenticated runtime adapter and bounded task lifecycle | `internal/hermes/client.go`, `internal/host/execution.go`, `test/contract/setup_hermes_run_test.go` | Pinned-runtime lost-stream/restart and cross-machine external Runs probes remain open in `docs/PLAN.md` Phase 02 |
| Installation | Implemented: combined/external setup, artifacts, doctor and service definitions | `internal/setup/`, `cmd/lumen/`, `deploy/README.md` | Real artifact update/rollback and container-generation replacement remain open |
| Conversations and memory | Not merged into `main` | `internal/space/model.go`; [draft PR #20](https://github.com/AshwanthReddy-exe/Lumen/pull/20) adds the candidate implementation | Independent review, deployed certification, restart proof and product client journey |
| Node execution | `main` has authority records but no general remote execution fabric | `internal/space/`, `internal/host/` | PR #20's branch-only `internal/node/` proves bounded `node.status/read`, not pairing or useful file/app action |
| Android companion | Disabled shell, not a paired product client | `apps/android-host/`; `README.md` | Enrollment, synchronized conversation, approvals and voice |
| GSD | Seven codebase-map documents created in this Phase 01 branch | `.planning/codebase/` | Requirements, roadmap, state and Phase 01 plan artifacts are not yet complete |

## Verification on this branch

| Check | Result | What it establishes |
| --- | --- | --- |
| `git fetch origin main` | Passed; `HEAD` and `origin/main` both `ebe42cd` before creating `docs/gsd-phase01` | Current base was checked; it does not validate PR #20 |
| `rtk go test ./...` | Passed: 463 tests in nine packages, outside the restricted sandbox | Automated Go baseline for `main` code; no live deployment claim |
| `rtk mise run phase0-check` | Passed outside the restricted sandbox: Gradle JVM test and Swift native-schema checker | Historical cross-platform contract baseline; not current product-client proof |
| GSD map verification | Seven documents present, each over 20 lines; `gsd_run stamp-codebase-map` stamped `ebe42cd`; secret-pattern scan found no matches | Mapped source at the stated revision; manual factual review still applies |

The first sandboxed `mise run phase0-check` failed while initializing Gradle native services and writing caches. It passed after rerunning with the access its toolchain requires. Prior sandboxed Go runs likewise failed on local socket and temporary-directory permissions; the authorized run above passed. Neither sandbox failure is classified as a product defect.

## Draft PR #20 is not merged evidence

On 2026-09-26 the [GitHub PR page](https://github.com/AshwanthReddy-exe/Lumen/pull/20) showed a **draft** targeting `main` from `feat/lumen-continuity`, 26 commits, and zero PR checks. Its committed head was `0b67f81` at inspection. The PR description claims canonical conversation/memory state, a separate isolated Hermes chat runtime and a signed `node.status/read` proof. It explicitly leaves deployed certification, two-client Host/Hermes restart proof, owner-confirmed pairing, useful restricted execution, and real-device journeys open.

The existing PR worktree has uncommitted edits to chat certification, provider/model routing, control timeouts, tests and `docs/PLAN.md`, plus untracked files. Those edits are **not** treated as committed PR behavior or as validated evidence. Phase 01 must review PR #20 by committed diff and separately inventory its dirty worktree without overwriting it. The old `feat/m2-conversation-nucleus` branch is historical input, not an independent completion claim.

## Historical live evidence and limitations

`deploy/README.md` records a 2026-09-14 Azure combined Docker journey, a same-VPS external Hermes/mTLS journey, and a native macOS Host plus pinned Hermes gateway journey. These are recorded live evidence with the limitations stated there; they do not prove a second-machine external deployment, real artifact update/rollback, general mobile node operation, or the user-facing Lumen journey. The exact runtime/configuration and commands must accompany any new acceptance run.

## Phase 01 exit checks still open

- Reconcile stale implementation-status statements and links to deleted Superpowers files in `docs/PRD.md`, `docs/ARCHITECTURE.md`, `docs/DECISIONS.md`, `docs/README.md`, and Hermes research.
- Give every active requirement one phase destination and mark PR #20 work as branch-only until reviewed/merged.
- Create GSD project, requirements, roadmap, state and Phase 01 plan/verification artifacts; retain the owner review checkpoint before Phase 02.
- Do not commit planning artifacts before the owner reviews the artifact set, as required by `docs/PLAN.md`.
