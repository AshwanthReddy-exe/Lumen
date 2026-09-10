# Lumen Architecture

## Product system

```text
Web · voice · messaging · native apps · ambient companions · owner CLI
                                 │
                                 ▼
                    Space Authority / Host
 identity · conversations · memory · policy · approvals · tasks · audit
              │                                      │
              ▼                                      ▼
 Hermes-native capability platform              Node capability fabric
 agent loop · models · tools · skills          files · apps · screen · shell
 MCP · browser · gateways · voice              microphone · speaker · camera
 automation · delegation · workers                 notifications · local models
              │                                      │
              └───── scoped requests, events, and receipts ────┘
```

The Space is the product. Surfaces are replaceable ways to interact with it, Hermes is the preferred intelligence and capability platform, and nodes make permitted device abilities available to it. **The Host is an active coordinator and a service role, not an app role.** It knows which paired nodes and Hermes-backed capabilities are currently eligible, selects a target under policy, obtains approval where required, dispatches work, observes it, and records an honest durable outcome.

V1 has one active headless Host process. The same service artifact may run on Linux/VPS, macOS, an old PC, or Android Termux. systemd, launchd, Docker, or Termux/runit supervises it; the executable itself stays a normal foreground process with explicit startup, readiness, shutdown, and exit behavior. A surface or companion may share the physical device but remains a separate client and principal.

## Current implementation status

The repository currently implements the Go Space authority, encrypted durable storage, owner-restricted local control, Host-local task execution and recovery, and a versioned Hermes Runs adapter. The public setup, doctor, and service experience is not yet complete. Canonical conversations and product memory, authenticated node transport, remote node execution, restricted file capabilities, web and messaging surfaces, and the Jarvis voice experience described below are target architecture, not shipped behavior. Phase-specific evidence is tracked in [PLAN.md](./PLAN.md); architecture must not be read as an implementation claim.

## Component responsibilities

### Space Host

The Host owns the Space identity, canonical conversations, accepted memory and shared context, node and integration registries, capability grants, durable task and automation state, routing, approval records, audit history, and portable encrypted state. It authenticates messages and actively selects only eligible execution targets. It does not need to proxy local model calls or same-device tool traffic.

The first production composition is a native Go Host service that combines the pure Go Space authority core with encrypted durable storage, a local operator control boundary, structured health, and a versioned Hermes Runtime Adapter. One native command contract runs on Linux/VPS, macOS, and Android Termux. Host correctness never depends on Hermes availability. Kotlin remains Android application code and Swift remains Apple application code; nodes share schemas and conformance fixtures rather than Host implementation internals.

The previously merged Android foreground service and Android-owned canonical store proved the portable boundary on a phone but assigned authority to the wrong process. They have been removed from the companion application and remain available only through Git history. [PHASE-2-HOST-HERMES.md](./PHASE-2-HOST-HERMES.md) records the replacement slice.

The pure Go core owns Space semantics, policy, task state, recovery, and context rules. The headless service owns Host composition, durable authority storage, lifecycle, and runtime adapters. Platform apps own only their node identity, local cache, connection, UI, capabilities, and OS permissions.

### Host process boundary

`lumen` is the owner-facing lifecycle and integration CLI. `lumen setup` plans and resumes installation, `lumen doctor` reports the whole deployment, and `lumen service` controls installed supervisors. `lumen-host` remains the internal foreground service binary and exposes low-level initialization, serving, task, approval, cancellation, and shutdown primitives. Operator commands reach the running daemon only through an owner-restricted Unix-domain socket with an independent credential and pinned Host endpoint identity. Secrets come from restricted files or service-manager credentials, never command arguments, prompts, or logs.

The setup layer owns platform detection, pinned artifact compatibility, Hermes acquisition or adoption, typed configuration generation, separate credentials, create-once Host initialization, supervisor installation, boot enablement, and redacted setup evidence. It records locally verifiable stages so interruption resumes without rotating identities, overwriting encrypted state, duplicating services, or downgrading the selected profile. Platform adapters perform launchd, systemd, Docker, and Termux/runit operations; they do not define separate product semantics.

