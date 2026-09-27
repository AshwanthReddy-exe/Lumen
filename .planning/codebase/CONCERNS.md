---
last_mapped_commit: ebe42cda0332c603644874b73ff920964ba853f1
last_mapped_at: 2026-09-26
---
# Codebase Concerns

**Analysis Date:** 2026-09-26

## Tech Debt

**Repository status and planning authority:**

- Issue: The current checkout contains uncommitted Phase 01 planning edits and earlier cleanup deletions (`.superpowers/sdd/...` and `docs/superpowers/...`). The master plan previously mixed “no feature implementation yet” with the implemented Go foundation; Phase 01 is reconciling that wording and distinguishing `main` from draft PR20. The current branch is `docs/gsd-phase01` at `ebe42cd`.
- Files: `docs/PLAN.md`, `.superpowers/sdd/2026-09-10-lumen-owned-setup/task-7-report.md`, `docs/superpowers/plans/`, `docs/superpowers/specs/`.
- Impact: Readers can mistake planned work, current `main` code, historical evidence, and unmerged PR20 work as the same state. Deleted planning records may be user-authorized cleanup and must not be restored accidentally.
- Fix approach: Reconcile claims against the current base and evidence inventory in Phase 01; retain the cleanup deletions and replace stale links with GSD navigation (`docs/PLAN.md:381-400`).

**Historical verification claims:**

- Issue: This map was written before the Phase 01 check run. The 463-test Go baseline and `phase0-check` were subsequently rerun and recorded in `.planning/research/BASELINE-EVIDENCE.md`; the deployment evidence summarized in `docs/PLAN.md` remains historical and has not been rerun here.
- Files: `docs/PLAN.md:36-57`, `mise.toml:38-72`.
- Impact: A past test pass or a runnable script can be mistaken for present release acceptance.
- Fix approach: Track test output and live evidence as dated artifacts tied to commit, platform, runtime identity, and acceptance criterion; label unavailable hardware/credentials blocked or unverified as required by `docs/PLAN.md:379`.

## Known Bugs

**No confirmed runtime bug established by this mapping.**

- Symptoms: Not detected through an execution pass; this work was read-only.
- Files: Relevant risk and validation surfaces are `internal/store/store.go`, `internal/host/execution.go`, `internal/hermes/client.go`, and `docs/THREAT_MODEL.md`.
- Trigger: Not established.
- Workaround: Keep claims at the level of static evidence until the documented checks are run.

## Security Considerations

**PR20's dirty real-provider routing change expands the Hermes credential boundary:**

- Risk: On the PR20 branch, uncommitted `internal/host/chat_docker.go:192-229` reads a real provider key and injects it into both `OPENAI_API_KEY` and `OPENROUTER_API_KEY`; the new untracked `scripts/lumen-mac-real-chat-check:22-44,76-120` reads the Hermes-managed route/key, writes temporary key/environment files, and launches the chat container. This can expose the key to the configured provider or an unintended endpoint if endpoint policy/resolution/egress is insufficient.
- Files: PR20 worktree only: `internal/host/chat_docker.go`, `internal/host/service.go`, `integrations/hermes/container_chat_probe.py`, `scripts/lumen-mac-real-chat-check`. These edits are uncommitted in the separate `feat/lumen-continuity` worktree; they are not part of this `main` checkout.
- Current mitigation: The dirty certifier checks HTTPS and excludes empty hostnames, localhost, and `127.*` (`internal/host/chat_docker.go:208-229`); provider URL is compared with configured route; container checks require a read-only root filesystem, unprivileged user, loopback port publishing, and restricted mounts (`:129-160`). These checks do not by themselves establish DNS resolution safety, rejection of all private/link-local/metadata destinations, or network egress restrictions.
- Recommendations: Independent security review before reuse. Prove endpoint allowlisting/resolution policy, redirect behavior, network egress controls, credential visibility and rotation, and why the same credential is needed in both provider variables. Treat this as an open risk to validate, not a proven exploit. It aligns with blocking trust-boundary requirements T-005 and T-018 in `docs/THREAT_MODEL.md:83,96`.

**Hermes zero-tool and runtime-certification claims:**

