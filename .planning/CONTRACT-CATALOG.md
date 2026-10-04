# Lumen contract and file catalog

Status: proposed execution contract for Phases 03–17, 2026-10-03. [PRD](../docs/PRD.md), [architecture](../docs/ARCHITECTURE.md), [decisions](../docs/DECISIONS.md), and [master plan](../docs/PLAN.md) remain authoritative. A proposed function is an implementation target, not a claim that it exists. Check names and signatures against current code before editing. The repository currently uses `internal/store`, `internal/space`, `internal/host`, and `internal/hermes`; do not create a second authority layer.

## Structural rules

```text
cmd/lumen/                 owner CLI: setup, doctor, releases, backup/restore
cmd/lumen-host/            foreground Host process and service wiring
internal/space/            pure Space command/reducer/policy/state types
internal/store/            encrypted durable commits, migration, export input
internal/host/             authenticated API, orchestration, broker, projections
internal/hermes/           pinned Runs adapter and event normalization
internal/setup/            install/adopt, journal, supervisor, release lifecycle
internal/conversation/     bounded Host-owned context and run projection (Phase 03)
internal/integrations/     typed provider bridges after Phase 11 gate
internal/automation/      deterministic automation computation/adapters only; canonical ledger remains in internal/space and Store
internal/managed/         dedicated-customer provisioner after Phase 16 gate
protocol/fixtures/        versioned wire and cross-language conformance cases
apps/web/                 Host client, no canonical state (Phase 03)
apps/phone/               selected stack only after physical Phase 05 gate
apps/mac-node/            first target implementation, no Host authority
apps/android-host/        existing companion; voice added in Phase 08
test/contract/            deterministic cross-component contracts
test/scenario/            real product journey and failure scenarios
test/eval/                independent retrieval/voice datasets where permitted
deploy/                   profile-specific release and supervision artifacts
scripts/                  bounded release and machine qualification scripts
```

Do not create all folders at once. Each phase creates only its required paths after the relevant stack/provider decision. Shared protocol types are independent of Go, web, mobile, Hermes, transport, and storage. Preserve the existing one-file encrypted snapshot until the Phase 03 storage gate has measured a replacement and proven encrypted migration.

## Authority and durable records

The table is a minimum schema checklist. Names are provisional; existing `State`, `Command`, `Task`, `HostRun`, and `Receipt` in `internal/space/model.go` should be extended instead of copied. Any record that authorizes work has a version and Host epoch. External runtime/provider IDs are mappings, never canonical Space IDs.

| Record | Owner and minimum fields | Invariants and index |
|---|---|---|
| `Conversation` | Host: id, Space id, owner/participants, created/updated revision, active turn id, retention | One active turn initially; stable across surfaces and runtime swaps |
| `Message` | Host: id, conversation id, author/surface/node, command id, ordered sequence, content/classification, committed state, timestamps | Accepted user message and command receipt commit together; final assistant content is separate from provisional tokens |
| `Run` | Host: id, conversation/message id, status, context revision, immutable profile digest, deadline/budget, checkpoint, event cursor, stop intent | One terminal outcome; client disconnect does not cancel; no effect after expired/cancelled authority |
| `RuntimeBinding` | Host: run id, adapter/version, opaque Hermes run/session id, create idempotency key, reconcile deadline | Persist create intent first; unknown create response reconciles by stable key without blind resubmit |
| `CommandReceipt` | Host: command id, payload digest, actor, accepted revision, subject id, outcome | Same id + same digest returns same receipt; same id + different digest rejects |
| `ConversationEvent` | Host: monotonically ordered cursor, run/conversation id, type, committed revision, safe payload | Replay pages end at fixed server watermark; live stream is only a hint |
| `MemoryProposal/Record` | Host: id, source/provenance, scope, classification, destination, retention/expiry, revision, confirmation, tombstone | Only accepted records project; deletion/scope revocation invalidates later projections |
| `Node` | Host: id, public key/fingerprint, enrollment/rotation/revocation revision, credential epoch, last health | Pairing grants no capability; revoked key cannot read or dispatch |
| `Grant` | Host: actor/node/capability/action/resource/data/time scope, deny/ask/allow, revision, expiry, offline permission | Default deny; adapter cannot widen; target rechecks OS permission |
| `Approval/Interaction` | Host: run/task, exact action digest or question id, actor, target, expiry, resolution, consume/delivery state | Clarification never grants authority; exact approval has one winner and one use |
| `Invocation` | Host: id, actor/source, required conversation id, optional source message id, selected target, capability/version, canonical args digest, grant/approval revision, deadline, status, dispatch attempt | Verify conversation participation; persist before send; never silently retarget; uncertain effect is not automatically repeated |
| `ExecutionReceipt` | Host plus target evidence: invocation id, target epoch, output digest/reference, observed effect, final/unknown state | Distinguish observed completion from request acceptance and delivery |
| `ArtifactReference` | Host: encrypted object id, digest, MIME, source, scope, retention, size | No raw artifact in ordinary audit/log/prompt; bounded delivery |
| `AutomationOccurrence` | Host canonical Space record: schedule id/version, nominal time/zone, occurrence id, lease, budget, outcome | Duplicate ticks map to one occurrence and one effect intent; `internal/automation` computes but does not own canonical persistence |