Hardened Linux, macOS, and VPS deployments isolate Hermes under another OS principal or container and reach it through a protected endpoint or proxy using pinned mutual TLS plus a separately scoped bearer credential. Plain loopback HTTP is an explicit development profile with synthetic state and cannot satisfy the security exit gate. Hermes and Lumen are separately supervised processes; Lumen does not import Hermes internals, edit Hermes state, or treat Hermes health as Host health.

Hermes capability discovery populates an untrusted inventory, not the public capability registry. Telegram, WhatsApp, browser, MCP, skills, voice, delegation, automation, and remote execution become available only after a typed Lumen contract defines action and target scope, credential exposure, policy and approval, idempotency, cancellation, receipt, audit, and uncertain-outcome behavior. Lumen owns `integration list|connect|disconnect|status`; Hermes may supply provider mechanics but cannot authorize or persist canonical connection state.

Android Termux runs the same Host artifact and contract. Processes within one Termux installation share an Android UID, so a co-located Hermes process is compatibility-only and cannot protect Host secrets. A security-valid Termux deployment reaches Hermes isolated on another machine or container through pinned mutual TLS; compromise of the Termux UID remains compromise of the Host.

The initialized active Host identity is also the first execution target and advertises the Host-local Hermes capability. The recoverable owner identity remains a separate principal. Authentication through the operator socket maps a request to that owner only after the Host validates the independent operator credential and endpoint identity; it does not merge owner, Host, Hermes, or future node identities.

### Node runtime

Every node owns its platform integration, local task runner, capability adapters, local context cache, permission prompts, and persistent authenticated connection to the Host. It advertises typed abilities and health, but advertisement never creates authority. The Host may invoke only actions allowed by the active grant, and the target node revalidates that grant and its local OS permission immediately before execution.

Device access is never ambient or device-wide. Capability families may include `files.search`, `files.read`, `files.write`, `application.open`, `application.control`, `screen.observe`, `screen.interact`, `shell.execute`, `notification.deliver`, `microphone.capture`, `speaker.speak`, `camera.capture`, `model.infer`, `artifact.receive`, and `artifact.publish`. Each action is separately grantable. A file grant binds permitted operations to canonical roots, data classes, size limits, duration, and task or workflow scope; it must reject traversal, symlink escape, stale paths, and undeclared writes. Access to one directory never implies access to the device, account, terminal, browser profile, or adjacent directory.

A node may execute locally while the Host is unavailable only if an unexpired cached grant explicitly permits offline use. It records a local event stream and permitted context delta and synchronizes later. Cross-node execution always requires current Host authorization.

### Companion surfaces

Web, messaging, native applications, voice endpoints, the Android desk companion, and the Mac pet are interfaces to the same Space conversations, tasks, and authority. They provide text or voice interaction, presence, progress, and approval prompts; none owns a separate memory or Hermes session silo. Removing, force-stopping, or uninstalling a surface does not stop the Host, delete the Space, or change authority.

### Deployment fabric

Deployment topology and security assurance are independent choices:

- **Combined:** Host, encrypted Space store, Hermes, gateways, and workers share one machine or VPS as separately supervised services.
- **Separated:** Host and canonical storage remain together while Hermes or heavy workers run elsewhere through an authenticated, pinned transport.
- **Managed:** a shared provisioning and billing control plane creates a dedicated isolated data plane for each customer; the control plane receives only minimal operational metadata.
- **Hybrid:** authorized nodes perform selected model or capability work locally and synchronize only permitted outcomes.

Every topology must preserve the same Space, capability, task, migration, and failure semantics. Assurance is declared separately as **development**, **personal**, or **hardened**. Development permits synthetic-state compatibility shortcuts. Personal is suitable for owner-operated real data with documented residual risk. Hardened requires enforceable process or machine isolation, pinned mutual identity, independent credentials, tested backup and recovery, and the applicable threat-model gates. Separation alone does not make a deployment hardened, and co-location does not prevent hardening when principals or containers provide an enforceable boundary.

## Execution paths

### Host-local Hermes path

