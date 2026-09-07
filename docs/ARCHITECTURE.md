# Lumen Architecture

## System model

```text
Operator CLI ──local authenticated control──▶ Headless Host service
                                                │
                         canonical Space state  │ versioned Runtime Adapter
                         policy · tasks · audit │
                                                ▼
                                         Hermes API server

Later authenticated node transport:
Android companion · Mac node · iPhone node ──▶ Headless Host service
```

**The Host is a service role, not an app role.** V1 has one active headless Host process. The same service artifact may run on Linux/VPS, macOS, an old PC, or Android Termux. systemd, launchd, Docker, or Termux/runit supervises it; the executable itself stays a normal foreground process with explicit startup, readiness, shutdown, and exit behavior. A companion can run on the same device but remains a separate client.

## Component responsibilities

### Space Host

The Host owns the Space identity, node registry, capability grants, canonical shared context, durable cross-node task state, scheduler, routing, approval records, and audit history. It authenticates messages and selects only eligible nodes. It does not need to proxy local model calls or same-device tool traffic.

The first production composition is a Kotlin/JVM Host service that combines the portable `SpaceHost` with encrypted durable storage, a local operator control boundary, structured health, and a versioned Hermes Runtime Adapter. JVM 21 is the deployment baseline so one distribution can run on Linux/VPS, macOS, and Android Termux. Host correctness never depends on Hermes availability.

The previously merged Android foreground service and Android-owned canonical store proved the portable boundary on a phone but assigned authority to the wrong process. They are superseded and must be removed from the companion application. See [PHASE-2-ANDROID-HOST.md](./PHASE-2-ANDROID-HOST.md) for the historical evidence and [PHASE-2-HOST-HERMES.md](./PHASE-2-HOST-HERMES.md) for the replacement slice.

The portable core is Kotlin Multiplatform and owns Space semantics, policy, task state, recovery, and context rules. The headless service owns Host composition, durable authority storage, lifecycle, and runtime adapters. Platform apps own only their node identity, local cache, connection, UI, capabilities, and OS permissions.

### Host process boundary

`lumen-host` exposes commands for initialization, serving, status, task submission, cancellation, approval resolution, and clean shutdown. Operator commands reach the running daemon only through an owner-restricted Unix-domain socket with an independent credential and pinned Host endpoint identity. Secrets come from restricted files or service-manager credentials, never command arguments, prompts, or logs. The Host refuses all work until its encrypted state is open and restart recovery is committed. It may report the Space ready while Hermes is degraded, but it refuses new Hermes-targeted work until that adapter reports compatible readiness.

Hardened Linux, macOS, and VPS deployments isolate Hermes under another OS principal or container and reach it through a protected endpoint or proxy using pinned mutual TLS plus a separately scoped bearer credential. Plain loopback HTTP is an explicit development profile with synthetic state and cannot satisfy the security exit gate. Hermes and Lumen are separately supervised processes; Lumen does not import Hermes internals, edit Hermes state, or treat Hermes health as Host health.

Android Termux runs the same Host artifact and contract. Processes within one Termux installation share an Android UID, so a co-located Hermes process is compatibility-only and cannot protect Host secrets. A security-valid Termux deployment reaches Hermes isolated on another machine or container through pinned mutual TLS; compromise of the Termux UID remains compromise of the Host.

The initialized active Host identity is also the first execution target and advertises the Host-local Hermes capability. The recoverable owner identity remains a separate principal. Authentication through the operator socket maps a request to that owner only after the Host validates the independent operator credential and endpoint identity; it does not merge owner, Host, Hermes, or future node identities.

### Node runtime

Every node owns its platform integration, local task runner, capability adapters, local context cache, permission prompts, and connection to the Host. A node may execute locally while the Host is unavailable if it has an unexpired cached grant. It records a context delta and synchronizes later.

### Companion surfaces

The Android desk companion and Mac pet are UI shells over the same node contracts. They provide voice or text interaction, presence, progress, and approval prompts. Removing, force-stopping, or uninstalling a companion does not stop the Host, delete the Space, or change authority.

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
3. The node talks directly to its configured model or adapter and executes locally.
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

Cross-node work requires the Host. When it is unavailable, the origin reports that fact and may queue only within an explicit delivery policy.

The first transport is local-network only: mDNS discovers a previously paired Host, then nodes use one mutually authenticated encrypted channel. Discovery is never trust; a relay, internet traversal, and push wake-up remain later work.

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

## Shared context

Runtimes may propose memory, but only the Host validates and persists canonical records. Records carry scope, provenance, classification, retention, expiry, confidence, and a digest; users can inspect, edit, export, and delete them within their authorized scope.

The Host stores canonical Space context as typed records and append-only events, not one unbounded prompt. Context namespaces include user preferences, projects, devices, tasks, schedules, and capability-specific memory. Each synchronized record carries origin node, version, timestamp, classification, retention, and content digest.

Users choose a synchronization level per capability: `none`, `metadata`, `summary`, or `content`. Local context remains usable when disconnected. Conflicting mutable records are preserved as conflicts or resolved by a type-specific rule; arrival order alone never silently wins.

The minimum durable entities are `Space`, `Identity`, `Node`, `CapabilityManifest`, `Grant`, `Task`, `TaskEvent`, `Approval`, `Schedule`, `ContextRecord`, and `Artifact`.

## Task lifecycle

`received → validating → queued → dispatched → running → awaiting_permission → synchronizing → completed`

Any active state may move to `cancelling`, `failed`, `paused`, `expired`, or `unknown_outcome`. Local tasks begin at `running` and later enter `synchronizing`. Transitions are compare-and-set and idempotent; uncertainty is never reported as success.

The implemented in-memory subset uses conservative [restart recovery](./PHASE-1-CONTRACT.md#restart-recovery-increment): queued tasks become unknown because there is no durable dispatch record yet. Historical command receipts never authorize redispatch. Storage and startup must commit recovery before accepting work; encrypted storage and target reconciliation remain unimplemented.

## Runtime and platform adapters

Hermes is the first execution adapter used to prove the Host before node networking begins. Integrate through its [documented Runs API](https://hermes-agent.nousresearch.com/docs/developer-guide/programmatic-integration) for start, status, SSE events, stop, approval, health, capability discovery, steering where policy permits, and idempotency. Lumen normalizes Hermes events and keeps independent task and context state.

Hermes receives only the task, capability-scoped context, deadline, and cancellation identity that a Host policy approved. Its output is evidence: events, proposed actions, and artifacts never become authority records by themselves.

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
- Backup, restore, and Host migration are owner-initiated. A monotonically advancing Host epoch rejects stale Hosts and prevents automatic failover from creating a second authority.

## Target code boundaries

`core/` contains portable Space semantics and must not import UI, service managers, HTTP clients, or Hermes packages. `services/host/` composes the Host process and owns its operator boundary and durable adapters. `adapters/hermes/` implements only the versioned Hermes contract. `packages/protocol/` is created when node transport begins. Platform apps remain clients and expose OS-specific permissions. The optional relay transports opaque envelopes and owns no Space authority.
