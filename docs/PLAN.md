
# Lumen master development plan

## 1. Product, release boundaries, and current baseline

Lumen is a personal AI Space that follows the user across devices. An always-on Host maintains conversations, accepted memory, permissions, tasks and device identity. Hermes supplies reasoning and supported integration capabilities. Paired devices provide interfaces and permission-controlled execution.

The defining demonstration is:

> Start a conversation on a laptop, continue it from a phone after closing the laptop, later ask the phone to perform an action on the laptop, approve the action on the phone, and receive a verifiable result in the same conversation.

Conversation continuity requires the Host to be reachable. Laptop-local execution additionally requires the laptop’s node to be awake and connected.

### Confirmed decisions

| Decision | Status and consequence |
|---|---|
| First release is a personal alpha | Confirmed in this planning session; commercial infrastructure follows demonstrated personal usefulness |
| One active, always-on Host | Confirmed; VPS, home server and always-on Mac remain deployment choices |
| Host and Hermes may share a machine | Confirmed; separate processes, credentials and protected storage |
| Maximize Hermes reuse | Confirmed; use its reasoning, providers, tools, skills and integration mechanics wherever compatible |
| Shared conversations and memory | Confirmed; Host owns canonical records and grants access to authorized surfaces |
| Phone can initiate laptop actions | Confirmed; Host routes to a paired, authorized laptop node |
| Requests from Hermes reach the user | Confirmed; support both approval requests and ordinary clarification questions |
| Existing Hermes or full setup | Confirmed; adopt an existing endpoint or provision a dedicated managed runtime |
| QR/link/code pairing | Confirmed; easy enrollment with authenticated device identity |
| Tailscale is acceptable | Confirmed as an option; application permissions remain Lumen’s responsibility |
| Wake word before STT/TTS | Confirmed as the voice development order |
| Speech local by default | Confirmed; remote speech requires explicit opt-in |
| FutureOS-style mobile implementation | Preferred if feasible; Android native plus phone web is the agreed fallback |
| GSD replaces Superpowers | Confirmed; use one planning workflow |
| Phase-gated implementation | Confirmed; Phase 01 planning/evidence work is authorized, while product feature phases remain subject to their gates |

The attached original prompt is reference material. Later conversation decisions override its earlier suggestions, including streaming pre-activation audio to another machine and treating Hermes as merely an optional early experiment.

### Repository baseline

Inspection found:

