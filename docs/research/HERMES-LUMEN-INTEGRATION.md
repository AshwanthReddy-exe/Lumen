# Hermes–Lumen integration research

Status: architecture research and recommendation. This document does not close Milestone 1, certify a production Hermes build, or grant any capability.

## Evidence and reading rule

Facts in this document are labelled by provenance:

- **Repository fact** means it is stated or implemented in this repository. The canonical sources are linked inline.
- **Official Hermes fact** means it is stated in Hermes's official documentation or release/source repository. It still requires a pinned behavioral certification before production use.
- **Report claim** means it appears in the uploaded report at `/Users/ashwanthreddyboddireddy/Downloads/hermes-lumen-compatability.md`. The report is useful due diligence, but it is untrusted input and is not an authority or compatibility certificate.
- **Recommendation** means a proposed Lumen design choice, not an existing implementation.

The supplied report uses citation tokens rather than durable source links in most places. Its release, feature, effort, and maturity claims must therefore be rechecked against the official Hermes source and documentation before being used as release evidence.

## Executive conclusion

Hermes is a strong native intelligence platform for Lumen, but it must remain a runtime adapter and untrusted dependency. The smallest durable seam is:

```text
Lumen Space Authority / Host
  ├── HermesRuntimeAdapter v1 ── authenticated Runs API + bounded SSE ──> Hermes
  └── Capability Broker / MCP v1 <── typed capability intents ───────── Hermes
                                      │
                                      └── Host policy → node protocol → node
```

Hermes can supply agent looping, model/provider routing, tools, skills, MCP, browser, gateways, voice, automation, delegation, workers, and observability. Lumen must own identity, canonical conversations and accepted memory, context projection, grants, target selection, approvals, credentials, durable task state, receipts, audit, revocation, migration, and uncertainty. This is the boundary in `docs/DESIGN-PRINCIPLES.md:25-37,45-71,75-77` and `docs/ARCHITECTURE.md:21-49,78-87,211-213`.

The report's most valuable architectural direction is its layered recommendation at lines 836–861: use Hermes where it is strong, while keeping a versioned Lumen capability broker and secure node fabric. The report's convenience APIs (`lumen.state.put`, arbitrary `lumen.call`, `lumen.broadcast`) must not be adopted because they would create a second authority path.

Milestone 1 remains active. The repository explicitly says that clean setup, pinned distribution, real configured-runtime behavior, reboot, rollback, and hardened external Hermes evidence remain incomplete: `docs/PLAN.md:16-20,43-61` and `docs/PHASE-2-HOST-HERMES.md:15-20,140-144`.

## Hermes snapshot, version, and maturity

### What is established