1. The local operator submits a typed task through the authenticated Host Unix-domain control socket.
2. The Host durably records the idempotency key and validates capability, policy, context scope, deadline, and approval state.
3. The Host checks Hermes `/v1/capabilities` and authenticated readiness, then creates a run through `/v1/runs`.
4. The adapter normalizes the Hermes SSE stream into bounded Lumen events. Unknown, oversized, reordered, or duplicated events fail closed or deduplicate under the recorded run mapping.
5. Hermes may report a runtime approval request, but only the Host may match and consume an exact Lumen approval before forwarding a decision.
6. The Host records completion, failure, cancellation, or `unknown_outcome` before reporting it to the operator. Hermes output is evidence, never the authority record.

### Local fast path

1. The user invokes Lumen on a node.
2. The node resolves a local capability and checks its cached policy.
3. The node talks directly to a local Hermes runtime or another explicitly approved local runtime adapter and executes under node-local policy.
4. It stores a local event stream and context delta.
5. It synchronizes the permitted delta with the Host asynchronously.

The Host is informed, not placed in the latency-critical path.

### Cross-node path

1. The origin node sends a signed, encrypted, expiring intent to the Host.
2. The Host persists it, evaluates policy, and resolves eligible target capabilities.
3. If selection or authority is ambiguous, the Host asks the user instead of guessing.
4. The Host dispatches an idempotent command to the selected node.
5. The target executes, requests action-bound permission when needed, and streams normalized events.
6. The Host updates canonical task state and shares authorized results with subscribed surfaces.

The origin surface does not need to name a device. The Host resolves a target from capability version, grant, resource scope, health, locality, latency, data policy, and user preference. If more than one materially different target remains, it asks instead of guessing. The target's receipt is evidence; the Host commits canonical task state only after applying its own transition rules.

Cross-node work requires the Host. When it is unavailable, the origin reports that fact and may queue only within an explicit delivery policy.

The first transport implementation is local-network: mDNS discovers a previously paired Host, then nodes use one mutually authenticated encrypted channel. Discovery is never trust. The secure-node milestone extends the same signed envelope and authority contract through outbound persistent sessions and a content-blind relay for managed and NAT-bound nodes; the relay has no Space keys or authority and cannot assert completion. Push wake-up remains capability-driven later work.

## Capability contract

Each capability manifest declares:

- stable ID and schema version;
- supported actions and typed arguments;
- node constraints and health;
- required OS permissions and secrets;
- context inputs and possible outputs;
- risk classification and approval points;
- whether local, remote, scheduled, and offline execution are supported.

A grant selects capability actions and scopes, then applies `deny`, `ask`, or `allow`. Optional restrictions include resources, paths, recipients, time windows, data classification, frequency, cost, and expiry. Default is deny. An adapter cannot expand its own grant.

State-changing capability contracts also declare authority, validation point, idempotency key, durable acceptance record, retry and timeout behavior, cancellation lineage, receipt semantics, and the result used when completion cannot be proven. File, screen, microphone, camera, shell, browser, account, and write capabilities remain separate even when one node implements several of them.

## Conversations, memory, and shared context

All surfaces map into Host-owned `Conversation`, `Message`, `Participant`, and `Surface` records. Hermes runtime sessions are replaceable execution mappings, not conversation authority. A runtime outage, upgrade, or replacement cannot erase canonical history.

Runtimes may propose memory, but only the Host validates and persists canonical records. Low-risk memory may be accepted automatically under an inspectable retention policy; sensitive or consequential durable memory requires confirmation. Records carry scope, provenance, classification, retention, expiry, confidence, and a digest; users can inspect why a record exists and edit, export, or delete it within their authorized scope. Deletion excludes it from future context projections and follows the documented durable deletion and backup policy.

The Host stores canonical Space context as typed records and append-only events, not one unbounded prompt. Context namespaces include user preferences, projects, devices, tasks, schedules, and capability-specific memory. Each synchronized record carries origin node, version, timestamp, classification, retention, and content digest.