For all mutating records, commit audit metadata with the state transition. If the current store's full-snapshot rewrite cannot meet throughput/atomicity, Phase 03 must choose an encrypted transactional path with a tested migration, not quietly add a second database for a subset of authority.

**Phase 07 invocation handoff.** An `ask` commits the exact pending invocation and Phase 06 approval request in one Host update; acceptance consumes the approved request once and advances that same invocation ID after revalidation. The Host commits redacted invocation and final receipt events under its verified source conversation ID before publishing, so snapshot and fixed-watermark replay show the same receipt across surfaces. A mapped Hermes Run supplies actor and conversation from its Host binding; only a pinned documented typed callback may enter the same broker. A parser without a supported callback is `UNSUPPORTED`, never an action path. The Mac entrypoint exposes owner-console local search/read and one-time local `ask` confirmation; local execution never borrows Host authority.

## State machines and legal transitions

The executor must write a transition table and negative tests before implementing each machine. `uncertain` is a truthful state requiring reconciliation or explicit owner resolution. UI labels may differ, but must not hide uncertainty.

| Machine | Legal progression | Terminal/recovery rules |
|---|---|---|
| Message | `accepted → queued → running → completed/failed/cancelled/uncertain` | A duplicate command returns prior receipt; final assistant message commits once |
| Continuous Run | `accepted → dispatching → running ↔ waiting_for_input/waiting_for_approval → cancelling → cancelled`; `running → completed/failed/uncertain` | Restart reconstructs from Host checkpoint and runtime status; no terminal-to-running transition |
| Runtime submission | `intent_recorded → submitted → mapped → observed`; `submitted → uncertain → reconciled/expired_unknown` | Submission timeout cannot create a new key; only supported Hermes status/idempotency evidence resolves it |
| Enrollment | `invited → key_proposed → owner_confirmed → active → rotating/revoked/expired` | Invite one use; approval of key and identity before credential issuance; revocation fences future Host reads |
| Node connection | `disconnected → authenticated → replaying → synchronized → degraded/disconnected` | Connection readiness is not synchronization; resume from durable cursor and fixed watermark |
| Attention | `pending → resolved_yes/resolved_no/answered/expired/cancelled → delivered/undeliverable` | Approval and question use disjoint legal resolutions; compare-and-set one winner |
| Invocation | `authorized → queued → dispatched → running → completed/failed/cancelled/uncertain/unavailable` | Target locally revalidates before effect; receipt loss recovers by ID; no silent alternate target |
| Voice | `idle → detecting → listening → understanding → thinking → acting/approval/speaking → idle` with `muted/interrupted/offline/degraded` from active states | No pre-wake egress; stop invalidates callbacks and interrupts playback |
| Install generation | `downloaded → verified → staged → validated → published → active`; interrupted stages resume/rollback | No unvalidated auto-start; keep previous bytes for offline rollback; schema downgrade is separately gated |
| Automation | `previewed → approved → scheduled → claimed → running → waiting/complete/failed/uncertain/cancelled` | Lease and grant revalidated at each effect; descendants cancel with parent |

