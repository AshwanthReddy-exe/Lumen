# Lumen product requirements

## Vision

Lumen is a proprietary, paid personal AI Space inspired by Jarvis: one continuous, natural relationship that is available from any surface, remembers appropriately, understands the user's permitted context, and safely makes things happen across devices and services.

The defining experience is not voice alone. It combines presence, continuity, awareness, agency, and restraint. Lumen must always make it understandable when it is listening, thinking, acting, waiting for approval, offline, or degraded.

"The same agent on every device" means each supported surface continues the same Host-owned Space identity, conversation, accepted memory, task state, and approvals. A surface may disconnect or be replaced without creating a second agent identity or a separate authoritative history.

## Audience and product shape

Lumen serves normal users, power users, and self-hosters through progressive disclosure. A normal user can start with managed hosting and a conversational interface; a power user can inspect memory, permissions, models, tasks, and automation; a self-hoster can operate the same Space contract without receiving a different product.

The launch wedge is one conversation that continues across a responsive web application, one Hermes-backed messaging gateway, and one lightweight capability node; it includes inspectable memory and one safe, useful cross-node action.

## Product principles

1. **The Space is the product.** Devices, surfaces, models, runtimes, and companions are replaceable components.
2. **One active Host coordinates V1.** It owns canonical state and actively discovers, selects, authorizes, dispatches, observes, and reconciles node work.
3. **Capabilities, not devices, grant authority.** Discovery never grants access, and a node revalidates every invocation locally.
4. **Continuity is canonical.** Web, messaging, voice, and native surfaces share conversations, accepted memory, tasks, and identity rather than creating runtime silos.
5. **Autonomy is earned.** Lumen may suggest proactively, but execution grows from preview to one-time approval to narrow, inspectable, revocable grants.
6. **Privacy is structural.** Collect and transmit the minimum context required for one action; pre-activation audio processing stays local with zero outbound audio.
7. **Hermes is the native intelligence platform, not the Space authority.** Adopt, configure, or extend its full surface through Host-owned profiles and task capabilities before evaluating another mature component or building custom infrastructure.
8. **Portability is a product right.** Managed and paid self-hosted deployments use the same protocols and encrypted export format.
9. **Failures are honest.** Unsupported, offline, timed-out, or uncertain work is never represented as completed.
10. **Journeys organize delivery.** Particular phones, computers, and servers are examples and evidence targets, not roadmap phases.

## Hermes integration boundary

Lumen will reuse Hermes's full native surface—agent loop, model and provider routing, sessions, profiles and personas, tools and toolsets, skills, MCP, browser and computer use, gateways, voice, automation, delegation, workers, remote execution, plugins, and observability—only when a Host-owned feature registry maps a certified feature to an immutable runtime profile and task capability. Discovery is descriptive; it never enables a feature or grants access. These are target integrations, not current availability. See the [Hermes integration research](./research/HERMES-LUMEN-INTEGRATION.md) for the capability matrix and qualification boundary.

The Host has two separate Hermes seams. `HermesRuntimeAdapter v1` is the inbound runtime adapter for one authenticated Hermes endpoint: it negotiates capabilities, submits Host-authorized Runs, consumes one bounded event stream, reconciles status, and treats output as evidence. A separate narrow outbound Lumen Hermes plugin/broker exposes only typed capability intents and redacted inventory; it cannot read or mutate canonical Space state, issue grants, select a node, approve itself, or use the owner control socket.

Each Host binds to one active Hermes runtime. A Hermes session maps to a Host-owned conversation or task and is replaceable; a profile is an immutable digest of approved tools, models/providers, context, budget, deadline, credential handles, and cancellation constraints. `ContextRecord` projections are filtered before serialization, and a runtime's `MemoryProposal` is untrusted until the Host accepts it under provenance, classification, retention, scope, and confirmation rules. For node work, Hermes may propose constraints, but the Host selects the eligible node and the node revalidates the invocation, grant, resource scope, expiry, Host epoch, and local permission.

Milestone 1 offers explicit `combined` and `external` setup choices. Combined owns the Lumen and Hermes artifacts and services in the Docker reference deployment; external adopts an existing independently managed Hermes endpoint and controls only the Host. Both use the same Host-owned task contract and report only `ready`, `degraded`, or `action_required`; topology and assurance remain independent. The [approved dual-topology design](./superpowers/specs/2026-09-11-milestone-1-dual-topology-design.md) defines the evidence required before this milestone can close.

## Core journeys