Users choose a synchronization level per capability: `none`, `metadata`, `summary`, or `content`. Local context remains usable when disconnected. Conflicting mutable records are preserved as conflicts or resolved by a type-specific rule; arrival order alone never silently wins.

The target durable entities are `Space`, `Identity`, `Conversation`, `Message`, `Participant`, `Surface`, `Node`, `Integration`, `CapabilityManifest`, `Grant`, `Task`, `TaskEvent`, `Approval`, `Automation`, `Schedule`, `ContextRecord`, `MemoryProposal`, `RetentionPolicy`, `Artifact`, and `Receipt`. Entities not present in the current Go model remain planned contracts.

## Task lifecycle

`received → validating → queued → dispatched → running → awaiting_permission → synchronizing → completed`

Any active state may move to `cancelling`, `failed`, `paused`, `expired`, or `unknown_outcome`. Local tasks begin at `running` and later enter `synchronizing`. Transitions are compare-and-set and idempotent; uncertainty is never reported as success.

The Go Host persists encrypted canonical state, create intent, exact task-to-runtime mapping, approvals, bounded terminal output, and reconciliation evidence before acknowledging the corresponding transition. On restart it resumes only durably mapped runs, never treats a historical receipt as redispatch authority, and records `unknown_outcome` when bounded reconciliation cannot prove completion. Cross-node reconciliation remains deferred to Block 3; the Block 2 Host-local contract is documented in [Phase 2](./PHASE-2-HOST-HERMES.md).

## Hermes intelligence boundary