Persist wall time for display/expiry plus monotonic or logical revision/sequence for ordering where applicable. Network messages carry schema version, Space/Host epoch, sender/recipient, operation id, issued/expiry, nonce/sequence, signature and bounded payload. Reject unknown critical fields/versions rather than reinterpret them.

## Phase 03 fixed choices and gates

**E4 workload and storage decision.** Benchmark on a named target Host with one owner, five enrolled-node records, two concurrently active conversations, 10,000 committed messages and 1,000 memory-sized records, the dimensions already declared in `docs/PLAN.md`. Record plaintext-encoded and encrypted envelope sizes against the current 8 MiB read limit in `internal/store/store.go`; exceeding it is a gate failure, not a successful snapshot benchmark. Use the same generated, nonpersonal corpus for encrypted snapshot and any transactional candidate. Crash-inject before/after state write, rename and directory sync; scan temp, journal, backup and index paths for plaintext. Keep the existing `internal/store.Store.Update` snapshot path only if the full state is readable, every injected case yields one old-or-new committed state, and p95 Host acceptance is under 100 ms. If any hard gate fails, stop Phase 03 implementation and write a separate whole-authority migration or justified limit-change decision/plan with memory bounds; do not split conversation records into another store or silently choose a database. A migration plan must include ciphertext-at-rest verification, portable Linux/macOS builds, complete record conversion, backup, restore and no older-binary downgrade. The owner reviews any one-way migration before it runs.

**Conversation command.** Use a canonical `commandId` scoped to Space and actor; digest canonical schema-versioned payload bytes, not display text. A committed send allocates one per-conversation sequence and one Host event cursor, stores user message, receipt and run intent in the same `Store.Update`, and marks the turn `accepted`. A durable completed/failed/cancelled transition releases the active turn only after the outcome is known; `uncertain` keeps the slot occupied until explicit reconciliation or owner resolution. Reject a second effectful turn while uncertain. Repeated `commandId` and same digest returns the original receipt; changed digest rejects. Retain command ID/digest at least as long as the referenced message exists, including backup/restore, so replay cannot recreate a message.

**Run projection functions.** `StartTurn` reads one committed intent and permitted context revision; it persists Hermes create intent and immutable profile digest before I/O. `ObserveRuntime` validates mapped run ID/event order and commits bounded progress/final message before publishing its Host event. `ReconcileRun` first checks a persisted runtime mapping, then uses only a pinned documented status/idempotency capability. If lookup is unavailable or inconclusive, persist `uncertain` and do not create again. `RequestStop` commits stop intent, sends cancel by mapped ID where supported, and reconciles; cancellation request alone is never a terminal receipt. Current `internal/host/execution.go` and `internal/hermes/client.go` are the call-path analogues; Phase 03 must reuse their certification and recovery policy rather than bypassing them.

