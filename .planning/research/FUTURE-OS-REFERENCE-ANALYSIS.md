# Future OS reference and reuse analysis

Review date: 2026-10-02; upstream delta checked 2026-10-05
Reviewed repository: `https://github.com/futuregene/future-os`
Reviewed revision: `907f38b046b32ed3ac795c07b641e681d8e52101` (`main`)
Purpose: identify implementation patterns and code that can inform Lumen's production system while preserving Lumen's Space and Host authority model.

## Scope and confidence

The reviewed tree contains 2,120 source, document, and image files: 1,779 code files, 326 documents, and 15 images, about 3.33 million words by the repository scanner. The largest areas are `desktop/` (720 files), `mobile/` (334), `docs/` (235), `orchestration/` (172), and `scripts/` (168). This is a source-based review of the complete current repository layout, with focused inspection of the agent/runtime, mobile/desktop/channel surfaces, RPC/crypto packages, security policy, and architecture docs. It is not a line-by-line review of every file or a runtime, device, build, or test certification. No app or test suite was run. The `skills/` directory does not contain the separately maintained Future Loop skill source, so that policy implementation was unavailable for audit.

The root project is MIT-licensed. The vendored/adapted `orchestration/loop` subtree has a separate Apache-2.0 license and upstream attribution. Any reuse must preserve the applicable notices and audit the exact dependency/license set for the copied files.

## Executive assessment

Future OS is a substantial multi-surface agent product: Rust agent service, gRPC clients, desktop backend/UI, native mobile clients, terminal UI, CLI, messaging bridges, remote pairing/sync, and a separate long-run orchestration control plane. Its strongest contribution to Lumen is proven implementation detail for session lifecycle, client outboxes, history replay, typed RPC migration, approval presentation, and encrypted node-to-node transport.

The product authority model is different. Future OS centers its Agent sessions and desktop service; its mobile remote surface connects to a Desktop-owned agent. Lumen centers a Host-owned **Space** shared across paired **nodes**. Future OS sessions, Desktop commands, relay subjects, project files, and tool permissions must therefore remain adapter inputs or client projections in Lumen. They must not become Lumen's canonical conversation, identity, memory, policy, grants, or receipt authority.

**Recommended approach:** port interaction and failure-handling patterns into Lumen-owned Go/protocol contracts; selectively reuse small MIT TypeScript UI utilities only after confirming the target client stack and data model; do not import the Future OS remote subsystem or Agent authority wholesale.

## Repository architecture

| Area | What it owns in Future OS | Lumen relevance |
|---|---|---|
| `agent/` | Shared Rust agent service, session manager, model/tool loop, run events, session persistence, prompt assembly, compaction, approvals, and sandbox adapters | High reference value for Phase 03 conversation-to-runtime turns and bounded context projection. Keep Lumen's durable Space state outside the replaceable Hermes/Future OS runtime. |
| `packages/rpc/` | Protobuf source of truth, generated Rust RPC modules, typed payloads, local transport discovery, retry policy | Strong pattern for one versioned Lumen protocol owner, stable field evolution, idempotency, and local IPC hardening. The Rust/tonic implementation itself does not fit the Go Host. |
| `desktop/` | Tauri desktop application and backend; local agent ownership; remote service/relay integration; platform sandbox and packaging | Contains mature UI, pairing, and OS integration patterns. Its Desktop-specific authority and default permission choices do not map directly to a generic Lumen node. |
| `mobile/` | Cross-platform remote client, pairing, session browsing, prompt outbox, replay, approvals, local credential storage | Valuable Phase 05/06 patterns for node enrollment, durable sends, reconnect, receipts, and approval UI. Rebind every operation to Space conversation and Host receipts. |
| `channels/` | Messaging-provider adapters and outbound delivery queue | Useful later for Phase 11 delivery retries and truthful provider receipts; channel threads must map explicitly to Space conversations. |
| `tui/`, `cli/` | Thin clients of the shared Agent RPC service | Useful example of keeping product surfaces as clients of one contract, rather than letting each surface implement its own state machine. |
| `orchestration/loop/` | Separate event-sourced long-running goal/todo/gate/lease/evidence control plane with deterministic scheduling and validation | Reference for Phase 14 automation/delegation. Lumen Phase 14 now proposes Goal/Occurrence/Lease records in canonical Host-owned Space state; reuse only algorithms after the blocking owner/license/security gate, and keep computation adapters from becoming a second ledger. |
| `packages/` TypeScript utilities | Shared thread projection, Markdown, and JSON rendering packages | Possible targeted web-client reuse after verifying license, framework compatibility, and whether the data model can be isolated behind Lumen DTOs. |
| `tests/`, `scripts/`, `docs/` | Protocol fixtures, provider tests, operational probes, architecture/security records, release validation | Reuse test-design ideas and failure cases; translate them into Lumen's Go contract tests and node/Host fixtures. |