- Risk: PR20's committed conversation implementation exists only on `feat/lumen-continuity`, not current `main`; the PR20 worktree is also dirty. Synthetic no-tool probes and package tests cannot independently prove production containment or bind the deployed process/configuration/endpoint to a Host certification.
- Files: PR20 branch: `internal/conversation/service.go:127-232`, `internal/host/chat_docker.go:72-160`, `integrations/hermes/container_chat_probe.py`; current plan gate `docs/PLAN.md:402-419`.
- Current mitigation: PR20 conversation flow checks certification and endpoint identity before dispatch and re-certifies after projection (`internal/conversation/service.go:186-215`); the Docker certifier inspects process/image/config and environment (`internal/host/chat_docker.go:119-229`). The current `main` state inventory explicitly says conversations and memory are absent (`docs/PLAN.md:49`).
- Recommendations: Review PR20 committed and dirty diffs separately, then require a production Host certifier plus intended deployment/device evidence before changing current-main or release-status claims. The PR20 dirty `docs/PLAN.md` addition records a provider HTTP 401 and failed assistant outcome, not successful live chat.

## Performance Bottlenecks

**Whole-state encrypted snapshot updates:**

- Problem: Every `Store.Update` reads/decrypts the current state and commits a replacement snapshot (`internal/store/store.go:180-212,215-235,270-367`).
- Files: `internal/store/store.go`; current state fields are aggregated in `internal/space/model.go:74-92`.
- Cause: The store serializes all Space authority state into one bounded envelope rather than independently indexed records.
- Improvement path: This is an acknowledged alpha boundary, not an established defect. Follow the bounded SQLite-versus-snapshot experiment and crash, backup, privacy, and volume criteria in `docs/PLAN.md:160-174` before selecting a migration.

## Fragile Areas

**External runtime evidence and ambiguous outcomes:**

- Files: `internal/host/execution.go`, `internal/hermes/client.go`, `docs/THREAT_MODEL.md:80-90,96`.
- Why fragile: Runtime calls and event streams cross process/network boundaries and can fail after dispatch; success must come from durable Host state, not a model response or untrusted event.
- Safe modification: Preserve durable intent before I/O, idempotent receipts, bounded reconciliation, and explicit `unknown_outcome`. Existing targeted examples include `internal/host/execution_test.go:141-163,165-184,186-255,258-290`.
- Test coverage: Broad unit/contract tests exist, but this mapping did not run them; live restart and external-runtime evidence remains a separate Phase 02 gate (`docs/PLAN.md:402-419`).

**Concurrent persistence and recovery:**

- Files: `internal/store/store.go`, `internal/store/store_test.go`.
- Why fragile: The store combines file locking, strict encrypted envelope validation, atomic replacement, directory sync, backup rollback, and recovery evidence.
- Safe modification: Retain fault injection through `NewWithHooks`; assert state before/after injected write, sync, rename, cleanup, and path replacement failures (`internal/store/store_test.go:220-260,313-390,474-550`).
- Test coverage: The existing tests cover many persistence failure paths; backup/export/restore and migration acceptance remain open in `docs/THREAT_MODEL.md:89-90`.

## Scaling Limits

**Single encrypted snapshot:**

- Current capacity: Envelope reads are limited to 8 MiB (`internal/store/store.go:228-235`); no measured alpha-volume threshold was established in this map.
- Limit: Every state transition decodes and rewrites the complete state (`:180-212,270-367`), so growth in conversations/history can raise write cost and approach the envelope bound.
- Scaling path: Measure declared alpha volumes and recovery before migrating; see `docs/PLAN.md:160-174`.

## Dependencies at Risk

**Hermes runtime contract:**

- Risk: Lumen relies on a separately versioned Hermes API, runtime behavior, and deployment artifact; upstream support differences can affect event, approval, stop, and containment behavior.
- Impact: Adapter assumptions can invalidate a certification or leave work uncertain.
- Migration plan: Pin and compatibility-test supported behavior; keep Hermes behind `internal/hermes/` and follow the runtime gates in `docs/PLAN.md:402-419` and threat T-018 at `docs/THREAT_MODEL.md:96`.

## Missing Critical Features

**Current `main` versus PR20:**

- Problem: The current `main` baseline has no conversations or memory (`docs/PLAN.md:49`); the implementation is on the unmerged `feat/lumen-continuity` branch and contains uncommitted edits in its worktree.
- Blocks: Do not claim PR20 features as current-main behavior or move to dependent pairing/cross-node acceptance until the branch is reviewed and gates are met.

## Test Coverage Gaps

**Deployment/recovery and security acceptance:**

- What's not tested: This mapping did not establish fresh live install, restart/reconciliation, cross-machine Hermes Runs, provider-success chat, or device-level journeys.
- Files: `mise.toml:38-72`, `docs/PLAN.md:402-419`, `docs/THREAT_MODEL.md:79-97`.
- Risk: Unit/contract pass could be mistaken for operational, runtime-containment, or device acceptance.
- Priority: High for Phase 02 release gates; keep explicitly blocked/unverified until evidence exists.

---

*Concerns audit: 2026-09-26*
