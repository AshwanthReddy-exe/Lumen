# Lumen execution blueprint

Status: planning artifact, 2026-10-03. This refines [the canonical 17-phase plan](../docs/PLAN.md) and [GSD roadmap](ROADMAP.md); it does not mark unimplemented features complete. The owner chose all 17 phases, core functionality first, and one installation command followed by interactive setup. Platform coverage is earned separately for each tested configuration.

## How to execute this blueprint

Each numbered slice is a bounded GSD plan. The listed files are the intended owner or new artifact, not permission to create a parallel subsystem when an existing file already holds the contract. Before executing a slice, inspect those files, the current branch and dirty tree, and the named requirement and decision. Freeze exact paths in that slice's `NN-XX-PLAN.md` after the preceding phase's output exists. Each slice ends with a recorded command, negative case, recovery case where state changes, and a user-visible result. `PASS`, `FAIL`, `BLOCKED`, and `UNSUPPORTED` remain distinct. Never infer a later phase's success from an earlier test.

For every mutation, record actor, authority, validation, durable intent/receipt, idempotency key, timeout, cancellation, and unknown-outcome reconciliation. The Host owns Space state; Hermes remains a bounded adapter; a node rechecks local permission. Require independent security review of protocol, authorization, persistence, migration, sandbox, and cryptography changes. Run `rtk mise run phase0-check` after shared contract changes and `rtk graphify update .` after code changes. Preserve the existing dirty Phase 02 work; do not reset or overwrite it during planning.

## Current evidence and corrections

- Phase 01 is merged. Phase 02 is marked complete in GSD with one real Host-mediated NIM answer, pinned synthetic lifecycle tests, and limited chat-only containment evidence. It does not prove a product conversation, a live cross-node action, or VPS release. See [state](STATE.md) and [Phase 02 evidence](phases/02-close-the-existing-host-hermes-foundation/02-12-EVIDENCE.md).
- Phases 03–17 now have 58 executable GSD plans. They are split into 143 bounded implementation/checkpoint tasks, but source review still finds decision, test, and live-evidence gaps; use `.planning/PLANNING-COMPLETENESS-AUDIT.md` and the dated current-state delta rather than assuming file-level plans equal routine-agent readiness.
- The 11–17 plans exist and gate later implementation on platform/provider/business decisions. Do not invent implementation detail behind an unmade decision.
- The checked-in Future OS reference remains pinned at `98f7f3a3385e12d38ee7fc75bdca2cc3856cf987`; the older broad source analysis cites `907f38b...`; focused current-upstream comparison is pinned at `52328e8009817c5eca66e4461ee4cf55e23fd6c9`. Read `.planning/research/FUTURE-OS-UPSTREAM-DELTA-2026-10-04.md` before changing a Lumen sync/queue plan. Port invariants and failure tests, not Future OS authority, crypto, or permission defaults.

**Post-PR #29 source delta:** Lumen already has Host-owned Space conversation state, Hermes projection/reconciliation and runtime certification. Phase 03-01/02 must audit and qualify those merged paths instead of rebuilding them. Browser HTTP/session/TLS, a browser client, and a full restart/reconnect acceptance journey remain unimplemented; Phase 03 is not complete. See `.planning/PLANNING-CURRENT-STATE-2026-10-04.md` for the exact source/plan reconciliation and branch disposition.

## Core functionality: phases 03–10

### 03 — Continuous Space conversation

Requirements FR-04, FR-30, FR-60, FR-75. Source: existing [03-01 plan](phases/03-conversation-and-web/03-01-PLAN.md). Future OS reference: `agent/src/rpc/session_prompt.rs`, `mobile/src/remote/replay.ts`, `packages/rpc/src/command_policy.rs`.