| Item | Provenance and status |
| --- | --- |
| Hermes is Nous Research's open-source agent platform | Official project identity: [NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent). The report identifies this interpretation at lines 3–9. |
| Official latest release `v2026.9.7`, internal version `0.21.1`, published 7 September 2026 | The release tag is the official snapshot and resolves to peeled commit `2237be355906fbe6065ce1815711eee52b2d646e` (`2237be3`), corroborated by the repository planning handoff at `docs/superpowers/plans/2026-09-11-milestone-1-foundation-closure.md:66-73` and the [official release tag](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.9.7). This establishes source identity only: it is not a Lumen certification. Curated feature notes below are intentionally deferred to the pinned-build probes and must not be treated as proof of behavior. |
| Lumen uses the documented API server surface | `docs/PHASE-2-HOST-HERMES.md:67-85` and the official [programmatic integration guide](https://hermes-agent.nousresearch.com/docs/developer-guide/programmatic-integration). |
| Existing Lumen adapter supports capabilities, health, Runs, status, SSE, approval, steer, stop | Repository fact: `internal/hermes/client.go:17-80,182-193,319-479`; `internal/hermes/events.go:16-190`. |
| Hermes is rapidly changing and pre-1.0 according to the report | Report claim at lines 9 and 77–79. Treat as an integration-risk signal, not a substitute for pinning and probes. |
| Official release status | **Latest tagged release.** Hermes production maturity and support guarantees are not established; Lumen still requires immutable source identity plus behavioral certification. |

### Feature inventory

The report describes the following Hermes families: agent loop, providers/models, sessions, profiles/personas, tools/toolsets, skills, MCP, browser/computer use, API/Runs, TUI/ACP, gateways, voice, memory providers, cron/loops, delegation/sub-agents, remote execution, plugins, A2A, and observability. The report's fit table begins at lines 11–23 and 249 onward; the official API boundary currently used by Lumen is narrower.

Feature availability must be discovered through authenticated `/v1/capabilities`, then certified behaviorally. Metadata is descriptive, never a Lumen grant.

## Complete feature matrix and Lumen boundary

| Hermes family | What Hermes may provide | What Lumen must own or constrain | Adoption milestone |
| --- | --- | --- | --- |
| Agent loop and orchestration | Reasoning, planning, tool choice, retries, compression, callbacks | Task authority, lifecycle, target, budget, deadline, cancellation, final outcome | M1 foundation; reused thereafter |
| Model/provider routing | Provider/model selection and fallback (provider-specific) | Approved provider, model, region, locality, retention, cost, data-class policy | M2 |
| Sessions and context windows | Runtime sessions and turn serialization | Canonical Conversation/Message records, session mapping, profile digest, context projection | M2 |
| Profiles and personas | Named agents, profile configuration, model/tool defaults | Immutable Lumen runtime profile; no identity, grant, approval, or target authority | M2 |
| Built-in tools and toolsets | Agent-facing tools | Lumen capability manifest, typed arguments, risk, grant, target, receipt | M2 design; cohort-by-cohort |
| Skills and plugins | Reusable agent behavior and executable extensions | Provenance, hashes/signatures, license, data access, sandbox/profile, rollback, kill switch | M2 registry; M9 cohorts |
| Context engine and curator | Context compression/caching, context files, and skill/memory curation; Hermes may auto-apply curator changes | Host projection policy, classification filters, retention, digest, and deterministic truncation; **Lumen treats** curator output as an untrusted proposal | M2; M8/M9 qualification |
| Hooks and trace exporters | Lifecycle hooks, middleware, metrics, and provider/plugin callbacks | Fail-closed policy hooks, redacted Host events, bounded callback time, and no authority through telemetry | M2/M7; provider-specific |
| Projects and worktrees | Project-root context, checkpoints, isolated Git worktrees, and batch refactors | Explicit workspace capability, immutable worker profile, path confinement, artifact receipts, and rollback | M8/M9 |
| MCP client/server integration | Typed external tools/resources over stdio or HTTP | Lumen broker is the normal physical-device gateway; no direct canonical-state writes | M2 design; M4 first action |
| Browser/computer use | Browser navigation, extraction, computer-use workflows | Allowlisted read-only start; protected profile reference; no ambient cookies/passwords; exact approval for writes/submits | M9 or separately certified cohort |
| HTTP API and Runs | Run submission, status, SSE events, cancellation, steering, approvals, discovery | Versioned adapter, durable create intent, one Host SSE consumer, reconciliation, bounded output | M1 |
| TUI/ACP/JSON-RPC | Rich local UI or IDE control | Optional surface adapter only; never bypass Host authority or canonical state | Later surface; not foundation |
| Messaging gateways | Telegram/Discord/Slack/WhatsApp/Matrix/etc. mechanics | Provider identity/thread mapping, canonical conversation, policy, approvals, dedupe, receipts, disconnect | M6 |
| Voice/STT/TTS | Voice orchestration, streaming, transcription, speech, barge-in | Node-local preactivation privacy, OS permissions, presence state, interruption, provider policy, cancellation lineage | M5 |
| Vision and media generation | Model-backed media capabilities | Typed capability, content classification, retention, artifact digest, approval, output limits | M9 cohort |
| External memory providers | Retrieval, synchronization hooks, semantic memory | Host-owned typed memory, proposal validation, retention/deletion, scope and classification | M2 |
| Native Hermes memory files/SQLite | Single-Hermes-home operational memory | Never the distributed Space database; do not synchronize live Hermes homes | M2 boundary |
| Cron and schedules | One-shot/recurring triggers and delivery mechanics | Host-owned Automation/Schedule, target action, expiry, grant, retry, audit, clock policy | M8 |
| Goals, loops, heartbeats, batch/kanban work | Background orchestration | Suggestion → preview → approval → narrow revocable grant; no background work by default | M8 |
| Delegated/sub-agents | Parallel child runs and specialized agents | Entire transitive tools, credentials, context, target, provider, budget, deadline, cancellation surface | M8 |
| Remote execution backends (SSH, Docker, Modal, etc.) | Worker execution environments (availability and egress differ by backend) | Immutable worker profile; coding worktree/sandbox; no generic device or shell authority | M8/M9 |
| Tool Gateway and Portal | Hosted web search, image generation, TTS, browser, and model/provider routing | Entitlement, provider/data policy, spend limits, egress, and output classification | M2 design; M9 cohorts |
| Dashboard, desktop, and Bot Mode | Management UI, profiles, named bots, routines, and gateway controls | Replaceable surface; all state changes use Host commands and receipts | M6/M8; later surface |
| Security, secrets, and vault integrations | Approval modes, secret prompts/stores, and optional security skills | Lumen credential handles, one-time grants, redaction, revocation, and policy authority | M1/M2; provider-specific |
| Egress and updating | Backend-specific proxying, image/install updates, snapshots, and rollback helpers | Pinned artifact identity, egress certification, migration/rollback, and no mutable dependency in authority path | M1/M7 |
| Platform/distribution support | macOS, Windows, Linux/WSL2, Docker, Android/Termux, and Tier 2 variants | Per-topology behavioral qualification; platform support does not imply isolation or feature parity | M1/M7 |
| A2A/agent peers | Agent-to-agent interoperability | Defer as peer-agent feature; node identity and deterministic device control remain Lumen protocols | M9 |
| Observability | Runtime events, logs, usage, health and traces (hooks/traces are provider- or plugin-specific) | Redacted Host audit, content-free telemetry, authority records, actionable health states | M1/M2; M7 operations |
| Android/Termux support | Runtime on Android/Termux with dependency differences | Same Host contract; same-UID co-location is compatibility-only, not hardening | M1 deployment evidence |
| Remote/private network access | Hermes network tools and MCP endpoints | Keep SSRF protections; use broker and authenticated node transport rather than private URLs from browser tools | M3/M4 |

**Inference from the reviewed official feature surfaces:** Hermes does not document a first-class durable personal-device registry, deterministic heterogeneous placement, Lumen-style capability ACLs, or distributed Space state. Verify this inference against future releases; absence from the reviewed docs is not proof that no experimental facility exists. These remain Lumen responsibilities. Report support: lines 185–194, 656–672. Repository contracts: `docs/ARCHITECTURE.md:31-39,99-103,146-158` and `docs/PLAN.md:78-105`.

### Official feature-family URL map

These are primary Hermes documentation URLs used for the feature snapshot. They are pointers for rechecking the pinned build, not promises that every option is enabled or compatible with Lumen.

| Family | Primary official Hermes reference | Boundary note |
| --- | --- | --- |
| API and Runs | [API server](https://hermes-agent.nousresearch.com/docs/user-guide/features/api-server); [programmatic integration](https://hermes-agent.nousresearch.com/docs/developer-guide/programmatic-integration) | Adapter input/output evidence only; Host owns task state. |
| MCP | [MCP integration](https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp); [use MCP with Hermes](https://hermes-agent.nousresearch.com/docs/guides/use-mcp-with-hermes) | Brokered capabilities only; no direct canonical-state writes. |
| A2A | [A2A](https://hermes-agent.nousresearch.com/docs/user-guide/messaging/a2a) | Agent-peer interoperability, not Lumen node authority. |
| Memory and providers | [Memory](https://hermes-agent.nousresearch.com/docs/user-guide/features/memory); [provider integrations](https://hermes-agent.nousresearch.com/docs/integrations/providers); [memory provider plugins](https://hermes-agent.nousresearch.com/docs/developer-guide/memory-provider-plugin) | Provider hooks such as `prefetch`, `sync_turn`, and `on_memory_write` are provider-specific; Lumen treats every retrieved or written item as an untrusted proposal and owns accepted memory. |
| Voice and wake | [Voice mode](https://hermes-agent.nousresearch.com/docs/user-guide/features/voice-mode); [wake word](https://hermes-agent.nousresearch.com/docs/user-guide/features/wake-word) | Lumen's preactivation privacy invariant is stricter than a remote capture feature. |
| Browser and computer use | [Browser automation](https://hermes-agent.nousresearch.com/docs/user-guide/features/browser); [computer use](https://hermes-agent.nousresearch.com/docs/user-guide/features/computer-use) | Protected profiles and exact approvals; no ambient device credentials. |
| Cron, goals, kanban, delegation | [Codex app-server runtime](https://hermes-agent.nousresearch.com/docs/user-guide/features/codex-app-server-runtime); [tools and toolsets](https://hermes-agent.nousresearch.com/docs/user-guide/features/tools/) | Schedules and child work require Host-owned grants, expiry, and cancellation. |
| Security, secrets, vault | [Configuration and security](https://hermes-agent.nousresearch.com/docs/user-guide/configuration); [secrets command](https://hermes-agent.nousresearch.com/docs/user-guide/secrets/command); [credential vault](https://hermes-agent.nousresearch.com/docs/user-guide/features/credential-vault); [1Password security skill](https://hermes-agent.nousresearch.com/docs/user-guide/skills/optional/security/security-1password) | Hermes secret stores/skills are not a Lumen vault or grant authority. |
| Tool Gateway and Portal | [Tool Gateway](https://hermes-agent.nousresearch.com/docs/user-guide/features/tool-gateway); [Nous Portal](https://hermes-agent.nousresearch.com/docs/integrations/nous-portal) | Current docs describe four default gateway tools; optional/provider-specific backends must be negotiated. |
| Dashboard, desktop, Bot Mode | [Web dashboard](https://hermes-agent.nousresearch.com/docs/user-guide/features/web-dashboard); [desktop](https://hermes-agent.nousresearch.com/docs/user-guide/desktop); [Bot Mode](https://hermes-agent.nousresearch.com/docs/user-guide/bot-mode) | Surfaces are replaceable; they cannot mutate Space state outside Host commands. |
| Platform support | [Platform support matrix](https://hermes-agent.nousresearch.com/docs/getting-started/platform-support) | Tier and feature differences are deployment facts to certify, not inferred portability. |
| Egress and iron-proxy | [iron-proxy](https://hermes-agent.nousresearch.com/docs/user-guide/egress/iron-proxy); [egress internals](https://hermes-agent.nousresearch.com/docs/developer-guide/egress-internals) | In this snapshot the documented proxy wiring is Docker-only; do not generalize it to SSH/Modal/Daytona/Singularity. |
| Updating | [Updating and uninstalling](https://hermes-agent.nousresearch.com/docs/getting-started/updating) | Pin and certify updates; Docker images are replaced, not updated in place. |
| Sessions and profiles | [Sessions](https://hermes-agent.nousresearch.com/docs/user-guide/sessions); [profiles](https://hermes-agent.nousresearch.com/docs/user-guide/profiles/); [configuration](https://hermes-agent.nousresearch.com/docs/user-guide/configuration) | Hermes session/profile state is replaceable and must not become canonical Space state. |
| Plugins and skills | [Plugins](https://hermes-agent.nousresearch.com/docs/user-guide/features/plugins); [skills](https://hermes-agent.nousresearch.com/docs/user-guide/features/skills) | Hash, provenance, scope, and rollback each installed extension. |
| Batch, hooks, projects, worktrees, context engine, curator | [Features overview](https://hermes-agent.nousresearch.com/docs/user-guide/features/overview/); [hooks](https://hermes-agent.nousresearch.com/docs/user-guide/features/hooks); [CLI commands](https://hermes-agent.nousresearch.com/docs/reference/cli-commands); [Git worktrees](https://hermes-agent.nousresearch.com/docs/user-guide/git-worktrees); [context engine plugin](https://hermes-agent.nousresearch.com/docs/developer-guide/context-engine-plugin); [curator](https://hermes-agent.nousresearch.com/docs/user-guide/features/curator) | Batch/background behavior, hooks, workspace mutations, and context engines are provider/runtime features; Lumen treats them as bounded evidence and recommends explicit policy wrappers. |

The official docs are not a substitute for release-pinned checks. In particular, feature counts, optional plugins, backend support, and endpoint schemas can vary between the documentation deployment and `v2026.9.7`.

## Target architecture

### Ownership and trust

```text
Surfaces: web · messaging · voice · native · companion · owner CLI
                              │
                              ▼
                    Space Authority / Host
 identity · conversations · accepted memory · policy · grants
 approvals · routing · tasks · automation · audit · encrypted state
             │                                      │
             ▼                                      ▼
 Hermes Runtime Adapter v1                    Node capability fabric
 Runs · models · tools · skills · MCP          paired identities · capabilities
 gateways · voice · workers                    local policy · OS permissions
```

The Host is active and authoritative. Hermes, models, tools, MCP servers, relays, nodes, external messages, and provider callbacks are untrusted evidence. See `docs/DESIGN-PRINCIPLES.md:27-37` and `docs/ARCHITECTURE.md:31-49`.

### Mac combined path

Recommendation for the first practical combined topology:

1. Run `lumen-host serve` and Hermes as separately supervised processes or containers on one Mac.
2. Keep the canonical encrypted Space store and operator Unix socket in the Host-owned private data directory.
3. Keep Hermes API loopback/protected; give it a distinct bearer credential stored in a restricted file.
4. Use a local MCP broker boundary; the broker calls Host policy, not arbitrary device URLs.
5. If Hermes worker isolation is required, use a Docker/worker profile with explicit limits and scoped credentials. A container is evidence only when the OS boundary and deployment configuration actually enforce it.
6. Treat `lumen doctor` as `ready` only after live Host state, supervisor state, Hermes authentication/capabilities, and required behavior pass; a socket or HTTP status alone is insufficient.

This aligns with `docs/PHASE-2-HOST-HERMES.md:53-65,81-89` and the combined/deployment rules in `docs/ARCHITECTURE.md:67-76`. A Mac local-loopback run is development or personal evidence until its isolation and recovery gates are met.

### External VPS path

Recommendation for a separated or hardened path:

1. Keep Host and canonical encrypted storage together on the VPS.
2. Run Hermes in a separate OS principal or container, preferably with a bounded worker backend.
3. Connect Host to Hermes using HTTPS with CA/hostname/certificate-validity verification, TLS 1.3, pinned leaf identity, client certificate authentication, and a separate bearer credential.
4. Reject redirects, URL userinfo, pin changes, and credential forwarding to another origin.
5. Persist Host create intent and profile digest before a run; never treat Hermes completion or a supervisor callback as authority.
6. For remote nodes, use the versioned Lumen envelope and paired mTLS node transport. A relay may forward opaque encrypted envelopes but has no Space keys, grants, or completion authority.

These are repository requirements, not report assumptions: `internal/hermes/client.go:209-317,588-615`, `docs/PHASE-2-HOST-HERMES.md:81-85`, `docs/O-003-TRANSPORT.md:11-28`, `docs/ARCHITECTURE.md:67-76`.

### Egress and iron-proxy caveat

**Official Hermes fact, snapshot-scoped:** the documented `iron-proxy` egress path is wired into Docker workers in this release. The official egress notes do not establish equivalent proxy environment/CA injection for SSH, Modal, Daytona, or Singularity backends. Therefore “Hermes has an egress proxy” is not evidence that every worker or provider is forced through the proxy, and it is not a substitute for Lumen credential scoping or outbound policy. Lumen should permit a backend only after capability negotiation and a backend-specific egress probe; otherwise mark egress isolation unavailable. See the [iron-proxy](https://hermes-agent.nousresearch.com/docs/user-guide/egress/iron-proxy) and [egress internals](https://hermes-agent.nousresearch.com/docs/developer-guide/egress-internals) references.

### Host-local and cross-node flows

Host-local Runs flow: authenticated owner request → durable Lumen command/create intent → capability and context validation → Hermes capability/readiness check → `POST /v1/runs` → durable task/run mapping → sole Host SSE consumer → status reconciliation → Host transition and receipt. See `docs/ARCHITECTURE.md:78-87` and `internal/host/execution.go:649-700,844-925,1048-1128`.

Cross-node flow: signed intent → Host persistence and policy → deterministic target selection → authenticated node invocation → target-local revalidation → normalized evidence → Host task transition and receipt. Hermes may propose an action, but cannot select an unapproved target or bypass node checks.

## Versioned `HermesRuntimeAdapter v1` contract

### Boundary

The existing `internal/hermes.Adapter` is the narrow evidence contract:

```go
Capabilities(ctx) (Capabilities, error)
Health(ctx) (Health, error)
CreateRun(ctx, CreateRunRequest, idempotencyKey) (Run, error)
RunStatus(ctx, runID) (Run, error)
Events(ctx, runID) ([]Event, error)
ResolveApproval(ctx, runID, decision) error
Steer(ctx, runID, SteerRequest) error
Stop(ctx, runID) (Run, error)
```

Source: `internal/hermes/client.go:182-193`. Keep this adapter free of Space persistence and policy. Host composition remains in `internal/host`.

### Contract record

Recommendation: persist a compatibility record alongside setup/runtime configuration:

```text
contract: lumen.hermes.runtime/v1
hermes_source: official repository or externally certified endpoint
hermes_tag_and_peeled_commit: immutable identity
runtime_version_and_lockfile_digest
required_features: run_submission, run_status, run_events_sse, run_approval, run_stop
optional_features: run_steer
auth: bearer plus deployment-specific TLS identity
event_status_mapping: queued/started/running/awaiting_approval/completed/failed/cancelled
bounds: response/event/stream bytes and event count
profile_digest
qualified_at and evidence_reference
```

The record is a compatibility certificate, not a grant. It must be included in the Host task/run mapping and approval binding.

### Request and response rules

- `GET /v1/capabilities` is mandatory for negotiation.
- `/health` is liveness evidence; readiness must include authenticated identity and required behavior.
- `POST /v1/runs` receives only task input, runtime session mapping, approved model/provider constraints, and projected context. It must carry the stable idempotency key.
- `GET /v1/runs/{id}` is authoritative only as dependency evidence; Host commits the resulting transition.
- `/events` is bounded and lossy evidence. Only one Host consumer is permitted; duplicates and reordering cannot regress canonical state.
- **Official Hermes fact for this snapshot:** the Runs SSE surface supports clients attaching/detaching without losing run state, and unconsumed event buffers expire after five minutes. Lumen must treat attach/detach and that five-minute buffer as dependency behavior to probe and pin, not as durable task history; status polling and Host reconciliation remain required. See [API server Runs/events](https://hermes-agent.nousresearch.com/docs/user-guide/features/api-server).
- `/approval`, `/steer`, and `/stop` are sent only after Host authorization. Approval accepts only `once` or `deny`; `steer` is optional and a successful steer means queued input, not completed work.
- Credentials never enter prompts, ordinary logs, task output, or diagnostics.

Implementation evidence: `internal/hermes/client.go:319-479`, `internal/hermes/events.go:24-31`, `docs/PHASE-2-HOST-HERMES.md:69-85,91-109`.

### Non-negotiable Lumen invariants

These are Lumen invariants even when Hermes advertises a more permissive or richer surface:

1. **Capability negotiation is mandatory before use.** No tool, provider, event mode, backend, or optional operation is callable merely because a configuration file names it.
2. **There is one Host SSE consumer.** Fan-out, UI streaming, and reconnect logic consume Host-owned normalized evidence; they do not open competing authority streams to Hermes.
3. **SSE and runtime output are lossy evidence.** Drops, duplicates, reordering, truncation, and disconnects require bounded reconciliation and may end in `unknown_outcome`; an event, log, model response, or adapter result never proves a canonical transition.
4. **Host authority is non-delegable.** Hermes cannot grant, approve, select a device, commit memory, assert completion, or widen context/capability scope.

### Failure semantics

| Failure | Required Lumen result |
| --- | --- |
| Incompatible capabilities before dispatch | Bounded rejection/degraded evidence; no run created |
| Proven HTTP rejection | `failed`/unavailable; safe digest-only error |
| Timeout or transport failure during create | `unknown_outcome` unless creation is proven rejected or recovered by durable idempotency evidence |
| Event disconnect after dispatch | Bounded status reconciliation; terminal result or `unknown_outcome` |
| Duplicate key and same digest | Return durable prior result |
| Duplicate key and different digest | Reject collision |
| Host restart with no run mapping | Do not redispatch; recover/reconcile or record unknown |
| Cancellation race | First valid terminal transition wins under compare-and-set |

## Versioned `Lumen Capability Broker/MCP v1` contract

### Minimal surface

Recommendation: expose only these initial tools:

```text
lumen.capabilities.list
lumen.capability.request
```

`list` returns redacted, untrusted capability inventory. `request` submits an intent containing `space_id`, `request_id`, `capability_id`, `action`, typed `arguments`, target constraints (not an authority-bearing target), context scope, and expiry.

The broker then uses the normal Host command path. It must not expose `lumen.state.put`, arbitrary HTTP device endpoints, arbitrary shell, or unrestricted broadcast. It must not issue grants, approve itself, mutate canonical memory, or consume unrelated context.

### Invocation processing

1. Authenticate the broker/Hermes principal with a credential separate from operator, Host-state, Hermes API, and node credentials.
2. Validate protocol and capability schema version; reject unknown security-critical fields.
3. Canonicalize arguments and compute an action fingerprint.
4. Let Host policy evaluate grant (`deny`, `ask`, `allow`), capability, context scope, target eligibility, health, epoch, expiry, budget, and approval.
5. Persist the command before dispatch.
6. Return a task reference and honest state (`queued`, `awaiting_permission`, `failed`, `unknown_outcome`, or terminal receipt).
7. Treat tool/node output as untrusted data; annotate or bound it before returning it to Hermes.

The existing common contract requires `space_id`, `task_id`, capability/action, typed arguments, origin/target, idempotency, issue/expiry, and grant reference: `docs/O-006-CAPABILITIES.md:12-33`.

### Transport and isolation

Start with local stdio or a protected Unix-domain broker. Do not enable Hermes browser private-URL access merely to reach nodes. Remote MCP and node transport come after mTLS pairing, replay defense, epoch validation, and conformance fixtures. The accepted first node transport is mDNS discovery as a hint plus one mutually authenticated encrypted channel: `docs/O-003-TRANSPORT.md:11-28,72-102`.

## Context, sessions, personas, and memory

### Context projection

Host context is typed records with namespace, schema version, origin node, logical version, classification, retention, digest, and payload only when allowed. The selected level is a maximum and can only be reduced by classification policy:

| Level | Hermes/broker may receive |
| --- | --- |
| `none` | No user context or free-text result; operational receipt only |
| `metadata` | Header, namespace, classification, timestamps, origin, digest, coarse size/type |
| `summary` | Metadata plus bounded labelled summary |
| `content` | Declared typed payload, still excluding forbidden secrets/classifications |

Contract: `docs/O-002-CONTEXT.md:14-31`. Apply filtering before serialization and again at Host persistence. A capability's action approval does not imply permission to retain its content.

### Sessions and personas

```text
Lumen Conversation       → replaceable Hermes session_id
Lumen persona/profile     → immutable runtime profile digest
Lumen owner identity      → never a Hermes session/profile authority
```

Runtime replacement, session loss, or Hermes failure cannot delete canonical conversations. A profile may constrain tools, models, providers, context, budget, locality, and retention; it cannot create grants, change target selection, or approve an action. Relevant requirements: `docs/PLAN.md:63-76`, `docs/ARCHITECTURE.md:146-158`.

### Memory

Hermes memory providers expose provider-specific retrieval and synchronization hooks. As a Lumen adapter rule, every retrieved item or attempted write is converted into an untrusted memory proposal. Lumen accepts and persists canonical memory only after provenance, classification, retention, scope, confidence, and policy validation. Sensitive or consequential memory requires confirmation. Deletion and retention expiry require durable tombstones and must exclude the record from future projections.

Never synchronize live Hermes homes or use Hermes SQLite/files as the distributed Space database. Provider outage, profile replacement, or Hermes upgrade must not alter Lumen canonical state. Repository basis: `docs/DECISIONS.md:D-028,D-046`, `docs/DESIGN-PRINCIPLES.md:59-65`, `docs/O-002-CONTEXT.md:58-70`; report discussion: lines 171–179 and 535–568.

## Security and threat model

The adapter must preserve these repository threats and invariants:

| Threat | Required control | Reference |
| --- | --- | --- |
| Runtime expands grant or context | Default deny, typed broker, Host policy outside adapter | `docs/THREAT_MODEL.md:T-005,T-007,T-008` |
| Runtime endpoint redirects/SSRFs | HTTPS/mTLS/pin, no redirects/userinfo, broker instead of private browser URLs | `docs/THREAT_MODEL.md:T-018`; report lines 217–245 |
| Approval bypass/substitution | Exact task/action/arguments digest/actor/target/expiry; one-time `once`/`deny` only | `internal/space/execution.go:135-216`; `docs/PHASE-2-HOST-HERMES.md:102` |
| Event loss or forged completion | Sole Host SSE consumer, bounded reconciliation, Host-owned transition | `internal/hermes/events.go:24-31`; `docs/PHASE-2-HOST-HERMES.md:85,99-103` |
| Delegation escapes parent | Immutable transitive worker profile; disable if aggregate grant cannot be verified | `docs/ARCHITECTURE.md:162-174`; `docs/THREAT_MODEL.md:T-019` |
| Provider/model misrouting | Seal provider/model/region/data class into profile before projection | `docs/THREAT_MODEL.md:T-020`; `docs/PLAN.md:69-74` |
| Secrets leak | Restricted files, scoped handles, redacted diagnostics, separate credentials | `internal/host/execution.go:92-154`; `internal/control/protocol.go:95-157` |
| Same-UID Termux false isolation | Compatibility-only co-location; hardened proof requires isolated endpoint | `docs/PHASE-2-HOST-HERMES.md:81-85` |
| Split brain/stale Host | Monotonic epoch, explicit migration, reject stale envelopes | `docs/O-005-RECOVERY.md:5-30` |

Security-critical invariants are summarized in `docs/THREAT_MODEL.md:108-145` and `docs/DESIGN-PRINCIPLES.md:27-71`.

## Compatibility negotiation and behavioral certification

Capabilities metadata alone is insufficient. Certification should be a two-part record:

1. **Static identity:** official repository, immutable tag/peeled commit, lockfile/runtime versions, artifact digest, adapter contract version, profile digest, and deployment topology.
2. **Behavioral probes:** authenticated health, capability discovery, idempotent create, status, sole SSE consumer, duplicate/reordered events, disconnect/reconcile, exact `once`/`deny` approval, stop, bounded output, safe errors, credential separation, and restart recovery.

Negotiation must reject unsupported versions, downgrade, altered security fields, missing required features, endpoint identity mismatch, and advertised-but-broken features. `run_steer` remains optional and must be checked immediately before use.

The repository explicitly records known upstream behavioral concerns—approval bypass under safe defaults and single-consumer event behavior—and requires a real-Hermes gate: `docs/PHASE-2-HOST-HERMES.md:85`. Existing fake-server tests are useful but do not prove a real configured runtime: `internal/hermes/client_test.go:31-529` and `docs/PHASE-2-HOST-HERMES.md:107-114`.

## Feature-to-milestone adoption matrix

| Milestone | Hermes adoption | Lumen-owned exit condition |
| --- | --- | --- |
| M1 Reliable combined foundation | Runs/API, capability negotiation, events, approval, stop, optional steer, health | Clean setup, pinned artifacts, reboot, rollback, real configured behavior, honest recovery; **not currently complete** |
| M2 Conversation/memory nucleus | Sessions, profiles/personas, model routing, external memory retrieval, normalized runtime events, component registry | Canonical continuity and inspectable/deletable memory survive runtime replacement |
| M3 Secure node fabric | Hermes remains host runtime; broker boundary is prepared | Pairing, mTLS, replay/freshness, epoch, advertisements, revocation, deterministic selection |
| M4 First cross-node action | MCP/tool call to restricted node capability | `files.search`, `files.read`, harmless delivery; local revalidation and honest receipt |
| M5 Voice/presence | Hermes voice orchestration, STT/TTS, streaming/barge-in where certified | Local preactivation privacy, visible states, stop, permissions, handoff |
| M6 Messaging continuity | One Hermes gateway | Canonical identity/thread mapping, dedupe, delivery receipt, disconnect/revocation |
| M7 Managed/portable beta | Runtime and gateway deployment per dedicated data plane | Migration/export, content-free control plane, backup/deletion, upgrade rollback |
| M8 Earned autonomy | Cron, loops, goals, heartbeats, delegation, workers, remote backends | Graduated grants, transitive confinement, pause/revoke, restart and uncertainty |
| M9 Ecosystem/public launch | Additional gateways, browser/computer use, MCP/skills, vision/media, A2A | Public schemas/SDKs/fixtures and component qualification without weakening authority |

Canonical sequencing: `docs/PLAN.md:43-61,63-76,78-105,107-169`.

## Correcting the supplied report

The report's useful observations are at lines 11–23, 150–194, 217–245, 656–672, and 674–861. Correct these recommendations before adoption:

| Report recommendation/claim | Correction for Lumen |
| --- | --- |
| Hermes owns AI-facing memory retrieval | Hermes may retrieve a Host-approved projection; Host owns accepted memory, retention, deletion, and classification. |
| Add `lumen.state.put` | Prohibited. Canonical state changes must use Space transitions and encrypted Host persistence. |
| Generic `lumen.call(node, capability, args)` | Replace with a capability intent. Host resolves target and validates grant, health, epoch, scope, approval, and expiry. |
| Add `lumen.broadcast` | Defer. Fan-out needs an explicit action contract, per-target authorization, dedupe, receipts, and reconciliation. |
| “Agent chooses capability; broker resolves node” | Hermes can propose intent/constraints; Host validates capability and deterministically selects or asks on ambiguity. |
| NATS/MQTT early device bus | Accepted baseline is paired mDNS discovery plus mTLS channel. A bus is a later transport option and cannot own authority. |
| SSH for quick cross-host execution | Compatibility/developer evidence only; production coding uses isolated worktrees/sandboxes and narrow capabilities. |
| Synchronize Hermes homes/provider workspaces | Never sync live Hermes homes. External memory is retrieval/proposal only and must obey Lumen context policy. |
| Direct browser/private-device HTTP | Preserve SSRF controls; route through the typed broker and authenticated node channel. |
| Loopback API key is sufficient | It is necessary but not sufficient for hardened separation. Use pinned mTLS, separate credentials, endpoint confinement, and behavioral certification. |
| Full Hermes on every phone | Prefer a thin node/client. Same-UID Termux Hermes is not isolation evidence. |
| Provider/hook/trace surfaces are universal | Provider adapters, memory backends, hooks, and trace/telemetry exporters are provider-, plugin-, or deployment-specific. Lumen may recommend a redacted Host event/trace adapter, but must not assume a stable Hermes-wide trace contract. |
| Two-to-four-week MVP estimates | Estimates are not evidence and conflict with the repository's still-open M1 gates. |
| A2A as device protocol | Defer A2A for agent peers; deterministic node control uses Lumen protocol and broker. |

## Mac combined and external VPS deployment checklist

| Check | Mac combined | External VPS/separated |
| --- | --- | --- |
| Host/store | Same Mac, private encrypted data directory | Same VPS data plane; Host remains canonical owner |
| Hermes process | Separate supervisor/container; loopback protected | Separate principal/container; protected HTTPS endpoint |
| Credentials | Distinct operator, state, Hermes, and broker credentials | Same separation, plus client cert and server leaf pin |
| Runtime profile | Immutable tools/providers/context/budget/digest | Same, plus endpoint/topology binding |
| Readiness | Live state + supervisor + authenticated Hermes probes | Same plus TLS identity, pin, and external endpoint probes |
| Node access | Local broker first | Versioned mTLS envelopes; relay only opaque transport |
| Evidence status | Development/personal until gates pass | Hardened only after isolation, mTLS, rollback, and recovery evidence |

Do not infer release readiness from a local process restart, socket presence, synthetic state, or HTTP health alone. `docs/superpowers/specs/2026-09-11-milestone-1-dual-topology-design.md:91-124` defines the truthful doctor and deployment requirements.

## Exhaustive negative and recovery test matrix

### Adapter and negotiation

- Missing required feature; optional `run_steer` absent must produce unsupported without a request.
- Unknown security-critical capability fields, unsupported version, downgrade, malformed auth metadata.
- Advertised feature fails behavioral probe.
- Invalid bearer, credential source failure, credential rotation.
- Redirect, URL userinfo/query/fragment, wrong origin, private endpoint confusion.
- TLS version below 1.3, wrong CA, hostname, validity, client certificate, or leaf pin.
- Hermes credential appears in arguments, prompts, logs, diagnostics, or error bodies.

### Runs and events

- Same idempotency key and digest returns prior durable result.
- Same key with changed digest rejects collision.
- Proven 4xx rejection versus timeout/ambiguous create classification.
- Missing/invalid run ID, status, output encoding, content type, trailing JSON.
- Oversized response, event, stream, event count, and terminal output.
- Unknown SSE field, malformed JSON, malformed retry field, incomplete record.
- Duplicate/reordered/ID-less events do not repeat or regress transitions.
- Second SSE consumer is rejected or prevented from becoming authority.
- Disconnect after dispatch reconciles through bounded status polling.
- Forged terminal SSE event is checked against run status.
- Host dies before mapping, after mapping, during event persistence, and during reconciliation; no unsafe redispatch.

### Approval, steer, and cancellation

- Mutated task, action, canonical arguments digest, actor, target, run, profile digest, or expiry is rejected.
- `session`/`always` approval values are rejected; only `once`/`deny` forward.
- Expired, reused, concurrent, or double-consumed approval is rejected.
- Approval forwarding fails after request; state becomes uncertain and is reconciled.
- Steer absent, unsupported, queued-but-not-complete, or sent after terminal state.
- Stop failure, timeout, and completion/cancellation race produce one honest terminal result.

### Broker/MCP and context

- Unauthorized broker principal/token, wrong scope, replay, stale expiry, stale Host epoch.
- Arbitrary target/capability/action/resource/credential request.
- Direct `state.put`, self-grant, self-approval, or grant-widening attempt.
- Prompt injection through Hermes, MCP, node, provider, file, or message output.
- Context exceeds `none`, `metadata`, `summary`, or `content` policy.
- Deleted/tombstoned context reappears; conflicting offline versions silently overwrite.
- Secrets, raw prompts, private files, credentials, or raw audio leak to broker results/logs/traces.

### Sessions, personas, memory, and providers

- Cross-conversation/session mix-up and profile digest substitution.
- Concurrent same-session writes and stale session lineage.
- Hermes memory write auto-promotes to canonical memory.
- Cross-profile/provider memory contamination.
- Provider unavailable, changed, or region-disallowed; no unauthorized classified payload leaves Host.
- Runtime replacement, Hermes restart, or memory-provider deletion cannot erase canonical records.

### Nodes and deployment

- Forged/replayed/expired node envelope, wrong Space/recipient/epoch, revoked node, duplicate command.
- Malicious capability advertisement, stale health, ambiguous target, offline target.
- Node-local policy denies an invocation even when Host allowed it.
- Termux same-UID co-location is never counted as hardened isolation.
- Relay reads, forges, reorders, or asserts completion for an opaque envelope.
- Host migration interruption leaves exactly one active epoch; stale Host rejects work.
- Artifact tampering, mutable source, license failure, compatibility regression, rollback failure.

### Voice, messaging, automation, and delegation

- Zero preactivation audio/transcript/activation metadata/model/tool traffic.
- False wake, locked device, revoked microphone grant, background capture, stop latency, barge-in, network loss.
- Forged messaging identity/thread, duplicate inbound/delivery, disconnect/revocation, uncertain provider delivery.
- Clock rollback, DST/timezone ambiguity, unbounded schedule retry, expired automation.
- Child profile mutation, transitive tool/credential/context/target/provider expansion, recursive delegation, child self-approval, child work after parent cancellation.

The matrix follows `docs/THREAT_MODEL.md:T-001-T-029`, `docs/O-002-CONTEXT.md:66-71`, `docs/O-003-TRANSPORT.md:80-102`, `docs/O-006-CAPABILITIES.md:119-138`, and `docs/PHASE-2-HOST-HERMES.md:105-114`.

## Prioritized implementation roadmap

1. **P0 / active M1:** finish public setup/doctor/service, immutable Hermes artifacts, configured-runtime certification, Runs v1, sole SSE consumer, exact approvals, stop/cancel, reconciliation, restart, rollback, secret redaction, and combined deployment evidence.
2. **P1 / M2:** add canonical conversations, memory proposals/records, session/persona mapping, context projection, provider/model constraints, runtime event normalization, and component registry.
3. **P2 / M3:** publish node envelope/pairing/capability/invocation/event/receipt contracts; implement mTLS, replay/freshness, epoch, revocation, and deterministic selection.
4. **P3 / M4:** connect the broker to one lightweight node with `files.search`, `files.read`, and one harmless notification/delivery capability.
5. **P4 / M5–M6:** add local preactivation voice and postactivation streaming, then one Hermes messaging gateway mapped into canonical conversations.
6. **P5 / M7:** add dedicated managed data planes, encrypted export/import and migration, content-free telemetry, entitlement isolation, and rollback.
7. **P6 / M8–M9:** add earned autonomy, delegation, workers, browser, skills, MCP cohorts, A2A peers, public SDKs, and ecosystem conformance after transitive authority and component qualification.

## Explicit unknowns and conflicting counts

- The official architecture page currently states “70+ tools” and “28 toolsets.” Provider, tool, and messaging-platform totals vary by page, release, configuration, and enabled plugins; they are snapshot descriptions rather than Lumen contracts. The supplied report's aggregate counts therefore require independent verification before reuse.
- The official latest release snapshot is `v2026.9.7`/`0.21.1`, published 2026-09-07 and pinned here to peeled commit `2237be3`; the repository handoff's “candidate” wording describes Lumen planning status, not a Hermes release status. No real-runtime Lumen certification is recorded yet. Curated feature notes remain deferred until probes run against that exact immutable build.
- Remote-backend documentation has two scopes that must not be conflated: the official tools and architecture pages list seven execution backends (local, Docker, SSH, Daytona, Modal, Singularity, Vercel Sandbox), while the egress documentation only establishes `iron-proxy` wiring for Docker in this snapshot. Backend availability, persistence, and egress isolation must be negotiated and tested per backend.
- The report describes API, TUI gateway, ACP, Python, MCP, messaging, A2A, voice, cron, memory, and remote-execution surfaces. The exact feature flags, wire schemas, event names, approval behavior, and concurrency semantics must be read from the pinned Hermes build and tested; do not infer from prose or old RFCs.
- Hermes's durable storage and memory implementation details may change. Lumen must not couple canonical schemas to Hermes SQLite, profile files, or internal Python objects.
- The report recommends NATS/MQTT, Honcho/OpenViking/Mem0/Hindsight, Piper/Kokoro, faster-whisper, and other components. Provider choices, hooks, traces, and these components are not universal Hermes contracts and none is an adopted Lumen dependency; each requires provenance, commercial-rights, security, data-flow, benchmark, fallback, rollback, and kill-switch review under `docs/DESIGN-PRINCIPLES.md:75-77` and `docs/PLAN.md:171-179`.
- Exact Mac Docker and external VPS process/supervisor settings remain deployment work, not this architecture certificate. `docs/PHASE-2-HOST-HERMES.md:116-124` records development evidence only.

## Sources

1. [Hermes Agent official repository](https://github.com/NousResearch/hermes-agent) — primary source repository and release identity.
2. [Hermes programmatic integration](https://hermes-agent.nousresearch.com/docs/developer-guide/programmatic-integration) — API, Runs, sessions, events, and capability discovery boundary.
3. [Hermes tools documentation](https://hermes-agent.nousresearch.com/docs/user-guide/features/tools/) — official tool/capability guidance referenced by Lumen decision D-026.
4. [Hermes releases](https://github.com/NousResearch/hermes-agent/releases) — primary release history; verify the pinned tag and commit before certification.
5. [`docs/PRD.md`](../PRD.md) — product requirements and launch capability cohort.
6. [`docs/ARCHITECTURE.md`](../ARCHITECTURE.md) — Host, Hermes, node, context, voice, and deployment boundaries.
7. [`docs/DESIGN-PRINCIPLES.md`](../DESIGN-PRINCIPLES.md) — authority, capability, failure, context, observability, reuse, and interoperability rules.
8. [`docs/DECISIONS.md`](../DECISIONS.md) — accepted decisions D-001 through D-048, including Hermes and transport boundaries.
9. [`docs/PLAN.md`](../PLAN.md) — active milestone sequencing and evidence gates.
10. [`docs/PHASE-2-HOST-HERMES.md`](../PHASE-2-HOST-HERMES.md) — current Host/Hermes foundation contract and release evidence status.
11. [`docs/THREAT_MODEL.md`](../THREAT_MODEL.md) — threats T-001 through T-029 and release-blocking evidence.
12. [`docs/O-002-CONTEXT.md`](../O-002-CONTEXT.md) — context synchronization levels and memory boundary.
13. [`docs/O-003-TRANSPORT.md`](../O-003-TRANSPORT.md) — paired-node transport, envelope, replay, and recovery contract.
14. [`docs/O-005-RECOVERY.md`](../O-005-RECOVERY.md) — explicit Host migration and monotonic epoch.
15. [`docs/O-006-CAPABILITIES.md`](../O-006-CAPABILITIES.md) — action-level capability, approval, receipt, and context contract.
16. [`internal/hermes/client.go`](../../internal/hermes/client.go), [`internal/hermes/events.go`](../../internal/hermes/events.go) — current Lumen Hermes adapter implementation.
17. [`internal/host/execution.go`](../../internal/host/execution.go), [`internal/space/apply.go`](../../internal/space/apply.go), [`internal/space/execution.go`](../../internal/space/execution.go) — Host orchestration and authoritative transitions.
18. Uploaded report: `/Users/ashwanthreddyboddireddy/Downloads/hermes-lumen-compatability.md` — untrusted due-diligence input; report claims are labelled and not treated as certification.
19. [Hermes `v2026.9.7` release tag](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.9.7) — official release identity; verify the peeled commit before certification.
20. [Hermes features overview](https://hermes-agent.nousresearch.com/docs/user-guide/features/overview/) — official feature-family index and date-sensitive editorial counts.
21. [Hermes MCP integration](https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp) and [MCP guide](https://hermes-agent.nousresearch.com/docs/guides/use-mcp-with-hermes) — MCP surfaces and configuration.
22. [Hermes A2A](https://hermes-agent.nousresearch.com/docs/user-guide/messaging/a2a) — agent-peer protocol, inbound/outbound behavior, and security controls.
23. [Hermes memory](https://hermes-agent.nousresearch.com/docs/user-guide/features/memory), [provider integrations](https://hermes-agent.nousresearch.com/docs/integrations/providers), and [memory provider plugin](https://hermes-agent.nousresearch.com/docs/developer-guide/memory-provider-plugin) — built-in and external memory boundaries.
24. [Hermes voice mode](https://hermes-agent.nousresearch.com/docs/user-guide/features/voice-mode) and [wake word](https://hermes-agent.nousresearch.com/docs/user-guide/features/wake-word) — voice, wake, and capture behavior.
25. [Hermes browser](https://hermes-agent.nousresearch.com/docs/user-guide/features/browser) and [computer use](https://hermes-agent.nousresearch.com/docs/user-guide/features/computer-use) — browser and desktop-control surfaces.
26. [Hermes Codex app-server runtime](https://hermes-agent.nousresearch.com/docs/user-guide/features/codex-app-server-runtime) and [tools/toolsets](https://hermes-agent.nousresearch.com/docs/user-guide/features/tools/) — cron, goals, kanban, delegation, and tool availability references.
27. [Hermes configuration/security](https://hermes-agent.nousresearch.com/docs/user-guide/configuration), [secrets command](https://hermes-agent.nousresearch.com/docs/user-guide/secrets/command), and [1Password security skill](https://hermes-agent.nousresearch.com/docs/user-guide/skills/optional/security/security-1password) — secret-handling and vault-adjacent integrations.
28. [Hermes Tool Gateway](https://hermes-agent.nousresearch.com/docs/user-guide/features/tool-gateway) and [Nous Portal](https://hermes-agent.nousresearch.com/docs/integrations/nous-portal) — gateway/provider-specific web, image, TTS, and browser tools.
29. [Hermes web dashboard](https://hermes-agent.nousresearch.com/docs/user-guide/features/web-dashboard), [desktop](https://hermes-agent.nousresearch.com/docs/user-guide/desktop), and [Bot Mode](https://hermes-agent.nousresearch.com/docs/user-guide/bot-mode) — replaceable user surfaces and named bots.
30. [Hermes platform support](https://hermes-agent.nousresearch.com/docs/getting-started/platform-support) — tiered OS, architecture, and installation support matrix.
31. [Hermes iron-proxy](https://hermes-agent.nousresearch.com/docs/user-guide/egress/iron-proxy) and [egress internals](https://hermes-agent.nousresearch.com/docs/developer-guide/egress-internals) — backend-scoped outbound proxy behavior.
32. [Hermes updating](https://hermes-agent.nousresearch.com/docs/getting-started/updating) — update, backup, rollback, and Docker image caveats.
33. [Hermes sessions](https://hermes-agent.nousresearch.com/docs/user-guide/sessions), [profiles](https://hermes-agent.nousresearch.com/docs/user-guide/profiles/), and [configuration](https://hermes-agent.nousresearch.com/docs/user-guide/configuration) — session persistence and profile isolation.
34. [Hermes plugins](https://hermes-agent.nousresearch.com/docs/user-guide/features/plugins), [skills](https://hermes-agent.nousresearch.com/docs/user-guide/features/skills), and [plugin developer guide](https://hermes-agent.nousresearch.com/docs/developer-guide/plugins) — extension, skill, and hook surfaces.
35. [Hermes hooks](https://hermes-agent.nousresearch.com/docs/user-guide/features/hooks), [CLI commands](https://hermes-agent.nousresearch.com/docs/reference/cli-commands), [Git worktrees](https://hermes-agent.nousresearch.com/docs/user-guide/git-worktrees), [context engine plugin](https://hermes-agent.nousresearch.com/docs/developer-guide/context-engine-plugin), and [curator](https://hermes-agent.nousresearch.com/docs/user-guide/features/curator) — batch/background, hooks, project/worktree, context, and curation references.