Hermes is Lumen's deeply integrated native intelligence and capability platform. Lumen reuses rather than reimplements Hermes's agent loop, model and provider routing, context compression, built-in tools and toolsets, skills, MCP clients, browser and computer use, messaging gateways, voice orchestration, vision and image generation, schedules and loops, delegated agents, remote execution, plugins, and observability. Each surface still enters the Space through a versioned Lumen contract and the [documented Hermes API](https://hermes-agent.nousresearch.com/docs/developer-guide/programmatic-integration); Lumen may upgrade or replace Hermes without migrating Space authority.

Capability work follows one mandatory order:

1. Adopt the existing Hermes subsystem when it meets the contract.
2. Configure or extend it through supported Hermes tools, skills, plugins, providers, gateways, or profiles.
3. Integrate a mature external open-source component only when reproducible quality, latency, privacy, resource, maintenance, licensing, and operational benchmarks demonstrate a meaningful gap.
4. Build custom functionality only for Space-specific authority and continuity, a native OS boundary, or a requirement no suitable upstream component fills.

Every adopted component has pinned provenance and version, license and commercial-hosting review, data-access declaration, compatibility evidence, fallback behavior, staged rollout, rollback, and a capability kill switch. Discovery and installation never grant authority.

Hermes receives only the task, target, capability-scoped context, model/provider constraints, deadline, and cancellation identity that Host or node policy approved. It may propose plans, tools, model choices, child runs, remote targets, memory, speech, events, and artifacts. Only Lumen validates authority, resolves the execution node, brokers credentials, consumes approval, persists context, and commits final task state.

The Host imports Hermes capability metadata into a deny-by-default registry. Discovery is descriptive, never a grant. MCP servers, skills, and tools are pinned and reviewed executable dependencies. Each run uses an immutable, verifiable runtime profile or isolated worker containing only its approved tool allowlist, model/provider configuration, remote backend, and scoped credential handles. A Hermes remote-execution backend is deployment configuration for that worker, not a node transport or authorization shortcut.

Every delegated Hermes run maps to one Host-owned parent task. Hermes currently inherits a parent's enabled toolsets and provider credentials when delegating, so Lumen enables delegation only inside a worker whose entire transitive tool, credential, model, remote-backend, context, budget, deadline, and cancellation surface has been granted to the parent task. If that aggregate grant is too broad or the worker profile cannot be verified, delegation is disabled. Children cannot approve themselves or create untracked durable tasks. Model routing is constrained by the immutable profile and Lumen policy for data classification, allowed provider and location, cost, latency, and capability requirements.

## Jarvis voice and presence

The Jarvis inspiration is a system behavior, not merely speech input and output: Lumen is reachable from the nearest authorized surface, continues one relationship and conversation, understands only permitted current context, coordinates real actions, and makes its restraint visible.

Every voice surface implements the same observable state model:

```text
idle → wake-detected → listening → understanding → thinking
                                                ├─→ approval-needed
                                                └─→ acting → speaking

Any active state may move to interrupted, offline, or degraded.
```

A button, text command, shortcut, or node-local wake word activates a conversation. Wake-word and voice-activity detection before activation run entirely on the node without model calls; Hermes client capture and remote wake detection are prohibited. No audio, transcript, activation metadata, model call, or tool call leaves the node before activation. Listening is visibly and audibly understandable, capture is short-lived by default, microphone permission is capability- and OS-gated, and the user always has immediate stop and barge-in control.

After activation, Lumen prefers Hermes voice orchestration and selects local or cloud speech-to-text, text-to-speech, interruption, and diarization components only under the upstream qualification rule and data policy. Wake detection remains local. Voice handoff changes the active surface without creating another conversation or memory. Proactive suggestions are allowed within policy; background actions require explicit, narrow, inspectable, and revocable automation grants.

`browser.run` is a capability adapter, not a general browser attached to the runtime. The first actions are read-only research, navigation, and extraction on allowlisted public sites. A browser profile is a protected credential boundary: the Host stores only an opaque profile reference, never cookies or passwords as context. Draft and submit actions require an exact preview, one-time approval, durable receipt, and an honest unknown outcome when completion cannot be verified.

For `coding.run`, Hermes works in an isolated Git worktree or sandbox, never the canonical project. Lumen validates the produced patch and applies it only under the capability’s approval policy. A Hermes command approval is a runtime safeguard, not a Lumen capability grant.

Platform features use narrow adapters. For example, Apple Reminders may be implemented with native APIs/App Intents or Hermes’s reviewed `apple-reminders` skill on a Mac. The adapter exposes only the declared reminder actions; it does not receive general shell or account authority.

## Security and recovery rules

- Device keys are hardware-backed where the platform permits; secrets never enter prompts or ordinary logs.
- Messages include Space, active Host epoch, sender, recipient, task, schema, nonce, issue time, expiry, and signature.
- Approvals bind to exact action arguments or artifact digest and can be consumed once.
- Nodes and adapters receive minimum context; remote transport carries end-to-end encrypted envelopes.
- Host export is encrypted and requires explicit owner action. Host migration prevents two active Hosts from accepting new work.
- A Host restart restores durable tasks and reconciles target-node status before retrying.
- A revoked node loses future synchronization and execution authority immediately after the Host records revocation.
- The target node revalidates capability action, resource scope, expiry, Host epoch, and current OS permission before every execution; Host authorization cannot override a local denial.
- Backup, restore, and Host migration are owner-initiated. A monotonically advancing Host epoch rejects stale Hosts and prevents automatic failover from creating a second authority.

## Product and interoperability boundary

Lumen's application, managed service, product experience, Space Authority implementation, and commercial distribution are proprietary paid products. Interoperability must not depend on access to those internals. The node pairing and transport protocol, capability manifests and invocation contracts, runtime event contract, conversation ingress and delivery adapter contract, context retrieval and memory-proposal contract, provider/model constraint contract, public SDK, conformance fixtures, and compatibility/version policy are public specifications.

Managed and paid self-hosted deployments use the same protocols and encrypted Space export format. A public protocol enables independent nodes and integrations; it does not allow them to bypass entitlement, pairing, grants, approval, local permission, or Host authority.

## Target code boundaries

`internal/space/` contains pure Go Space semantics and must not import UI, service managers, HTTP clients, filesystem storage, or Hermes packages. `cmd/lumen-host/` owns the process entry point; `internal/host/` composes the authority core with `internal/control/`, `internal/store/`, and `internal/hermes/`. `protocol/` owns versioned schemas and cross-language conformance fixtures. Platform apps remain clients and expose OS-specific permissions. The optional relay transports opaque envelopes and owns no Space authority.
