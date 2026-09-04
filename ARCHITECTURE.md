# Lumen Architecture

## System model

```text
                         ┌────────── Lumen Space ──────────┐
                         │                                 │
Old Android phone        │  Active Host                    │
desk companion ─────────▶│  context · policy · tasks       │
                         │  registry · scheduler · audit    │
                         │          │                      │
                         │     cross-node routing           │
                         │       ┌──┴──────────┐            │
                         │       ▼             ▼            │
                         │   Mac node      iPhone node      │
                         │   pet + code     app + personal  │
                         └─────────────────────────────────┘
```

**Deployment role and device type are independent.** A phone, Mac, Linux machine, old PC, home server, or VPS may become Host if it implements the Host contract. V1 has one active Host. A node may simultaneously be an interaction surface, execution node, companion, and Host.

## Component responsibilities

### Space Host

The Host owns the Space identity, node registry, capability grants, canonical shared context, durable cross-node task state, scheduler, routing, approval records, and audit history. It authenticates messages and selects only eligible nodes. It does not need to proxy local model calls or same-device tool traffic.

The first Android Host should be a lightweight native Lumen service with a foreground-service lifecycle, local encrypted storage, health reporting, and restart recovery. Hermes on Termux may be offered as an optional Host-local execution adapter, but Host correctness cannot depend on it.

The portable core is Kotlin Multiplatform and owns protocol validation, Space semantics, policy, task state, and context rules. Android uses native Kotlin platform code and native UI; iOS and macOS use native Swift/SwiftUI surfaces and platform adapters. UI, device keys, encrypted storage, lifecycle, and OS permissions remain platform-owned rather than crossing the portable boundary.

### Node runtime

Every node owns its platform integration, local task runner, capability adapters, local context cache, permission prompts, and connection to the Host. A node may execute locally while the Host is unavailable if it has an unexpired cached grant. It records a context delta and synchronizes later.

### Companion surfaces

The Android desk companion and Mac pet are UI shells over the same Space and node contracts. They provide voice or text interaction, presence, progress, and approval prompts. Removing or closing a companion does not delete the Space or change authority.

## Execution paths

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

The Host stores canonical Space context as typed records and append-only events, not one unbounded prompt. Context namespaces include user preferences, projects, devices, tasks, schedules, and capability-specific memory. Each synchronized record carries origin node, version, timestamp, classification, retention, and content digest.

Users choose a synchronization level per capability: `none`, `metadata`, `summary`, or `content`. Local context remains usable when disconnected. Conflicting mutable records are preserved as conflicts or resolved by a type-specific rule; arrival order alone never silently wins.

The minimum durable entities are `Space`, `Identity`, `Node`, `CapabilityManifest`, `Grant`, `Task`, `TaskEvent`, `Approval`, `Schedule`, `ContextRecord`, and `Artifact`.

## Task lifecycle

`received → validating → queued → dispatched → running → awaiting_permission → synchronizing → completed`

Any active state may move to `cancelling`, `failed`, `paused`, `expired`, or `unknown_outcome`. Local tasks begin at `running` and later enter `synchronizing`. Transitions are compare-and-set and idempotent; uncertainty is never reported as success.

## Runtime and platform adapters

Hermes is one execution adapter, principally for capable desktop/server nodes. Integrate through its documented Runs API for start, status, SSE events, stop, approval, health, capability discovery, and idempotency. Lumen normalizes Hermes events and keeps independent task and context state.

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

`core/` contains portable Space semantics and must not import UI or Hermes packages. `packages/protocol/` owns versioned wire schemas. `adapters/` implement capabilities behind contracts. Platform apps compose these modules and expose OS-specific permissions. The optional relay transports opaque envelopes and owns no Space authority.