1. **03-01 storage decision and invariant map.** Read `internal/space/{model,apply}.go`, `internal/store/`, existing preserved conversation branch, and the actual persistence API. Write `03-01-PLAN.md`/`03-01-EVIDENCE.md` with exact atomicity, growth, recovery and encryption findings. Decide snapshot extension versus transactional encrypted database by the gate in `docs/PLAN.md`; do not split receipt and intent across stores. Verify crash at commit boundaries and no plaintext artifacts.
2. **03-02 canonical reducer.** Own `internal/space/conversation.go`, focused tests, and `protocol/fixtures/conversation-v1.json`. Define conversation/message/run/event/receipt IDs, versioned state and transitions. Atomically accept user message, command receipt and execution intent. Identical retry returns the same receipt; changed payload with same ID rejects. Test concurrent sends, Unicode, crash before/after commit, and one active turn per conversation.
3. **03-03 Hermes projection and recovery.** Own `internal/conversation/{service,projection,recovery}.go`, `internal/host/conversation.go`, focused tests. Review preserved code file by file, including unresolved findings. Bind immutable profile/context revision before dispatch; persist runtime mapping and checkpoints; reconcile lost create response and independent Host/Hermes restart without blindly resubmitting. Test no runtime access to canonical authority and truthful `uncertain` state.
4. **03-04 authenticated API and replay.** Own `internal/host/conversation_http.go`, `cmd/lumen-host/main.go`, `protocol/fixtures/conversation-http-v1.json`, tests. Implement send, receipt/status, stop, snapshot and cursor pages; check owner/node scope, size, method and version before state access. Pin replay to a fixed watermark, then switch to SSE live events. Test missing/expired auth, cursor gaps, duplicate request, slow client, cancellation and reconnect.
5. **03-05 web surface and journey.** Own `apps/web/` after checking its actual stack and supported `mise` task. Show committed versus provisional text, active/waiting/failed/uncertain states, stop, reconnect and keyboard/screen-reader flow. A real provider answer survives browser close and separate Host/runtime restarts in the same conversation. Record revisions, event cursors and redacted evidence. A responsive browser client is a surface, not the Space authority.

### 04 — Accepted memory and context

Requirements FR-33, FR-36, FR-61, FR-63. Future OS prompt assembly is a source of context-shaping ideas, not accepted memory authority.

1. **04-01 memory model.** Own `internal/space/memory.go`, tests and `protocol/fixtures/memory-v1.json`. Add proposed/accepted/corrected/deleted states with source, scope, classification, retention, revision and destination. Reject unconfirmed promotion and stale-revision edits; deletion is an authority tombstone, not a UI hide.
2. **04-02 projection.** Own `internal/conversation/projection.go` and policy tests. Build each run's bounded context from allowed history and accepted memory; record included source IDs and context revision. Invalidate runtime sessions after deletion/scope revocation. Test forbidden data, revoked data, expired records, restart and destination changes.
3. **04-03 controls.** Own `internal/host/memory_http.go`, `apps/web/src/memory.tsx`, tests. Provide inspect, accept, correct, delete and export with accessible confirmation and exact revision checks; show pending proposals separately. The next answer reflects correction, while deletion is absent from all subsequent projection.
4. **04-04 evaluation.** Own `.planning/phases/04-accepted-memory-and-context-controls/04-EVIDENCE.md` and a small independent question/qrel set under `test/eval/` only if it contains no personal data. Compare scoped lexical retrieval against the current baseline. Add embeddings only after measured gain; privacy canaries must never enter a disallowed provider request.

### 05 — Node identity and continuity

Requirements FR-02, FR-05, FR-10, FR-15, FR-23, FR-24, FR-66, FR-84. Future OS reference: `mobile/src/remote/usePromptOutbox.ts`, `replay.ts`, `syncEngine.ts`, `packages/remote-crypto/`. Review cryptography independently; do not transplant its identity model.

1. **05-01 client feasibility gate.** On named physical Android and iPhone devices, measure secure key storage, background reconnect, durable outbox, notifications/approvals and native audio. Record the selected stack and tested versions in `05-EVIDENCE.md`; choose the documented fallback if the shared-client gate fails. The user wants broad portability, but only passing devices enter the support matrix.
2. **05-02 versioned protocol.** Own `protocol/enrollment-v1.json`, `protocol/command-v1.json`, fixtures and `internal/space/device.go` tests. Specify key binding, invitation expiry/one use, signature, nonce, sequence, request size, compatibility, clock skew, rotation and revocation. Pairing creates identity only, never a grant.
3. **05-03 Host enrollment.** Own `internal/host/device_identity.go`, owner API/CLI and tests. Owner confirms device fingerprint; Host commits membership and scoped credentials. Reject guessed invitation, wrong key, duplicate enrollment, stale version and revoked key; revoke blocks new Host reads immediately after commit.
4. **05-04 client sync.** Own the selected client's pairing/chat/outbox modules (draft path `apps/phone/src/`) and tests. Store pending command ID with local projection; query receipt before retrying the same ID. Replay to fixed watermark and merge by Host IDs/revisions. Label stale/offline state and reattach to an active Host run.
5. **05-05 physical recovery.** Suspend between send and acknowledgement, change networks, reorder events, skew clock, rotate/revoke key and restart app. Prove one accepted message and one receipt, or visible pending/uncertain state. Record offline-replica deletion limits.

### 06 — Attention and human input

Requirements FR-31, FR-81, FR-82. Future OS approval cards inform UI only; Host approval is exact and single use.