The primary evidence for this architecture is the workspace manifest, [`README.md`](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/README.md), [`docs/architecture/packages.md`](../../docs/architecture/packages.md), and [`docs/architecture/rpc.md`](../../docs/architecture/rpc.md).

## Upstream freshness check (2026-10-05)

The upstream `main` reference was refreshed to `e3bf4da7540b00164a86a0b866e5db2d1184d165`. Compared with the previously reviewed upstream revision, the observed source delta is confined to `agent/src/skills/manager.rs` (skill reconciliation now uses a fingerprint and detects externally changed skill directories/manifests) and the separately maintained `skills` submodule pointer. This delta does not change the reviewed Host/Space authority, pairing, replay, or Future Loop recommendations above. It is a targeted changed-path check, not a new full-tree or runtime audit; the architecture/security source links below remain pinned to the fully reviewed `907f38b046b32ed3ac795c07b641e681d8e52101` source snapshot.

## Reusable implementation patterns

| Future OS pattern and source | Adaptation in Lumen | Reuse mode / phase |
|---|---|---|
| Session-scoped history loaded before a run, then incrementally persisted as user, assistant, and tool events (`agent/src/rpc/session.rs`, `session_prompt.rs`, `session/manager.rs`, `session/sqlite_store.rs`) | Space owns canonical conversation/messages/receipts. Host projects only the permitted history into each Hermes turn. Runtime session IDs are bindings/cache keys, never Space identity. | Port state and lifecycle pattern, not schema or SQLite ownership; Phase 03. |
| Run settings/context snapshot at acceptance; stable run identity; explicit duplicate, cancellation, interruption, and incomplete-output behavior (`agent/src/rpc/session_prompt.rs`, `agent/src/runtime/session_runtime.rs`, `agent/src/agent/run_loop.rs`) | Persist the accepted message, selected runtime profile, context revision, and idempotency key before dispatch. Reconcile an uncertain Hermes Run without blind resend. | Port the invariants into `internal/space`, `internal/host`, and `internal/conversation`; Phases 03/09. |
| Prompt context assembly from project context, skills, and memory files; model sidecar kept separate from user-visible text (`agent/src/rpc/prompt_helpers.rs`, `session_prompt.rs`) | Build a Host-owned context projection with source, scope, classification, version, and permission checks. Do not automatically ingest workspace files or Hermes memory into Space. | Adapt as a context-projection design; Phase 03, then Phase 04. |
| Context-window-aware compaction with durable checkpoints and preserved transcript (`agent/src/agent/run_loop.rs`, `agent/src/compaction/`, `session/sqlite_store.rs`) | Define explicit summaries with source message ranges and revision; retain the canonical transcript separately so a context window reset cannot erase Space history. | Design reference only until Lumen has evaluation and compaction requirements; Phase 03/04. |
| Mobile send outbox stores a stable command ID, checks for a receipt before retry, and fences replay across changed pairing identity (`mobile/src/remote/usePromptOutbox.ts`) | Each node persists a pending send and the Host returns one durable message receipt. Retry the same operation ID; never issue a new message after an unknown acknowledgement. | Port with Lumen protocol types; Phase 05. |
| Reconnect fetches current state/history and pages events to a fixed watermark (`mobile/src/remote/replay.ts`; `docs/internals/mobile/streaming-sync-performance.md`) | Give clients a committed-event cursor and replay through a server-declared boundary. “Connected” and “fully synchronized” must be separate states. | Port algorithm/tests; Phase 03 and 05. |
| Sync engine merges transient live items with durable history and filters duplicates by run identity (`mobile/src/remote/syncEngine.ts`) | Treat local node cache as a projection; merge by Space message/event IDs and Host revisions. Host state remains authoritative when cache and replay disagree. | Adapt to Lumen DTOs; Phase 03/05. |
| Exact approval cards and in-flight UI guards (`mobile/src/remote/useConversationController.ts`, `desktop/src/features/agent/ApprovalPrompt.tsx`) | Display the Host-issued action fingerprint, target node, expiry, and decision state; submit one resolution and render the durable Host winner. Keep clarification answers separate from approval authority. | Port UI behavior, reimplement the contract; Phase 06. |
| RPC package centralizes proto ownership; append-only field numbers and explicit compatibility/dual-write rules (`packages/rpc/`, `docs/architecture/rpc.md`) | Maintain one versioned Lumen protocol schema and generated Go/client contracts. Add compatibility fixtures for mobile/web client and Host revisions. | Strong pattern; direct Rust crate reuse is unsuitable; protocol work in Phases 03/05. |
| RPC command policy names timeout, retry, execution class, idempotency requirements; unknown mutations do not retry by default (`packages/rpc/src/command_policy.rs`) | Every mutating Space command has an idempotency key, durable receipt, timeout, cancellation semantics, and an honest uncertain outcome. | Port policy table and tests to Go; Phases 03–10. |
| Per-user Unix socket checks and Windows current-user-only named-pipe ACLs (`packages/rpc/src/transport.rs`) | Keep local operator control restricted to the owner and verify peer identity. Preserve explicit network transports as separately authenticated contracts, never implicit fallback. | Port security properties, not Rust implementation; Host service. |
| Noise pairing and AEAD records use sequence/replay protection, context-bound AAD, frame bounds, and zeroized keys (`packages/remote-crypto/src/lib.rs`) | Use only after Lumen defines enrollment authority, Host identity pinning, revocation, key rotation, replay epoch, and recovery. Check identity before accepting/persisting a peer. | Strong protocol reference; not drop-in Go code. Phase 05, independently reviewed. |
| Messaging outbox retries provider deliveries and records outcomes (`channels/src/outbox.rs`) | Preserve Space message identity across provider retries and record delivered/failed/unknown separately from the canonical message. | Adapt when a specific channel is selected; Phase 11. |
| Future Loop event ledger, deterministic should-run decision packet, gates, leases, acceptance/evidence checks (`orchestration/loop/`, `docs/architecture/loop-control-plane.md`) | Later automation can have a separate durable work ledger that references Space messages/tasks without replacing them. Keep human gates and evidence explicit. | Concepts first; assess code/Apache obligations at Phase 14. |