**Browser authentication decision.** The existing local Unix-socket operator credential remains server-only. Phase 03 adds a loopback HTTP listener for Host web routes and a private Tailscale Serve HTTPS reverse-proxy profile; Host never binds a non-loopback address. Serve terminates TLS and restricts reachability under tailnet access rules; its `Tailscale-User-Login` header is only an identity hint, not Lumen authentication or proof of one device, and a forged local header never authorizes. Public Funnel/sharing is disabled and rejected by the smoke check. The owner CLI uses the Unix control socket to mint a one-use, short-lived browser exchange code bound to the intended actor and client slot; the browser posts it to a same-origin exchange route over Serve HTTPS and receives a distinct short-lived, HttpOnly, Secure, SameSite session. Local loopback browser use must use the same TLS profile or be declared unsupported; do not weaken the Secure cookie. Session IDs are random, stored only as digests with actor, client slot, expiry and revocation revision in Host state; owner CLI can revoke one/all sessions and rotation invalidates them. No operator secret, exchange code or Hermes credential enters JavaScript storage, URLs or logs. Mutations require same-origin plus CSRF token/header; check `Origin` and `Host`, bound request body and per-actor rate. Before exposing routes, 03-03 records the exact Serve command/configuration and a real HTTPS/auth smoke test in `03-03-EVIDENCE.md`; if the profile cannot supply these properties, browser access remains BLOCKED.

**Replay handoff.** Snapshot response includes committed revision and watermark `W`. Register SSE subscription from cursor `W+1` before declaring the client synchronized; server must either replay all `>W` committed events from durable history or return `cursor_too_old` with a fresh snapshot requirement. Each event has stable Space/conversation/run IDs and monotonic cursor. The client deduplicates by cursor and rejects gaps/reordered events, then fetches a new page. Test a commit between snapshot read and subscription registration, slow subscriber overflow, and Host restart.

## Phase 04 fixed memory and projection rules

**Record schema.** `MemoryRecord` minimally contains id, source kind/id and source revision, owner, text or typed value, `public|personal|sensitive` classification, purpose/scope, allowed destination set, retention/expiry, created/updated revision, confirmation actor/time and correction predecessor. A proposal has `proposed` state and no projection authority. `proposed → accepted/rejected`; `accepted → corrected/deleted/expired`; a correction creates a new revision linked to the prior revision; deleted/expired are tombstones. Sensitive or consequential records require explicit owner confirmation. Export includes lineage and tombstones under the owner's encrypted export; runtime gets only current accepted revisions. Stale revision, untrusted proposal provenance or duplicate command ID rejects.

**Projection order.** For every destination/capability, evaluate `deny`/revocation first, then destination allowlist and data class, then per-capability sync preference `none < metadata < summary < content`, then task relevance and expiry, finally byte/token budget. The most restrictive applicable rule wins. `none` emits nothing; `metadata` emits only opaque ID/type/class; `summary` emits an owner-reviewed bounded summary with source IDs; `content` emits only permitted current record bytes. Sort by relevance then stable record ID and truncate deterministically; record included IDs/revisions and budget in a Host manifest bound to the run. A deleted/scope-revoked record cannot enter any new projection; already submitted provider bytes cannot be recalled, so cancel affected active run if further projection or effect would rely on them, mark `cancelling/uncertain` truthfully, and prevent session reuse.

**E8 quality.** Use at least 40 independently authored task questions over at least 100 owner-accepted public-source Lumen memory records. Qrels name accepted record IDs/revisions and authorized scope; preserve a versioned redacted manifest, URL/license/checksum provenance, and raw evaluation material outside the repository. Freeze before indexing. E8 PASS requires Success@5 >= 0.85, MRR@10 >= 0.70, denominator and Wilson interval, and zero unauthorized projections in separate permission canaries. LoTTE with published qrels is a separate retrieval baseline, not accepted-memory usefulness evidence. A document-derived query is only an infrastructure diagnostic. Keep scoped lexical search unless a reproducible owner-approved candidate improves quality at acceptable latency/cost.

## Phase 05 fixed identity and client gate

**Physical D-056 gate.** Name one Android and one iPhone device, OS/build, secure-storage primitive and exact app build. Each must pass device-generated nonexportable key use where supported, owner-confirmed pairing, outbox persistence across process death, lost-ACK receipt lookup, fixed-watermark replay, synchronized approval state and foreground native audio/permission bridge. Any required failure selects Kotlin Android plus responsive phone web; unsupported iPhone-native behavior is explicit. Record battery/background restrictions and tested network changes. Before this gate, `apps/phone/*.tsx` is a candidate path only. The 05-04/06-03 executor must first read `05-01-EVIDENCE.md` and edit only the selected client files; if evidence is absent, stop as BLOCKED.