1. **06-01 authority records.** Own `internal/space/attention.go` and tests. Separate clarification answer from action approval. Bind approval to actor, target, capability, canonical arguments/artifact digest, grant revision, expiry and run ID. Compare-and-set one winner; reject changed action, replay, expiry, revocation and concurrent replies.
2. **06-02 Host endpoints and projection.** Own `internal/host/attention_http.go` and tests. List/detail/resolve requests under client scope, persist decision before reply delivery, and publish committed events. A question answer changes input only. Test stale browser and phone races.
3. **06-03 web/phone cards.** Own `apps/web/src/attention.tsx` and selected phone client's equivalent. Show exact action preview, target, expiry, risk and separate question form. Accessibility and offline/expired/denied states must be testable.
4. **06-04 Hermes compatibility.** Own `internal/hermes/client.go`, `internal/host/interaction.go` and tests. Qualify the pinned documented Runs request/reply identifiers with a harmless live request and lost acknowledgement/restart. If the API cannot safely resume a waiting run, mark that path `UNSUPPORTED` and keep it disabled rather than fabricating a reply endpoint.

### 07 — Cross-node capabilities and useful action

Requirements FR-11–14, FR-20–22, FR-35, FR-64–65. The first real effect is selected-root `files.search/read` on an awake target computer; later effects use the same broker.

1. **07-01 target node and manifest.** Own `apps/mac-node/` for the first tested target, `protocol/fixtures/node-v1.json`, and node tests. Pair and authenticate separately from grants. Advertise versioned capabilities/health; local stop and key storage must work before execution.
2. **07-02 local file boundary.** Own target node `Files` adapter and tests. Explicitly selected roots only; canonicalize and revalidate immediately before open, including symlinks, rename races, file types, size limits and OS permission loss. Return bounded content with source metadata; deny everything else.
3. **07-03 Host broker.** Own `internal/space/invocation.go`, `internal/host/capability_broker.go`, protocol fixtures and tests. Resolve an explicitly selected node; validate actor, grant, manifest, args, expiry and any exact approval. Persist invocation before dispatch; runtime text cannot select another node or create grants.
4. **07-04 transport and receipt.** Own `internal/host/node_transport.go`, target node connection code and tests. Use authenticated bidirectional connection, stable invocation IDs, deadline and cancellation. Target durably records outcome; Host reconciles disconnect before/after effect without repeating non-idempotent work. Source and target projections show the same Host receipt.
5. **07-05 real journey and review.** From source node, request selected target files, approve from an authorized surface and observe one bounded receipt in the same conversation. Inject offline target, stale manifest/grant, replay, node restart, target substitution and receipt loss. Independent review must close authorization and filesystem findings.

### 08 — Voice companion

Requirements FR-07, FR-40, FR-71–73, FR-83. Voice is a node/surface over the same conversation; local wake precedes audio egress.

1. **08-01 wake gate.** Own Android voice state module and physical-device evidence. Identify model/license and run the existing E6 threshold on named hardware, including negative household audio. If it fails, narrow supported conditions or mark wake unsupported.
2. **08-02 state/privacy.** Add visible and accessible idle/mute/detecting/listening/stop behavior. Prove no pre-activation audio egress, stale callback cancellation, app-suspend handling and immediate local stop.
3. **08-03 transcript/playback.** Post-wake local recognition submits exactly one canonical transcript via Phase 05 outbox. Local playback is interruptible and cannot auto-fallback to remote speech. Test echo, barge-in, duplicate transcript and revoked permission.
4. **08-04 physical E7.** Measure recognition/playback quality, latency, battery, thermals, language/noise and sustained use with a reproducible dataset and exact model/engine version. Log no raw private audio.

### 09 — Integrated local product

Requirement FR-62 local continuity; no install/release claim.

1. **09-01 fixture and script.** Own `test/scenario/personal_alpha_local_test.go` and `09-EVIDENCE.md`. Define source, approval and target node, real provider/profile, accepted memory fact, grant and selected root. State exact expected Host IDs/receipts.
2. **09-02 complete journey.** Web/phone conversation, corrected memory in next answer, exact approval, remote file read and receipt in one timeline. Verify no second dispatch on reconnect.
3. **09-03 fault matrix.** Inject Host/runtime/provider restart, source/target disconnect, revoked grant, storage full, lost receipt and browser/phone reconnect. Record true pass/fail/uncertain state for each.
4. **09-04 quality.** Measure latency and resource envelope on named machines; run accessibility, redaction and independent boundary review. Do not promote local evidence to always-on release.

### 10 — One-command installation and personal alpha