## Security and architecture limits not to inherit

Future OS's own [`SECURITY.md`](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/SECURITY.md) is unusually direct about limitations. It says local persistence is not offline processing: prompts, selected attachments, and included tool results go to the configured model provider. Remote traffic is encrypted end to end after pairing, but the relay still sees metadata, and the pairing envelope/refusal reasons are not fully encrypted.

The same policy documents defaults that do not meet Lumen's default-deny contract: fresh sessions can have broad `all` permission, desktop defaults to sandbox `off`, channel clients can use broad permissions, network filtering is absent, Linux currently has no seccomp filter, and the credential file is not an encrypted vault and is exempted from one hard-deny list for CLI-based skills. Sandbox guarantees vary by platform, and some protected-path checks occur after a command rather than dynamically preventing access. These are not reasons to discard the repo; they are clear boundaries for selective reuse.

Do not copy these as Lumen semantics:

- Agent session, Desktop, relay subject, or runtime Run as the canonical Space/conversation/node identity.
- Workspace `FUTURE.md` or session-local project context as accepted Space memory.
- Future OS permission levels or desktop sandbox defaults as Lumen grants.
- Open TCP, NATS relay topology, or provider-specific session history as a replacement for Host authorization and replay.
- A successful Noise handshake as sufficient owner authorization; the caller must validate and persist the expected peer identity before completing trust establishment.
- `future-loop` goals/todos as ordinary conversation messages or as a replacement for Lumen's task/approval records.