**Signed envelope (Phase 05 proposed freeze; independent security approval required).** Use [RFC 8785 JCS](https://www.rfc-editor.org/rfc/rfc8785.html) over I-JSON and [Ed25519 from RFC 8032](https://www.rfc-editor.org/rfc/rfc8032.html); do not invent canonical JSON or signature code. Encode public keys as 32-byte base64url without padding and signatures as 64-byte base64url without padding. The UTF-8 wire body is at most 64 KiB; payload is at most 48 KiB; reject duplicate JSON member names, invalid UTF-8, non-I-JSON values, unknown critical fields, integers outside ±(2^53−1), and unsupported versions before signature verification. `issuedAtMs` and `expiresAtMs` are integer Unix milliseconds; `expiresAtMs` must be after `issuedAtMs` and no more than 300,000 ms later. Permit at most 120,000 ms future skew for `issuedAtMs`; expiry remains strict (`nowMs < expiresAtMs`) and is never extended by skew. A device command carries schema version, Space ID, Host ID and epoch, sender node ID and key ID, recipient Host ID, operation ID, 32-byte cryptographically random nonce (base64url without padding), issue/expiry times, payload digest, and signature. Compute `payloadDigest = SHA-256(JCS(payload))`; sign Ed25519 over UTF-8 `LUMEN-NODE-COMMAND\0v1\0` followed by JCS of every envelope field except `signature`. Verify bounded parse, schema, destination, epoch, key state, strict expiry/skew, payload digest, then signature, then persist `(keyID, HostEpoch, nonce, payloadDigest, expiresAtMs)` atomically with the accepted operation before any effect. Same nonce always rejects as `replay_detected`, including same-payload retries; retries use the original operation ID and Host receipt lookup, not a new effect. Retain replay rows until `expiresAtMs + 120,000 ms`; prune only after that time using the Host clock and durable transaction. Validation errors are stable (`body_too_large`, `invalid_json`, `unsupported_version`, `unknown_critical_field`, `wrong_recipient`, `stale_host_epoch`, `unknown_key`, `revoked_key`, `clock_out_of_range`, `command_expired`, `payload_digest_mismatch`, `invalid_signature`, `replay_detected`, `idempotency_key_reused`). Golden vectors must include raw wire bytes, canonical bytes, payload digest, signing bytes, public key, signature and expected error.

This is an explicit candidate profile, not permission to ship cryptography without review. Task 0 of Phase 05-02 must obtain independent security review of this contract and the RFC 8785/RFC 8032 implementation surfaces. The reviewer either approves the profile unchanged or returns exact edits; implementation tasks remain BLOCKED until a review artifact records reviewer identity, reviewed revision, disposition and findings. Never use an author self-review as this gate.

**Enrollment ceremony (proposed bounds).** Node generates its key locally; Host creates a 32-byte CSPRNG one-use invitation valid for 300,000 ms, bound to Space ID, Host ID/epoch, expected node ID and operation. QR/link contains endpoint, Host key fingerprint and invitation token, never sole authorization. New node signs a Host nonce challenge valid for 90,000 ms; owner surface displays Host and node SHA-256 fingerprints plus a six-digit short code derived from the challenge transcript, then explicitly confirms. The Host accepts at most 5 failed attempts per invitation and 10 new invitations per Host per 10 minutes; it stores no plaintext invitation token after first redemption. Only after valid challenge and owner confirmation does one atomic store commit node ID, public key, credential epoch and consumed-invitation record; grants remain absent. Rotation requires a signature from the current key plus owner confirmation and proof of possession of the replacement key in the same transaction. Lost-device recovery requires a separate owner-confirmed recovery ceremony and immediately revokes the old key; it cannot be selected by a client flag. Commit revocation before rejecting future Host reads/effects; old and pending credentials are both rejected after that commit. Independent security review is required before live pairing. These numerical limits are reviewable defaults; any change must update schemas, fixtures, tests, and the review record together.

**Client fork.** If D-056 passes, `apps/phone/` is Expo/React Native with locked package scripts and native modules. If it fails, Android native work stays under `apps/android-host/` and iPhone uses the authenticated responsive `apps/web/` phone layout; do not create a fake shared TSX app. 05-04 and 06-03 must rewrite their `files_modified` inventory to the chosen branch before execution and run that branch's client test/build plus named physical acceptance. A Go Host test alone cannot pass a client task.

## Phase 06 fixed attention and reply rules

**Shared authority insertion point.** Extend the current `internal/space` command reducer and the `internal/host/execution.go` durable acceptance/dispatch path. `internal/host/approval.go` may present helpers but cannot become a bypassable second authority. All callers that can dispatch a capability, including Hermes-originated intents, pass the same Host validation: actor, target, capability/version, canonical args/artifact digest, grant revision, approval ID, expiry and single-use consumption in one store update. Test every dispatch caller, changed args, grant revocation and concurrent web/phone resolution.

**Waiting/reply state.** Persist `pending → resolved → delivery_pending → delivered/uncertain/expired/cancelled` for attention; the bound Run moves `running → waiting_for_input` or `waiting_for_approval` only after the request commits. A reply uses stable request ID plus stable runtime reply idempotency key. If the documented pinned Hermes API supports reply status/receipt, reconcile lost ACK by that key. If it lacks lookup or safe idempotency, do not resend blindly; mark delivery `uncertain`, keep Run waiting or uncertain and present owner recovery action. A question answer never changes Grant or Approval. Expiry/cancellation after decision but before delivery rechecks current authority and blocks stale effect.

**Compatibility gate.** Phase 06-04 must record pinned Hermes version, documented reply endpoint/event fields, real harmless request/reply, duplicate/lost ACK, restart and cancellation. If any required operation is absent, mark that route `UNSUPPORTED`, disable it in runtime profile and UI, and do not claim Phase 06 complete for the unsupported requirement. No guessed Hermes internal endpoint or model-text parsing substitutes for a protocol contract.

## Proposed module function contracts

The names below are intended API boundaries; they are not required verbatim if current code already has an equivalent. Each proposed function must have one caller or clearly shared use, and tests must exercise its security or recovery property.

| Module/file | Functions or responsibility | Input → output and failure behavior |
|---|---|---|
| `internal/space/model.go`, `internal/space/apply.go`, `internal/space/reducer.go`, `internal/space/state.go` | Existing Conversation/Message/Command/Apply/reducer/state validation surfaces; extend their existing symbols | Extend the live canonical types and reducer in place; no duplicate `conversation.go` authority file. Bind each added transition to a fixture, receipt and replay test. |
| `internal/store/store.go` | existing `Update`, `Read`; migration/export additions only after gate | Atomic authority commit; no acknowledged write without durable ciphertext; recover prior valid generation |
| `internal/conversation/service.go` | `StartTurn`, `ObserveRuntime`, `ReconcileRun`, `RequestStop` | Host run id → bound Hermes operation; persist intent before I/O and evidence before terminal transition |
| `internal/conversation/projection.go` | `BuildProjection`, `InvalidateBinding` | Authorized message/memory/context revision → bounded destination-specific payload with included-source manifest |
| `internal/host/conversation_http.go` | `Send`, `Receipt`, `RunStatus`, `Stop`, `Snapshot`, `Events` handlers | Authenticated scoped request → durable response; fixed-watermark pages, explicit rate/size limits |
| `internal/space/memory.go` | `Propose`, `Accept`, `Correct`, `Delete`, `ExportSelection` | Actor/revision/destination checks; deleted record never projects again |
| `internal/host/device_identity.go` | `BeginEnrollment`, `ConfirmEnrollment`, `RotateKey`, `RevokeNode`, `AuthenticateNode` | Owner-confirmed key and one-use invite → node credential; replay/old epoch denied |
| `internal/host/node_transport.go` | `Attach`, `Replay`, `Dispatch`, `Cancel`, `ObserveReceipt` | Authenticated node stream; stable IDs, bounded frames, reconnection and unknown-outcome handling |
| `internal/space/attention.go` | `RequestQuestion`, `RequestApproval`, `ResolveQuestion`, `ResolveApproval` | Exact binding, expiry, compare-and-set; question has no grant side effect |
| `internal/host/capability_broker.go` | `SelectTarget`, `AuthorizeInvocation`, `ResumeApprovedInvocation`, `DispatchInvocation`, `ReconcileInvocation` | Actor+conversation+capability+typed args+owner target constraint → one durable invocation; pending ask commits attention, accepted decision consumes once, default deny and node revalidation |
| `internal/host/execution.go` | `AcceptRuntimeCapabilityIntent` from a mapped, documented Hermes callback | Derive actor/conversation from persisted Run, then call the same broker; unsupported callback cannot claim action support |
| `apps/mac-node/.../main.swift` | Owner-console `local grant-root|search|read` caller | Selected bookmark and exact one-time ask preview → LocalAction; wrong-user/noninteractive requests deny |
| `apps/mac-node/.../Files.swift` | `Search`, `Read` | Selected security-scoped root + canonical relative path → bounded results; race/symlink/oversize denial |
| `apps/phone/.../outbox` | `Queue`, `FindReceipt`, `RetrySameID`, `ApplyReplay` | Persist ID before network; pairing change fences pending sends; merge by Host revision |
| `apps/android-host/.../DeskIdentity.kt` | Bind selected D-056 desk identity | Shared-client pass enrolls a distinct desk node; fallback reuses this app's Phase 05 key; blocked branch cannot submit |
| `internal/host/interaction.go` | `ObserveRuntimeRequest`, `DeliverDecision`, `ReconcileDecision` | Documented Hermes request ID only; uncertain delivery remains visible |
| `internal/integrations/messaging.go` | `ValidateIngress`, `MapThread`, `QueueDelivery`, `ReconcileDelivery` | Explicit linked identity/thread → Space message/receipt; forged sender and duplicate provider event denied |
| `internal/automation/` | Pure scheduling/eligibility computations and bounded adapters | Inputs/outputs are validated by Host; canonical Goal/Occurrence/Lease state is committed through `internal/space` and `internal/store` |
| `internal/managed/provisioner.go` | `ProvisionTenant`, `SuspendTenant`, `DeleteTenant`, `ReconcileTenant` | Dedicated data-plane state machine; no shared canonical customer content |

## Public API and compatibility checklist

The wire contract starts as versioned JSON as decided in `docs/PLAN.md`; generated clients are optional. HTTP names below are proposed routes to freeze in Phase 03/05 plans. Local CLI control stays on the owner-restricted socket. No web route may expose it by proxy.

| Route family | Phase | Required behavior |
|---|---:|---|
| `/v1/conversations`, `/{id}/messages`, `/{id}/events` | 03 | Create/list scoped conversations, idempotent send, snapshot, fixed-watermark cursor replay and SSE live hint |
| `/v1/runs/{id}`, `/stop` | 03 | Read committed status/checkpoint; idempotent stop intent and truthful terminal outcome |
| `/v1/memory`, `/proposals`, `/{id}` | 04 | Owner inspect/accept/correct/delete with revision checks; records and tombstone lineage are included in the D-024 encrypted full-Space backup owned by Phase 10 (no memory-only export format) |
| `/v1/context-preferences` | 04 | Owner reads/sets per-destination capability disclosure level with version preconditions; unknown, unpaired, or ungranted target denies |
| `/v1/enrollment`, `/v1/nodes`, `/{id}/rotate|revoke` | 05 | One-use invitation, confirmation, authenticated node identity and management |
| `/v1/attention`, `/{id}/resolve` | 06 | Separate question/approval schemas and one accepted resolution |
| `/v1/invocations`, `/{id}/receipt|cancel` | 07 | Authenticated actor and conversation, typed action, Host target/grant check, pending exact approval, durable receipt in same conversation replay and uncertainty |
| `/v1/integrations`, `/v1/automations` | 11/14 | Explicit lifecycle/status, never raw Hermes administration |

Every mutating route requires schema/version, authenticated actor, operation ID and payload digest. Define response codes for validation denial, stale revision, unsupported version, unknown outcome, and dependency unavailable. Publish cross-language fixture vectors for canonical argument hashing, signatures, cursor gaps, duplicate commands, and backward/forward compatibility. Each client supports at least the current and prior published protocol generation or fails with upgrade guidance; no silent schema downgrade.

## Security and operational proof by boundary

| Boundary | Required negative check | Recovery/ops check |
|---|---|---|
| Browser/phone → Host | Unauthorized read/write, CSRF/origin where cookie auth applies, size/rate limit, stale cursor | Reconnect, duplicate send, stop during stream |
| Host → Hermes | Wrong profile, runtime event injection, timeout, tool/network/file denial for claimed profile | Lost create response, SSE loss, runtime restart |
| Host → node | Wrong key/epoch, replay, stale manifest/grant, changed args, target substitution | Lost receipt, target restart, revocation mid-work |
| Node → OS | Symlink/TOCTOU, missing permission, out-of-scope root/account | Permission revoked during action, partial effect |
| Host store | Plaintext temp/journal/index, storage full, partial commit, corrupt key/ciphertext | Backup/restore, schema migration, old-Host fencing |
| Installer/supervisor | Wrong digest/signature, unsafe path, existing Hermes config overwrite | Interrupted stage/publication, reboot, offline rollback |
| Automation/managed | Stale lease, duplicate trigger, tenant cross-read/effect, entitlement blocking export | Worker restart, tenant restore/delete and audit |

Audit records contain IDs, digest, operation, policy decision and outcome, not prompts, credentials, full arguments, raw device data or model traces. Tests and evidence use only synthetic harmless content unless a named live gate explicitly requires a real provider or device. Published support requires revision, runtime digest, OS/device, profile, test method, negative/recovery results and owner-visible limitations.

## Future OS reuse register

| Source in checked-in `references/future-os` | Lumen use | Phase/gate |
|---|---|---|
| `agent/src/rpc/session_prompt.rs`, `agent/src/agent/run_loop.rs` | Snapshot run settings; separate provisional/committed history; bounded context | 03, without runtime canonical authority |
| `mobile/src/remote/usePromptOutbox.ts`, `replay.ts`, `syncEngine.ts` | Stable command ID, receipt lookup, pairing fence, fixed-watermark replay | 03/05, port algorithm and fault cases |
| `packages/rpc/src/command_policy.rs` | Explicit timeout/retry class; no automatic repeat of unknown mutation | 03–17 |
| `desktop/src/features/agent/ApprovalPrompt.tsx` | Visible exact action preview and one-resolution UI behavior | 06 |
| `packages/remote-crypto/src/lib.rs` | Threat/test reference for sequence, framing and identity | 05; independent crypto review before adoption |
| `channels/src/outbox.rs` | Durable provider delivery and honest unknown result | 11 |
| `orchestration/loop/` | Separate ledger, leases, deterministic gates and evidence contracts | 14; Apache-2.0 provenance check |

The local checkout is `98f7f3a3385e12d38ee7fc75bdca2cc3856cf987`, while the prior research document analyzed `907f38b046b32ed3ac795c07b641e681d8e52101`. A phase that copies code must pin one reviewed SHA, check its license/notice, and record line-level source and local modification. Future OS `SECURITY.md` documents broad permission/sandbox defaults and open network paths; none become Lumen defaults.