1. Create a Space, begin a conversation on the web, receive a streamed answer through Hermes, and continue after Host or runtime restart.
2. Inspect what Lumen remembers, why it was retained, where it may be used, and correct or delete it.
3. Pair a node, inspect its live capabilities, and grant access to one bounded resource such as a selected folder without exposing unrelated files or tools.
4. Ask naturally for work that requires another node; preview any consequential action, approve it once or use an existing narrow grant, observe progress, and receive a durable receipt or honest uncertain outcome.
5. Continue the same conversation through one messaging provider without creating a separate memory or authority silo.
6. Speak to Lumen from a nearby surface, interrupt it naturally, and move the conversation to another surface without losing context.
7. Configure a repeated workflow by progressing from suggestion to preview to an inspectable, pausable, narrow automatic grant.
8. Export and migrate the Space between managed and self-hosted deployments without losing identity, conversations, memories, grants, tasks, or audit integrity.

## Requirements

### Stable foundation requirements

The original identifiers remain stable. New product requirements use `FR-60` and above rather than reassigning published IDs.

| ID | Requirement |
| --- | --- |
| FR-01 | Create one private Space with one active Host and a recoverable owner identity. |
| FR-02 | Pair and revoke nodes using device-generated keys and explicit confirmation. |
| FR-03 | Let any eligible deployment environment run the Host contract and migrate explicitly; co-located Host and node identities remain separate. |
| FR-04 | Keep canonical shared context, policies, task history, and node registry on the active Host. |
| FR-05 | Authenticate signed, expiring messages and reject repeated nonces or revoked senders. |
| FR-06 | Run the Host as a headless foreground process supervised externally under one portable executable contract. |
| FR-07 | Keep companion applications outside Host authority even when they share a physical device. |
| FR-08 | Provide resumable `lumen setup` that installs or adopts compatible Lumen and Hermes artifacts, generates configuration, and initializes Host state exactly once. |
| FR-09 | Supervise Host and Hermes at boot where supported and report exactly `ready`, `degraded`, or `action_required` without silent downgrade. |
| FR-10 | Let a node advertise versioned capabilities, health, constraints, and local execution support. |
| FR-11 | Configure each capability as `deny`, `ask`, or `allow`, with action, resource, time, and data scopes. |
| FR-12 | Validate capability, scope, user authority, expiry, node health, and approval before execution. |
| FR-13 | Report unsupported or offline work as queued, failed, or unavailable, never completed. |
| FR-14 | Support many adapters without giving one access to unrelated capabilities or context. |
| FR-15 | Pair nodes through QR or short code with explicit confirmation and device-management controls. |
| FR-20 | Execute same-node work locally when policy permits without routing model or tool traffic through the Host. |
| FR-21 | Synchronize only the user-permitted task outcome and context delta with the Host. |
| FR-22 | Route cross-node work through the Host using idempotent commands and durable task state. |
| FR-23 | Allow local work during Host unavailability only within unexpired cached grants and show unsynchronized state. |
| FR-24 | Reconcile duplicates, conflicts, and delayed events without silently overwriting newer context. |
| FR-30 | Persist an honest task lifecycle across restart, disconnect, cancellation, and unknown outcomes. |
| FR-31 | Bind one-time approval to the exact task, action, arguments or artifact digest, actor, node, and expiry. |
| FR-32 | Support immediate and scheduled tasks with explicit target, authority, delivery, and expiry. |
| FR-33 | Let users select `none`, `metadata`, `summary`, or `content` synchronization per capability where supported. |
| FR-34 | Encrypt sensitive Host state and cross-node content with export, backup, restore, and deletion controls. |
| FR-35 | Record redacted audit events for routing, policy, approval, execution, synchronization, and outcome. |
| FR-36 | Let a runtime propose typed memory while only the Host validates and persists canonical context. |
| FR-37 | Keep browser profiles and credentials outside shared context and gate browser actions by capability policy. |
| FR-38 | Integrate Hermes through a versioned adapter with discovery, runs, events, approvals, cancellation, and untrusted-result handling. |
| FR-39 | Reuse approved Hermes model routing, tools, skills, MCP, browser, voice, delegation, and remote execution through immutable scoped profiles. |
| FR-40 | Perform no background model or tool work by default; keep pre-activation wake-word and VAD local with zero outbound audio, while schedules and triggers require explicit inspectable grants. |
| FR-41 | Expose Host-local Hermes reasoning first as `agent.run/execute` without granting transitive authority. |
| FR-42 | Expose Hermes-backed integrations only through Lumen lifecycle, policy, credential, audit, cancellation, receipt, and uncertainty contracts. |

### Product-extension requirements