| Area | Verified condition |
|---|---|
| Git | Phase 01 and its cleanup merged to `main` at `81b60e3` in PR #21; Phase 02 work starts from that commit |
| Go foundation | Space state transitions, grants, approvals, tasks, runtime mappings and recovery logic exist |
| Persistence | AES-256-GCM encrypted whole-state storage with locking and atomic replacement; not a relational conversation database |
| Operator interface | Owner-restricted local control boundary exists |
| Hermes | Authenticated Runs adapter, event consumption, approval delivery, cancellation and bounded reconciliation exist |
| Installation | Combined/external setup, configuration, supervision, doctor and executable release-generation management exist |
| Live evidence | Recorded Linux/VPS lifecycle and native macOS runtime evidence exist; remaining recovery and update probes are explicitly documented |
| Conversations and memory | Absent from current `main` |
| Android | Disabled companion shell; no working pairing, microphone or execution surface |
| Mobile/web/Mac node | Product clients and remote node execution remain to be built |
| Draft conversation work | [PR #20](https://github.com/AshwanthReddy-exe/Lumen/pull/20) is draft on `feat/lumen-continuity`, 26 committed changes ahead of `main` at `0b67f81` when inspected. It includes conversation/memory and bounded node-status work but retains open live gates; its separate worktree also has uncommitted edits. The older `feat/m2-conversation-nucleus` work is incorporated there and is not an independent completion claim |
| GSD | Runtime identity confirms `@opengsd/gsd-core` version `1.14.0` |

A fresh `go test ./...` on the Phase 01 branch passed **463 tests across nine packages** outside the restricted sandbox. `mise run phase0-check` passed there as well after its sandboxed Gradle initialization failed on cache/native-service permissions. These establish automated baselines for `main`-derived code, not PR #20 validation or live device acceptance. [Phase 01 evidence ledger](../.planning/research/BASELINE-EVIDENCE.md) records the scope and remaining proof.

The PRD and architecture status statements and active links to deleted plans have been reconciled in the Phase 01 artifact set, approved by the owner on 2026-09-27. Do not restore the deleted Superpowers workflow.

### Release sequence

**Personal alpha:** one Space, one owner, always-on Host, Hermes, web administration, phone conversation and approval surface, Mac execution node, inspectable memory, Android desk voice, recovery and measured latency.

**Expanded personal beta:** one messaging integration, selected Apple abilities, controlled browsing and coding, richer native integrations, sustainable updates.

**Paid self-hosted beta:** installation support, license compliance, signed distribution, recoverability, migration, support diagnostics and entitlement behavior.

**Managed beta:** dedicated customer data planes, account and billing systems, provisioning, backups and operational service targets.

**Later product:** bounded automation, additional operating systems, public integration SDK and explicitly designed shared Spaces.

Multi-user customer hosting and multiple people sharing one Space are different features. The former can ship first.

## 2. Architecture and engineering contracts

### Ownership and execution

```text
Phone / web / desktop / Android desk companion
                  │
        authenticated Lumen connection
                  │
         Always-on Lumen Host
  conversations · memory · policy · approvals
  device routing · task records · durable events
          │                         │
     Hermes runtime            Paired nodes
  reasoning, providers,      Mac files/apps/tools
  supported integrations     local voice and devices
          │                         │
          └── bounded evidence and receipts ──┘
```

Use the existing Go Host as a modular service. Retain inward dependency direction: domain rules remain independent of Hermes, UI, transport and storage implementation.

Separate two execution classes:

- **Runtime-local work:** reasoning, isolated research, approved speech generation and other allowed work near Hermes.
- **Device work:** files, local applications, desktop interaction, local speech and hardware access on paired nodes.

Hermes may request an ability and express target constraints. The Host selects an eligible node using explicit user choice, permission, availability and capability compatibility. A request naming “my laptop” must not silently move to another computer.

### Hermes integration and enforcement

Preserve the existing Runs adapter. Add a narrow broker for Lumen capabilities rather than exposing the owner control interface.

Adoption order:

1. Use an existing Hermes capability directly when its execution and data boundaries meet the task.
2. Configure a dedicated supported Hermes profile.
3. Extend through a supported tool/plugin/MCP integration.
4. Use a native node adapter for device-specific enforcement.
5. Build a new intelligence subsystem only after documenting a concrete unsupported requirement.

A profile digest records the intended configuration; it does not prove isolation. Enforce permissions through the actual enabled tools, process privileges, filesystem access, credentials, network policy and broker validation. Behavioral tests supplement these controls.

For an externally managed Hermes endpoint, Lumen can verify endpoint identity and observed behavior but cannot prove a hostile remote operator’s internal configuration. Setup must explain this trust boundary. Stronger isolation claims apply only to controlled deployments.

Do not promise that every Hermes approval or clarification feature is available through the pinned Runs API. An early compatibility experiment must verify exact events and response paths. Unsupported interactions require a supported extension or must remain disabled.

### Conversation and memory

A canonical conversation consists of messages and product events, independent of Hermes session IDs.

Required records:

| Record | Minimum purpose |
|---|---|
| Conversation, Message, Surface | Shared timeline, origin, author, message ordering and completion state |
| Task, RuntimeBinding | Durable user work and replaceable Hermes run/session mapping |
| CommandReceipt | Idempotent handling of retried user and node commands |
| ConversationEvent | Replayable product updates with a durable cursor |
| ContextRecord, MemoryProposal | Accepted knowledge and proposed additions with provenance |
| ApprovalRequest | Exact permission decision and delivery state |
| InteractionRequest | Clarification question, choices/free text, expiry and reply binding |
| Node, Capability, Grant | Device identity, advertised abilities and actual authority |
| Invocation, ExecutionReceipt | Device dispatch and recorded outcome |
| ArtifactReference | Encrypted attachment metadata, access scope and retention |

Every accepted message stores its receipt and execution intent before runtime I/O. Retry with the same command ID returns the same receipt. Reusing an ID with different content is rejected.

One conversation has one active turn initially. New messages queue by default; stopping or replacing a turn is explicit. Separate conversations may execute concurrently within configured limits.

Token deltas are provisional display data. Batch checkpoints and persist final messages; do not rewrite the complete encrypted Space for every streamed token.

Memory begins with explicit user preferences and clearly proposed facts. Each accepted record has:

- Source message or artifact.
- Scope and sensitivity.
- Retention/expiry.
- Revision and correction history.
- Permitted destinations.
- Confirmation state.

Context assembly applies authorization and destination policy before serialization. Runtime session reuse must not reintroduce deleted or newly restricted context; invalidate/rebind sessions after relevant policy changes.

Start retrieval with scoped recent history, explicit preferences and lexical search. Add embeddings and reranking only when an independently authored retrieval evaluation demonstrates a useful improvement.

### Persistence decision

The current encrypted snapshot store remains the baseline for foundation closure. Conversation growth warrants an early storage experiment.

The proposed evolution is one transactional local database per Space, preferably SQLite with a maintained encryption solution, plus encrypted attachment storage. The experiment must compare this against retaining the snapshot store for a bounded alpha.

Selection gate:

- Atomic message, receipt, task-intent and event-cursor commit.
- Tested crash recovery and backup restoration.
- No plaintext payload leakage through journals, temporary files or indexes.
- Reproducible Linux/macOS builds and acceptable licensing.
- Measured performance at declared alpha data volumes.

Do not introduce PostgreSQL, Redis, Kafka or a vector service for one personal Space. If a database migration is selected, migrate all mutually dependent authority records transactionally; avoid splitting one operation across independent stores without an explicit recovery protocol.

Keep the previous encrypted snapshot as an immutable recovery input. Never downgrade a newer data schema simply because an older executable is installed.

### Device connectivity and synchronization

For the personal alpha, use Tailscale-assisted private reachability as the simplest remote-access baseline. Retain a transport-independent Lumen protocol. Public relay infrastructure follows only when onboarding evidence demonstrates its need.

Pairing flow:

1. Owner opens enrollment on a trusted surface.
2. Host issues a short-lived, single-use invitation.
3. QR/link carries connection information and authenticated pairing material.
4. New device generates its own key.
5. Owner confirms the expected device and verification code.
6. Host records membership and issues scoped credentials.
7. Device advertises abilities; grants are a separate step.

A short human code is not the sole cryptographic secret. Invitations expire, are rate-limited and cannot be reused.

Use HTTPS request/response for user commands and paginated retrieval; SSE for browser event streaming; a persistent authenticated node connection for bidirectional dispatch. Specify versioned JSON contracts and cross-language fixtures first. Introduce binary RPC only if measured requirements justify it.

Required operations include:

- Start/complete enrollment; list/revoke nodes.
- Submit message; retrieve command receipt.
- Read snapshot and cursor-based event pages.
- Resolve approval or answer clarification.
- Advertise capabilities and health.
- Dispatch/cancel invocation and report receipt.

A live connection is an update hint. After gaps, reconnect or restart, clients replay durable events against a fixed watermark. Store the authorized local projection and its cursor atomically.

Offline behavior:

- Read previously synchronized, locally authorized history.
- Queue messages with stable command IDs.
- Clearly label pending and stale state.
- No new cross-node execution.
- Optional offline local work only under short-lived cached grants.

Revocation prevents future Host access immediately after commitment. A disconnected device cannot receive that decision immediately; cached-grant expiry bounds its remaining offline authority. Remote deletion of an offline replica is not guaranteed.

### Approvals and clarification

These require distinct UX and lifecycle records.

**Approval:** “May I send this exact message to this recipient?” The Host binds the decision to task, action, arguments, destination, expiry and current grant revision.

**Clarification:** “Which project did you mean?” The answer supplies input and grants no new authority.

Both appear in a durable attention inbox and the associated conversation. Notifications contain minimal information and link into an authenticated surface.

Concurrent answers from phone and laptop use a single accepted resolution. Other surfaces show the result. Expired prompts cannot resume work. A changed action requires a new approval. Delivery failure retains the decision for bounded reconciliation.

Host approval does not override a local OS denial. Remote approval also cannot satisfy every native macOS or mobile consent dialog.

### Cross-node execution

For a phone-to-Mac request:

1. Host accepts the message.
2. Hermes requests a typed capability.
3. Host selects the Mac and checks grants.
4. Host obtains any required approval.
5. Host persists an invocation before sending it.
6. Mac checks identity, epoch, expiry, resource scope and local permission.
7. Mac durably records receipt/acceptance before performing a side effect.
8. Host records returned evidence and updates every authorized surface.

Use at-least-once delivery with idempotent command handling. Do not claim universal exactly-once external effects. If an external service lacks an idempotency mechanism and the result is lost, report an uncertain outcome and reconcile before retrying.

Start with `files.search`, `files.read` and a harmless notification. File roots, search/read permissions, output size and content sharing are independently scoped. Revalidate paths at execution, including symlink and file-replacement races.

### Security and recovery

Use separate credentials for owner access, device identity, runtime access and provider accounts. Private keys belong in platform secure storage where supported; runtime secrets must not appear in prompts, logs or launch arguments.

Encryption at rest protects stored data; a running Host processing plaintext remains a trusted component. Managed hosting must communicate this honestly.

Host recovery requires fencing. A locally incremented epoch alone cannot stop an isolated old Host from executing work on equally isolated nodes. V1 migration requires explicit retirement/revocation of the previous authority and fail-closed reconnection rules. Automatic failover is deferred.

Prompt and tool content remain untrusted. Enforcement occurs outside model instructions. Downloads, browser content, attachments and plugins receive explicit limits and provenance.

## 3. Technology, experience, performance and experiments

### Technology decisions

| Subsystem | Proposed selection | Reason and alternative |
|---|---|---|
| Authority/service | Existing Go implementation | Preserves tested code and distribution; no backend rewrite |
| Intelligence | Pinned Hermes runtime | Reuses the agent stack; certify upgrades before promotion |
| Web | React + TypeScript, static build served by Host | Shared language with proposed mobile client; avoids a separate server-rendering service |
| Phone apps | Expo/React Native with stable local native modules | FutureOS demonstrates this shape; native Kotlin/Swift handles device-specific work |
| Mobile fallback | Existing Kotlin Android direction plus responsive phone web | Applied if the bounded shared-client experiment fails |
| Mac node | Native Swift service/application integration | Native permissions, secure storage, filesystem and app integration |
| Native mobile bridge | Kotlin/Swift Expo modules | Keeps continuous audio and secure operations outside the JavaScript event loop |
| Wire contract | Versioned JSON, HTTP/SSE and authenticated node channel | Inspectable and compatible with existing fixtures |
| Persistence | Existing encrypted store, then experiment-gated transactional evolution | Avoid unmeasured migration while addressing conversation growth |
| Search | Scoped lexical retrieval first | Low operating cost; embeddings require quality evidence |
| Speech | Local wake/VAD, local STT, native/offline TTS where verified | Device measurements choose model assets; Hermes handles approved remote speech |
| Deployment | Existing systemd/launchd/Compose paths | No Kubernetes for alpha |
| Remote alpha connectivity | Tailscale-assisted | Avoid building relay infrastructure before proving the product |
| Observability | Structured redacted logs and bounded metrics | Add distributed tracing when node execution creates a real diagnostic need |

FutureOS’s mobile package contains Expo, React Native, secure storage, camera pairing, notifications and native modules. Expo supports local Swift/Kotlin modules through development builds. This supports feasibility of shared mobile UI, but does not prove Lumen’s audio, offline-store or permission requirements. [Expo native modules](https://docs.expo.dev/workflow/customizing/)

Use stable supported versions verified at phase entry. Do not copy FutureOS’s dependency versions automatically.

### Voice and platform policy

Wake detection is the first voice milestone. Implement it independently of remote inference, followed by STT/TTS integration.

The Android desk companion is the first continuous-listening evidence target. Microphone service activation must respect foreground and while-in-use restrictions. [Android foreground-service restrictions](https://developer.android.com/develop/background-work/services/fgs/restrictions-bg-start)

The iPhone alpha provides chat, approvals and foreground voice. Do not promise a universally available custom background wake word. Apple restricts background service purposes and requires consent and recording indicators. [Apple review guidelines](https://developer.apple.com/app-store/review/guidelines/uk/)

Candidate local speech pipeline:

- Native capture and bounded in-memory pre-roll.
- Local keyword detector; evaluate sherpa-onnx as an initial candidate.
- Local VAD and endpoint detection.
- Local recognition model selected against the actual phone.
- Canonical transcript submission to Host.
- Hermes response.
- Local verified-offline speech synthesis and playback.

Sherpa-onnx supports keyword spotting and mobile platforms, but individual model licenses and resource requirements still require qualification. [Keyword spotting](https://github.com/k2-fsa/sherpa/blob/master/docs/source/onnx/kws/index.rst), [platform support](https://github.com/k2-fsa/sherpa/blob/master/docs/source/intro.rst)

Piper and Kokoro remain candidates, not defaults. Engine, voice-weight and dataset terms require separate review; Piper voice documentation explicitly points to per-voice model cards. [Piper voice licensing guidance](https://github.com/OHF-Voice/piper1-gpl/blob/main/docs/VOICES.md)

Never treat “no API key” as proof that TTS runs locally.

### Interface specification

Primary destinations:

| Destination | Required content |
|---|---|
| Conversation | Timeline, attachments, streaming, stop, pending questions and task cards |
| Activity | Running, queued, waiting, failed, cancelled and uncertain tasks |
| Attention | Approvals and clarification requests with expiry |
| Memory | Accepted records, provenance, scope, edit/delete and retention |
| Devices | Pairing, connection quality, last seen, capabilities and grants |
| Settings | Hermes connection, provider policy, voice, privacy, export and recovery |

Phone navigation emphasizes conversation, attention and devices. Web exposes administration and detailed inspection. Mac uses a status/menu interface for connectivity, stop, local permissions and selected roots. Android desk mode provides large listening/speaking indicators and touch-accessible stop/mute controls.

Shared components include task cards, approval previews, clarification forms, connection banners, memory provenance panels and resource selectors.

Every interface must cover empty, loading, streaming, stale, reconnecting, permission denied, expired, unsupported, storage full and unknown-outcome states. Distinguish “accepted by Host,” “started on device” and “completed.”

Accessibility acceptance includes keyboard navigation, screen-reader labels, text scaling, visible focus, adequate contrast, reduced motion, transcript alternatives and no color-only status.

Design tokens cover color, typography, spacing, motion and semantic status. Native permission dialogs remain native. Produce and review a GSD `UI-SPEC.md` before implementing each substantial surface.

### Latency and capacity

Treat earlier statements that VPS latency is “small” as hypotheses. Measure the complete route.

Track:

- Local interaction acknowledgement.
- Client-to-Host round-trip.
- Host validation and durable-write time.
- Hermes queue time.
- Provider time to first token.
- Tool/node dispatch and execution time.
- Voice endpointing, recognition and first-audio delay.

Initial alpha acceptance targets—not measured claims:

| Metric | Target under declared test conditions |
|---|---|
| Local UI acknowledgement | p95 under 100 ms |
| Host acceptance, excluding WAN | p95 under 100 ms at the alpha workload |
| Lumen overhead before runtime dispatch | p95 under 150 ms, excluding WAN/provider |
| Connected approval appearance | p95 under 1 second after Host commitment |
| Local playback stop | p95 under 150 ms |
| First response audio | p50 under 2.5 s, p95 under 5 s for a simple voice turn |
| Device reconnection and small missed-history replay | p95 under 5 s after usable connectivity returns |

Declare hardware, model, region, payload size and concurrency for every result. Report direct and relayed Tailscale paths separately.

Use an alpha workload of one owner, five enrolled devices, two concurrently active conversations, ten thousand messages and one thousand memory records. These are benchmark dimensions, not promised scaling limits.

### Required experiments

| Experiment | Setup and measurements | Pass/fail decision |
|---|---|---|
| E1 Hermes interaction contract | Pinned runtime; exercise approval, clarification, stop, lost stream and restart | Enable only verified interactions; add a supported extension for missing ones |
| E2 Runtime containment | Attempt unauthorized file/network/tool/memory access using a restricted profile | Any escape blocks the claimed profile |
| E3 Shared mobile client | Physical Android and iPhone; pairing, secure key use, replay, approval and native audio bridge | Adopt shared client if stable; otherwise use agreed fallback |
| E4 Storage growth | Existing store versus transactional candidate; crash injection and representative data | Choose based on atomicity, encryption, portability and latency targets |
| E5 Connectivity | Phone cellular, home Mac, VPS; direct/relayed paths and reconnect | Establish alpha network configuration and identify whether relay work is justified |
| E6 Local wake | At least 100 intended wake attempts plus eight hours of negative household audio | Target ≥95% intended detection and ≤1 false activation/hour; otherwise tune or restrict advertised conditions |
| E7 Speech resources | Real Xiaomi device; quiet/noisy speech, playback, battery and thermal monitoring | Select models meeting task accuracy, resource and latency budgets |
| E8 Memory usefulness | Independently authored questions with expected source records and privacy canaries | No unauthorized-context retrieval; measured retrieval usefulness before adding semantic search |
| E9 Side-effect uncertainty | Disconnect before/after node effect and before receipt persistence | No blind duplicate side effects; uncertainty visible and recoverable |
| E10 Migration fencing | Restore with old Host and nodes intermittently reachable | No automatic two-Host authority claim; document required retirement process |

Experiments use consented test material and isolated fixtures. No live messages, purchases or destructive user-file operations are required.

## 4. Dependency-aware roadmap

Effort estimates are engineering-person-week ranges, including tests and review. They are planning estimates, not delivery promises. Later phase estimates must be recalibrated after the experiments.

Every phase completes only when its requirement mappings, automated checks, negative cases, recovery evidence and relevant owner demonstration pass. A missing device or credential produces “blocked/unverified,” never a pass.

### Phase 01 — Reconcile the repository and establish GSD authority

**Prerequisites:** approval of the planning structure.
**Effort:** 0.5–1 week.

Tasks:

1. Produce complete codebase maps and an evidence inventory.
2. Classify existing docs as current requirement, historical evidence or superseded proposal.
3. Map conversation decisions to stable requirement IDs.
4. Replace references to deleted planning files with GSD navigation.
5. Record cleanup deletions without restoring removed workflow artifacts.
6. Inspect surviving M2 branch changes and classify reusable code and unresolved findings.
7. Create the master risk, experiment and decision registers.

**Deliverables:** GSD project, requirements, roadmap, state, codebase map and onboarding index.

**Acceptance:** every active requirement has one owner and destination phase; implementation claims link to code/evidence; deleted plans are no longer navigation targets.

**Risks/tests:** historical assertions mistaken for current behavior; broken links; accidental restoration of unwanted workflow state.

### Phase 02 — Close the existing Host/Hermes foundation

**Prerequisites:** Phase 01.
**Effort:** 1–2 weeks.

Tasks:

1. Verify clean installation and authenticated connection using the existing pinned runtime.
2. Demonstrate native artifact update and rollback on a real machine.
3. Test pinned-runtime stream loss, Host restart and reconciliation.
4. Demonstrate actual cross-machine external Hermes Runs.
5. Resolve image-visibility contradictions through an anonymous pull check.
6. Specify safe container-generation replacement or explicitly restrict alpha updates to the verified path.
7. Run E1 and E2.

**Acceptance:** no duplicate execution after ambiguous dispatch; approval denial holds; cancellation and unknown outcomes are honest; identity/state survive recovery.

**Risk:** upstream features may not provide the assumed control surface. Scope profiles accordingly rather than weakening enforcement.

### Phase 03 — Validate mobile, storage, network and wake foundations

**Prerequisites:** Phase 01; runtime tests depend on Phase 02.
**Effort:** 1–2 weeks.

Tasks:

1. Run E3–E6 with bounded prototypes.
2. Select mobile strategy and exact supported platform matrix.
3. Select persistence path and migration requirements.
4. Measure regional network routes and node reconnect behavior.
5. Prove local wake detection without inference or outbound microphone data.

**Acceptance:** reproducible experiment reports, dependency/license inventory, explicit pass/fail decisions and chosen fallbacks.

**Risk:** native mobile/audio requirements exceed the shared-client benefit. Apply the agreed fallback rather than maintaining two competing apps.

### Phase 04 — Canonical conversations and product API

**Prerequisites:** Phases 02–03.
**Effort:** 2–3 weeks.

Tasks:

1. Add conversation, message, surface, receipt and runtime-binding records.
2. Implement transactional acceptance and durable execution intent.
3. Add authenticated command and replay APIs.
4. Normalize provisional and committed output.
5. Implement one-active-turn ordering, queueing and explicit cancellation.
6. Build a minimal web conversation surface.

**Acceptance:** restart Host and Hermes independently without losing accepted messages; duplicate sends return one result; event gaps recover through replay.

**Tests:** crash at each persistence/I/O boundary, profile substitution, concurrent sends, oversized output and Unicode truncation.

### Phase 05 — Accepted memory and context controls

**Prerequisites:** Phase 04.
**Effort:** 1.5–2.5 weeks.

Tasks:

1. Implement classified context records and memory proposals.
2. Build deterministic context selection and budgets.
3. Add memory inspection, confirmation, correction and deletion.
4. Invalidate runtime context when privacy policy changes.
5. Run E8 and add retrieval metrics.

**Acceptance:** relevant memory survives runtime replacement; correction affects subsequent turns; revoked/deleted material is excluded from new projections.

**Risks/tests:** sensitive inference, stale session memory, conflicting facts, contaminated evaluation queries and backup retention.

### Phase 06 — Pairing, client identity and reliable synchronization

**Prerequisites:** Phase 04; mobile/network decision from Phase 03.
**Effort:** 2–3 weeks.

Tasks:

1. Implement expiring enrollment invitations and confirmation.
2. Add device key storage, membership and revocation.
3. Implement authenticated connections and version negotiation.
4. Add atomic authorized replica/cursor storage.
5. Implement persistent offline message outbox and receipt lookup.
6. Ship phone chat and device-management screens.

**Acceptance:** phone continues a laptop conversation while laptop is off; reconnect does not duplicate messages; a revoked device cannot fetch new state.

**Tests:** stolen/reused invitation, clock skew, app suspension, cache corruption, lost acknowledgements and older client versions.

### Phase 07 — Attention inbox and cross-device human input

**Prerequisites:** Phases 02, 04 and 06.
**Effort:** 1–2 weeks.

Tasks:

1. Distinguish approvals from clarifications in contracts.
2. Persist and display requests across surfaces.
3. Bind exact approval arguments and consume resolutions once.
4. Deliver replies through the certified Hermes interaction path.
5. Add expiry, cancellation, race handling and privacy-preserving notifications.

**Acceptance:** a Hermes request can be answered from the phone and reflected on web; two simultaneous answers produce one accepted resolution.

**Tests:** stale prompt, changed action, malicious reason text, missing runtime support, lost reply and restarted Host.

### Phase 08 — Mac execution node and useful cross-device action

**Prerequisites:** Phases 06–07.
**Effort:** 2–3 weeks.

Tasks:

1. Implement Mac node enrollment and capability advertisement.
2. Add user-selected file roots and local permission controls.
3. Implement search/read/notification adapters.
4. Connect Hermes typed requests through the Host broker.
5. Persist invocation and execution receipts.
6. Run E9 and add local/global stop controls.

**Acceptance:** ask from phone, approve from phone, execute on Mac, receive a bounded result and receipt.

**Tests:** Mac asleep, node disconnect, symlink escape, file replacement, revoked root, denied OS permission and duplicate invocation.

### Phase 09 — Speech and Android desk companion

**Prerequisites:** wake experiment from Phase 03; Phases 04, 06–08.
**Effort:** 2–4 weeks.

Tasks:

1. Productize local wake/mute/listening indicators.
2. Add STT after activation and TTS playback.
3. Implement partial transcription and canonical final transcript.
4. Add barge-in, echo handling and cancellation propagation.
5. Add explicit remote-speech consent and resource-aware engine selection.
6. Run E7 and long-duration desk-device checks.

**Acceptance:** no pre-activation audio egress; voice can initiate the same phone-to-Mac workflow; interruptions stop playback promptly; disabled mic remains visibly disabled.

**Tests:** noisy room, assistant self-trigger, calls/headphones, app suspension, missing offline voice, thermal throttling and network interruption.

### Phase 10 — Personal alpha acceptance and recovery

**Prerequisites:** Phases 02–09.
**Effort:** 1–2 weeks plus a minimum seven-day soak.

Tasks:

1. Complete encrypted backup/export and restore drills.
2. Run E10; document manual migration and old-Host retirement.
3. Execute end-to-end user journeys and accessibility review.
4. Measure latency/resource budgets.
5. Exercise outage, storage-full, provider failure and node-loss scenarios.
6. Publish the alpha’s supported configurations and limitations.

**Acceptance:** sustained use across Host, Mac and phone; no unexplained canonical data loss; no unauthorized effects; every injected failure yields the specified visible state.

### Phase 11 — One messaging integration

**Prerequisites:** Phase 10.
**Effort:** 1–2 weeks.

Default pilot: Telegram, subject to credential availability and Hermes contract qualification.

Tasks: authenticated ingress, durable provider-event dedup, identity/thread mapping, attachment limits, delivery outbox, uncertain-delivery handling and disconnect/revocation.

**Acceptance:** a selected conversation continues between app and messaging with explicit linking; no accidental merging of unrelated chats.

**Tests:** duplicated webhooks, forged sender metadata, rate limits, provider edits/deletions and lost send acknowledgement.

### Phase 12 — Apple abilities and controlled browsing

**Prerequisites:** Phases 08 and 10.
**Effort:** 2–4 weeks.

Tasks:

1. Add reminder creation as the first Apple write capability.
2. Qualify Notes and messaging separately.
3. Adopt isolated Hermes browsing for research.
4. Add exact approvals for consequential browser actions.
5. Bind artifacts, credentials and retention to each capability.

**Acceptance:** reminder creation has an external item reference; browser cannot inherit personal credentials without consent; unrelated device resources remain inaccessible.

**Tests:** prompt injection, account ambiguity, duplicate writes, changed webpage state and denied native permissions.

### Phase 13 — Controlled coding and broader device execution

**Prerequisites:** Phases 08, 10 and browser/tool qualification where used.
**Effort:** 2–4 weeks.

Tasks: isolated Git workspaces, bounded command profiles, credential restrictions, patch/test artifacts, approval before applying changes, cancellation and cleanup.

Add Windows execution as a separate conformance implementation when the Mac contract is stable.

**Acceptance:** remote coding produces a reviewable patch and evidence; failed or cancelled work cannot silently alter the user’s canonical project.

**Tests:** escaping paths, malicious repository instructions, long-running processes, interrupted patch application and worktree conflicts.

### Phase 14 — Durable automation and delegation

**Prerequisites:** Phase 10 and at least one stable effectful capability.
**Effort:** 2–4 weeks.

Tasks: automation preview, approved schedules, timezone/missed-run policy, durable trigger receipts, bounded Hermes workers, child authority subsets, budgets, pause/revoke and outcome delivery.

Reuse Hermes worker and scheduling mechanics only where one scheduler remains clearly authoritative.

**Acceptance:** automation survives restart, executes once per logical trigger where supported, and cannot expand its grants.

**Tests:** daylight-saving transitions, duplicate triggers, expired grants, child failure, budget exhaustion and cancellation of descendants.

### Phase 15 — Paid self-hosted readiness

**Prerequisites:** Phase 10 and selected beta cohorts.
**Effort:** 2–4 weeks.

Tasks: signed release chain, dependency/voice-model licensing review, onboarding, diagnostics, recovery documentation, entitlement UX, support workflow and compatibility matrix.

**Acceptance:** independent installation by a new user, verified update/rollback, usable export despite entitlement expiry, and no secret leakage in support bundles.

### Phase 16 — Managed service

**Prerequisites:** Phase 15.
**Effort:** 4–8 weeks.

Tasks: account authentication, dedicated per-customer data planes, provisioning, billing integration, key management, backups, operational alerts, quotas, incident response and deletion lifecycle.

**Acceptance:** automated cross-customer isolation tests; restore drills; measured operating costs; explicit provider/admin trust disclosures.

Shared multi-tenant agent processes are outside the initial managed design.

### Phase 17 — Public integration ecosystem and shared Spaces

**Prerequisites:** stable beta contracts and operational evidence.
**Effort:** split into later milestones; initial contract publication approximately 2–4 weeks.

Tasks: versioned SDK, fixtures, compatibility policy, signed extension manifests, permission declarations and review/disablement process.

Shared Spaces require a separate principal/role/ownership design, member-specific memory visibility, billing ownership and audited invitation/revocation.

**Acceptance:** an independent node passes conformance without proprietary internals; extensions cannot gain undeclared authority.

The personal-alpha path is Phases 01–10. Expected effort is approximately **14–25 engineering-person-weeks**, with experiments and UI work partially parallelizable. AI assistance can reduce implementation time but does not eliminate real-device, security, recovery or soak testing.

## 5. Verification, risks, commercial constraints and GSD delivery

### Failure behavior

| Scenario | Required detection and behavior |
|---|---|
| Host unavailable | Show last synchronized state; queue eligible user input; do not claim remote execution |
| Hermes/provider unavailable | Preserve accepted message and task; bounded retries only where safe |
| Laptop asleep/offline | Report unavailable or explicit queue with expiry; no silent alternate target |
| Lost runtime-create response | Recover by supported idempotency/status contract; otherwise unknown outcome |
| Node acted but receipt lost | Query durable receipt; avoid blindly repeating side effects |
| Concurrent approvals | Accept one resolution and synchronize all surfaces |
| Permission revoked mid-task | Prevent subsequent protected steps; attempt cancellation; preserve already-observed effects |
| Storage full | Fail new durable acceptance safely; never acknowledge a write that did not commit |
| Corrupt cache | Rebuild authorized projection; preserve separately recorded pending command IDs |
| Incompatible version | Refuse unsupported operations and show upgrade guidance |
| Failed migration | Preserve recoverable prior generation; do not launch incompatible older code on newer data |
| Malicious web/tool content | Treat as data; authorization remains outside model output |
| Lost device | Revoke future access; expire cached grants; explain limits of offline erasure |
| Cloud speech disabled | Keep audio local or report unavailable; no automatic cloud substitution |

### Quality gates

Use existing Go checks and real deployment scripts rather than inventing pass markers. Expand CI when each subsystem appears:

- Unit and deterministic state-transition tests.
- Go race tests and concurrency scenarios.
- Cross-language protocol fixtures.
- Pinned Hermes integration tests.
- Mobile/native build and real-device checks.
- Browser/client accessibility and end-to-end tests.
- Crash/network fault injection.
- Secret/dependency/license scanning.
- Signed artifact verification.
- AI, retrieval and speech evaluation.
- Backup/restore and migration acceptance.

Record code revision, runtime digest, configuration, hardware, test dataset, commands, results and limitations for each acceptance run.

No model response, task summary or green unit-test suite alone establishes production readiness.

### Commercial boundaries

Keep product pricing and payment-provider choice outside alpha implementation. Preserve export and recovery regardless of subscription state. Maintain separate estimates for infrastructure, model tokens, speech, browser services, storage, egress and support.

Licensing review covers source code, native libraries, model weights, voices, redistributed tools and hosted-service terms separately. A permissive repository license does not establish rights for every bundled model or integration.

Managed encryption claims must match the actual key-access design. Provider retention and regional routing are verified properties, not labels in a settings screen.

### GSD artifacts and workflow

Use the installed GSD 1.14.0 conventions:

```text
.planning/
  PROJECT.md
  REQUIREMENTS.md
  ROADMAP.md
  STATE.md
  config.json
  codebase/
  onboarding/SUMMARY.md
  research/
  phases/
    01-baseline-and-planning/
      01-CONTEXT.md
      01-RESEARCH.md
      01-01-PLAN.md
      01-02-PLAN.md
      01-03-PLAN.md
```

Keep product behavior in the existing PRD, system boundaries in architecture, and accepted decisions in the decision record. GSD owns sequencing, task plans, verification and progress. Use a master index and cross-references instead of duplicating each specification.

Preserve existing `FR-*` identifiers. Add new requirements only for actual additions, including clarification routing, explicit remote-speech consent, approval synchronization and shared-mobile feasibility.

Recommended GSD settings:

- Interactive mode.
- Research, plan checking and verification enabled.
- Automatic phase advancement disabled.
- At most three workers for independent work.
- Human checkpoints retained.
- No automatic feature execution or shipping during planning.
- Planning commits disabled until the owner reviews the artifact set.

Supported progression:

`gsd-map-codebase` → `gsd-ingest-docs` → project initialization as needed → `gsd-onboard` verification → `gsd-discuss-phase`/`gsd-spec-phase` → `gsd-plan-phase`.

Use `gsd-ai-integration-phase` for AI contracts, `gsd-ui-phase` for interface contracts, and security review for trust-boundary phases. Execution, verification and shipping commands become available only after the relevant plans are approved.

### Earliest execution-ready planning phase

**01-01 — Repository evidence map, wave 1**

- Inspect all source modules, tests, deployment configurations and retained branches.
- Produce the supported codebase map and current-state evidence table.
- Mark historical, mocked, live-proven and unverified claims distinctly.
- Verify the current automated baseline and link remaining live probes.
- Complete when each implementation claim has an evidence pointer and every major subsystem is classified.

**01-02 — Requirements and architecture reconciliation, wave 2; depends on 01-01**

- Incorporate the confirmed conversation decisions and this planning session’s answers.
- Correct stale implementation status.
- Record mobile/storage decisions as experiment-gated with explicit fallback rules.
- Separate approval from clarification and fix offline-revocation/fencing claims.
- Repair navigation to deleted plans.
- Complete when requirements, architecture and risk registers contain no conflicting ownership or release claims.

**01-03 — GSD roadmap and plan validation, wave 3; depends on 01-02**

- Create project, requirements, roadmap and state artifacts using installed templates.
- Map every requirement to phases and each initial plan to requirement IDs.
- Populate plan frontmatter: phase, plan, type, wave, dependencies, affected files, requirements and observable `must_haves`.
- Run plan validation and independent consistency review.
- Record owner review as the blocking checkpoint before Phase 02 execution.
- Complete when the roadmap is navigable, dependencies are acyclic, acceptance criteria are testable and no feature work has begun.

### Review outcome and remaining evidence

The proposed architecture retains the tested Go foundation, gives Hermes substantial responsibility, and concentrates Lumen-specific engineering on continuity, user control and device execution.

The highest-risk assumptions have explicit experiments before dependent implementation: runtime containment and interaction support, shared mobile native integration, storage evolution, wake reliability, remote connectivity and migration fencing.

This is the approved master plan. Phase 01 GSD artifacts merged in PR #21. Phase 02 is active for planning from updated `main`; its live evidence gates remain open until verified.