## Lumen phase mapping

- **Phase 03 — Conversation and web:** Start with Space-owned message and receipt transitions, duplicate-safe send, runtime binding, permitted context projection, event cursor/replay, and restart recovery. Borrow Future OS history-loading and replay patterns, but keep the Host as source of truth. Do not call the current one-shot test a Lumen conversation.
- **Phase 04 — Memory/context:** Introduce accepted records, provenance, correction/deletion, scope, and retention first; then project only allowed records into Hermes. Future OS context files are convenience inputs, not a memory contract.
- **Phase 05 — Node identity and sync:** Generalize pairing/outbox/replay into Lumen node identity and Host-issued enrollment/receipts. Treat Future OS Desktop as its implementation-specific peer role, not a Lumen type.
- **Phase 06 — Attention:** Reuse visible state and duplicate-submit handling; Host validates the exact action and owns the one winning resolution. A question response grants no capability.
- **Phase 07 — Node capabilities:** Reuse OS adapter and sandbox testing ideas only behind Lumen capability contracts. Node revalidates local permission; the Host selects/authorizes cross-node work; Hermes cannot choose a node or grant.
- **Phase 08 — Voice companion node:** Reuse interaction/state presentation ideas where applicable; gate capture, wake, mute, egress, playback, and stop on the actual supported node hardware.
- **Phase 09 — Integrated local product:** Combine real Hermes output, Space context/memory, messages across nodes, approvals, and receipts; exercise failure/recovery and measure the declared workload.
- **Phase 10 — Always-on Host and alpha:** Study Future OS release/update operations, but independently prove Lumen's install, boot, update, rollback, backup/restore, offline-node, and soak contracts.
- **Phase 11 — Messaging:** Adapt a channel outbox only after one provider is selected and every message/thread mapping is explicit.
- **Phase 14 — Automation/delegation:** Evaluate Future Loop's event sourcing, leases, verification gates, and evidence requirements as a separate control plane with references to Space objects.

## Recommended reuse policy

1. **Port invariants and small algorithms first.** Reimplement in Go where they touch Host authority or Lumen persistence; preserve test scenarios and failure behavior.
2. **Copy UI/utilities only after the Lumen surface stack is selected.** The MIT license permits reuse with notice, but Future OS types and state ownership must be removed at the adapter boundary.
3. **Treat cryptography and OS sandbox code as security projects.** Require independent design review, explicit identity/lifecycle contracts, and Lumen-specific tests before code reuse.
4. **Evaluate Future Loop separately.** Its own Apache-2.0 license and isolated control-plane semantics make it a later, explicit integration choice.
5. **Record provenance for every adopted file.** Preserve copyright/license notices, exact upstream commit, local modifications, transitive license inventory, and Lumen-specific acceptance evidence.

## Primary sources

All Future OS links below are pinned to the reviewed commit:

- [Repository README](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/README.md)
- [Security policy](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/SECURITY.md)
- [Agent run loop](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/agent/src/agent/run_loop.rs)
- [Session prompt/RPC handling](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/agent/src/rpc/session_prompt.rs)
- [Mobile prompt outbox](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/mobile/src/remote/usePromptOutbox.ts)
- [Mobile replay](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/mobile/src/remote/replay.ts)
- [RPC contract guide](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/docs/architecture/rpc.md)
- [Remote crypto crate](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/packages/remote-crypto/src/lib.rs)
- [Future Loop control plane](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/docs/architecture/loop-control-plane.md)
- [Third-party notices](https://github.com/futuregene/future-os/blob/907f38b046b32ed3ac795c07b641e681d8e52101/THIRD_PARTY_NOTICES.md)