Requirements FR-03, FR-08, FR-09, FR-34, FR-62. One verified command installs a signed/digest-pinned Lumen release and launches interactive `lumen setup`; the wizard chooses combined or external Hermes, credentials, Host placement/profile and pairing. Do not silently modify an existing Hermes installation. Every claimed OS/profile needs its own live matrix row.

1. **10-01 distribution and wizard.** Own installer under `scripts/`/`deploy/`, `cmd/lumen/main.go`, setup planner/runner and contract tests. Define supported OS/architecture prereqs, artifact digest/signature, least-privilege install, rollback on interruption, idempotent rerun and actionable doctor output. Test fresh/partial install and wrong artifact. The command and hosting URL are finalized only after release provenance is chosen.
2. **10-02 supervision.** Own `deploy/systemd/`, `deploy/docker/`, macOS LaunchAgent path and setup tests. Prove actual boot/login/reboot and Host identity continuity on each claimed profile. Distinguish a logged-in macOS user agent from pre-login service behavior.
3. **10-03 updates.** Own release generation code and tests. Install second pinned version, interrupt before/after publication, offline rollback to retained bytes, preserve encrypted state and reject incompatible schema downgrade.
4. **10-04 encrypted backup/restore.** Export and validate complete encrypted Space state on a clean quarantined replacement without activating it.
5. **10-05 source handoff.** Drain and reconcile effects, then obtain exact owner confirmation; a verified pre-transfer abort restores only the original source epoch.
6. **10-06 replacement activation.** Consume the one-use handoff, persist the new Host epoch atomically, and fence the retired source on return.
7. **10-07 node epoch pins.** The selected phone and Mac accept the confirmed replacement key/epoch before effectful work resumes; stale or offline nodes stay unavailable.
8. **10-08 recovery artifacts.** Inventory and owner-prune retained encrypted recovery copies only after a verified fresh backup.
9. **10-09 alpha acceptance.** With original laptop off and Host still reachable, continue from another node; later wake/select the target and perform approved cross-node action. Run seven-day soak, resource/latency checks, requirement-by-requirement evidence, support matrix and explicit owner sign-off. Missing hardware/soak/sign-off is `BLOCKED`.

## Phase 11–17 execution index

The executable plans live under `.planning/phases/11-*` through `17-*`; plan counts vary by phase. `.planning/PHASE-11-17-CONTRACTS.md` fixes their authority, state, file and live-evidence requirements. Decision-dependent downstream plans stay blocked until the binding decision is recorded. The GSD `NN-XX-PLAN.md` files and contract catalog control execution.

## GSD coordination and handoff contract

1. At phase entry, read the applicable PRD/architecture/decision/roadmap section, `.planning/CONTRACT-CATALOG.md`, this phase's `NN-XX-PLAN.md` files and any applicable branch contract. Run `gsd-spec-phase` or `gsd-discuss-phase` only for an unresolved product choice; preserve the approved 17-phase roadmap. Use `gsd-ai-integration-phase` for Hermes/memory/voice eval contracts and `gsd-ui-phase` for new UI surfaces when their stated skill gates apply.
2. Run `gsd-plan-phase` and `gsd-plan-review-convergence` for each phase before execution. Current source-grounded reviews are useful inputs but are **not** the required external cross-AI convergence. Record reviewer availability and unresolved HIGH/MEDIUM items honestly. Do not call a phase converged until an independent reviewer accepts the revised plans.
3. For an implementation task, freeze input SHA, resolve gate decisions, confirm owned files and tests, then execute one dependent plan at a time with `gsd-execute-phase`. Parallel workers may touch disjoint files only; shared schema, reducers, auth, migrations and release records stay serial. Run `gsd-code-review`, `gsd-secure-phase` for security boundaries, `gsd-verify-work`, and `gsd-validate-phase`/`gsd-eval-review` where relevant. Record actual tests and physical/provider evidence in phase evidence; no simulated pass substitutes for live acceptance.
4. Update `STATE.md` and requirement verdicts only after verification. Phase 10 alpha requires actual installation, powered-off-primary cross-node journey, recovery, seven elapsed days of soak and owner sign-off. Phases 15–17 require their separate commercial/public/shared-Space decisions. Use `gsd-audit-uat` and `gsd-audit-milestone` before release claims. Do not run execute or ship merely because planning files exist.

Every plan names `files_modified`, `<files>`, required behavior, action, check, evidence and output summary. Its check is one piece of proof; owner/platform/provider gates remain `BLOCKED` until the named evidence exists. If a selected branch changes file paths, rewrite that plan before implementation; do not silently implement the wrong branch.