| ID | Requirement |
| --- | --- |
| FR-60 | Represent canonical conversations independently from Hermes sessions and map web, messaging, voice, native, and ambient surfaces into them. |
| FR-61 | Give accepted memory provenance, classification, retention, scope, inspection, correction, deletion, and export; sensitive or consequential memory requires confirmation. |
| FR-62 | Preserve identity, conversations, memory, and task continuity across Host restart, Hermes failure or replacement, surface disconnect, restore, and migration. |
| FR-63 | Project only task-relevant, policy-permitted context to a runtime, model, integration, or node. |
| FR-64 | Make the Host select, dispatch, observe, cancel, and reconcile node work while the target node revalidates the exact invocation and current local permission. |
| FR-65 | Begin file access with separate `files.search` and `files.read` grants over user-selected roots and reject traversal, symlink escape, stale paths, oversized results, and implicit writes. |
| FR-66 | Provide a Devices and abilities view for pairing, health, grants, activity, synchronization, local stop, and revocation. |
| FR-67 | Graduate autonomy only through suggestion, preview, one-time approval, workflow approval, and a narrow inspectable revocable grant. |
| FR-68 | Bind delegated children to the parent's tools, credentials, context, targets, provider policy, budget, deadline, and cancellation lineage. |
| FR-69 | Prefer Hermes for intelligence subsystems; use another mature component or custom Lumen code only after supported extension fails and reproducible benchmarks justify it. |
| FR-70 | Map messaging identities and threads into canonical Space conversations and deliver through typed capabilities with deduplication and honest receipts. |
| FR-71 | Expose consistent presence states for idle, wake detection, listening, understanding, thinking, acting, approval, speaking, interruption, offline, and degraded operation. |
| FR-72 | Provide post-activation streaming recognition, interruptible speech, barge-in, immediate stop, and unmistakable listening state. |
| FR-73 | Route models and speech engines only within approved provider, region, retention, locality, latency, cost, and data-class constraints. |
| FR-74 | Support combined, separated, managed, and hybrid topologies under one Space contract, independently of development, personal, and hardened assurance. |
| FR-75 | Keep canonical encrypted storage with the Space Authority and limit Hermes and workers to bounded operational state. |
| FR-76 | Provision a dedicated isolated data plane for each managed customer and minimize shared control-plane metadata. |
| FR-77 | Offer managed and self-hosted Lumen as paid proprietary products with interoperable encrypted exports and equivalent semantics. |
| FR-78 | Publish versioned node, capability, runtime, and integration contracts, SDKs, conformance fixtures, and compatibility rules without publishing proprietary product code. |
| FR-79 | Qualify every upstream component for provenance, commercial rights, security, maintenance, data flow, benchmarks, fallback, rollback, and disablement. |
| FR-80 | Use progressive disclosure so ordinary users avoid infrastructure complexity while power users and self-hosters can inspect advanced controls. |

## Launch capability cohort

The first cohort proves the complete continuity-and-action journey rather than maximizing breadth:

- Canonical conversation and inspectable memory through the web.
- One Hermes messaging gateway mapped into the same conversation.
- A lightweight node exposing restricted `files.search` and `files.read` plus one harmless delivery or notification capability.
- Hermes-backed `agent.run/execute` for bounded reasoning, with separate authority for every transitive tool or external effect.
- Local wake word, post-activation speech, and cross-surface handoff once the secure node action path is proven.

Coding, reminders, browser actions, application control, camera, microphone, shell, smart-home integrations, and particular phone or desktop companions are later capability cohorts or platform examples. Each must reuse the same authority and lifecycle contracts.

## Launch acceptance

A launch candidate must demonstrate that a new user reaches a continuity-and-action moment within ten minutes; one conversation continues across web, messaging, and a capability node; memory is useful, inspectable, correctable, exportable, and deletable; every side effect has deterministic authority and an honest receipt; combined, separated, managed, and hybrid deployments pass the same applicable Space conformance suite; a Space migrates between managed and self-hosted environments; Hermes failure cannot corrupt canonical state; upstream upgrades can roll back without user-data loss; and sustained real-user use meets defined reliability, trust, recovery, and retention targets.

## Current implementation boundary

The repository has a substantive native Go Space authority, encrypted persistence, task and approval lifecycle, local operator boundary, constrained Hermes Runs adapter, setup planning and artifact/configuration components, and supervision definitions. Those foundations must be preserved.

Public setup and whole-deployment doctor paths exist for combined and external topologies, with the remaining release and real-runtime evidence in [PLAN.md](./PLAN.md). Canonical conversation records and a Host chat path exist on the active implementation branch, but production certification and restart continuity remain open. Memory UX, web and messaging surfaces, node pairing and transport, restricted file execution, Jarvis-like voice presence, managed hosting, migration, and the public SDK are not implemented. Requirements describe the target product and must not be read as claims of current availability.

## Non-goals for the first launch

The first launch does not promise every device action, unrestricted filesystem or screen control, multiple active Hosts, multi-user Spaces, automatic Host failover, an open-source Lumen core, or a public marketplace. New capabilities must use the same manifest, permission, context, task, receipt, audit, and revocation contracts.
