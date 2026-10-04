# Requirements: Lumen

## Scope

These are the 58 published FR requirements in [docs/PRD.md](../docs/PRD.md), retained verbatim as primary IDs. The owner column names the delivery contract owner, not a person. Each ID has exactly one destination phase in the fixed [master plan](../docs/PLAN.md#dependency-aware-roadmap). The requirements span personal alpha (Phases 01–10) and later releases (Phases 11–17); later phases are planned, not alpha prerequisites.

## Traceability

| Requirement | Description | Owner | Phase | Status |
|---|---|---|---|---|
| FR-01 | Create one private Space with one active Host and a recoverable owner identity. | Host/Hermes | Phase 02 | Complete |
| FR-02 | Pair and revoke nodes using device-generated keys and explicit confirmation. | Host identity | Phase 05 | Pending |
| FR-03 | Let any eligible deployment environment run the Host contract and migrate explicitly; co-located Host and node identities remain separate. | Host operations | Phase 10 | Pending |
| FR-04 | Keep canonical shared context, policies, task history, and node registry on the active Host. | Host | Phase 03 | Pending |
| FR-05 | Authenticate signed, expiring messages and reject repeated nonces or revoked senders. | Host security | Phase 05 | Pending |
| FR-06 | Run the Host as a headless foreground process supervised externally under one portable executable contract. | Host/Hermes | Phase 02 | Complete |
| FR-07 | Keep companion applications outside Host authority even when they share a physical device. | Android companion | Phase 08 | Pending |
| FR-08 | Provide resumable `lumen setup` that installs or adopts compatible Lumen and Hermes artifacts, generates configuration, and initializes Host state exactly once. | Host/Hermes | Phase 10 | Pending |
| FR-09 | Supervise Host and Hermes at boot where supported and report exactly `ready`, `degraded`, or `action_required` without silent downgrade. | Host/Hermes | Phase 10 | Pending |
| FR-10 | Let a node advertise versioned capabilities, health, constraints, and local execution support. | Node | Phase 05 | Pending |
| FR-11 | Configure each capability as `deny`, `ask`, or `allow`, with action, resource, time, and data scopes. | Host and Mac node | Phase 07 | Pending |
| FR-12 | Validate capability, scope, user authority, expiry, node health, and approval before execution. | Host and Mac node | Phase 07 | Pending |
| FR-13 | Report unsupported or offline work as queued, failed, or unavailable, never completed. | Host and Mac node | Phase 07 | Pending |
| FR-14 | Support many adapters without giving one access to unrelated capabilities or context. | Host and Mac node | Phase 07 | Pending |
| FR-15 | Pair nodes through QR or short code with explicit confirmation and device-management controls. | Client identity | Phase 05 | Pending |
| FR-20 | Execute same-node work locally when policy permits without routing model or tool traffic through the Host. | Node | Phase 07 | Pending |
| FR-21 | Synchronize only the user-permitted task outcome and context delta with the Host. | Node | Phase 07 | Pending |
| FR-22 | Route cross-node work through the Host using idempotent commands and durable task state. | Host and Mac node | Phase 07 | Pending |
| FR-23 | Allow local work during Host unavailability only within unexpired cached grants and show unsynchronized state. | Node | Phase 05 | Pending |
| FR-24 | Reconcile duplicates, conflicts, and delayed events without silently overwriting newer context. | Host synchronization | Phase 05 | Pending |
| FR-30 | Persist an honest task lifecycle across restart, disconnect, cancellation, and unknown outcomes. | Host | Phase 03 | Pending |
| FR-31 | Bind one-time approval to the exact task, action, arguments or artifact digest, actor, node, and expiry. | Host policy | Phase 06 | Pending |
| FR-32 | Support immediate and scheduled tasks with explicit target, authority, delivery, and expiry. | Host scheduler | Phase 14 | Pending |
| FR-33 | Let users select `none`, `metadata`, `summary`, or `content` synchronization per capability where supported. | Host | Phase 04 | Pending |
| FR-34 | Encrypt sensitive Host state and cross-node content with export, backup, restore, and deletion controls. | Host operations | Phase 10 | Pending |
| FR-35 | Record redacted audit events for routing, policy, approval, execution, synchronization, and outcome. | Host audit | Phase 07 | Pending |
| FR-36 | Let a runtime propose typed memory while only the Host validates and persists canonical context. | Host | Phase 04 | Pending |
| FR-37 | Keep browser profiles and credentials outside shared context and gate browser actions by capability policy. | Browser adapter | Phase 12 | Pending |
| FR-38 | Integrate Hermes through a versioned adapter with discovery, runs, events, approvals, cancellation, and untrusted-result handling. | Host/Hermes | Phase 02 | Complete |
| FR-39 | Reuse approved Hermes model routing, tools, skills, MCP, browser, voice, delegation, and remote execution through immutable scoped profiles. | Runtime integration | Phase 14 | Pending |
| FR-40 | Perform no background model or tool work by default; keep pre-activation wake-word and VAD local with zero outbound audio, while schedules and triggers require explicit inspectable grants. | Android companion | Phase 08 | Pending |
| FR-41 | Expose Host-local Hermes reasoning first as `agent.run/execute` without granting transitive authority. | Host/Hermes | Phase 02 | Complete |
| FR-42 | Expose Hermes-backed integrations only through Lumen lifecycle, policy, credential, audit, cancellation, receipt, and uncertainty contracts. | Runtime integration | Phase 12 | Pending |
| FR-60 | Represent canonical conversations independently from Hermes sessions and map web, messaging, voice, native, and ambient surfaces into them. | Host | Phase 03 | Pending |
| FR-61 | Give accepted memory provenance, classification, retention, scope, inspection, correction, deletion, and export; sensitive or consequential memory requires confirmation. | Host | Phase 04 | Pending |
| FR-62 | Preserve identity, conversations, memory, and task continuity across Host restart, Hermes failure or replacement, surface disconnect, restore, and migration. | Host operations | Phase 10 | Pending |
| FR-63 | Project only task-relevant, policy-permitted context to a runtime, model, integration, or node. | Host | Phase 04 | Pending |
| FR-64 | Make the Host select, dispatch, observe, cancel, and reconcile node work while the target node revalidates the exact invocation and current local permission. | Host routing | Phase 07 | Pending |
| FR-65 | Begin file access with separate `files.search` and `files.read` grants over user-selected roots and reject traversal, symlink escape, stale paths, oversized results, and implicit writes. | Mac node | Phase 07 | Pending |
| FR-66 | Provide a Devices and abilities view for pairing, health, grants, activity, synchronization, local stop, and revocation. | Client UX | Phase 05 | Pending |
| FR-67 | Graduate autonomy only through suggestion, preview, one-time approval, workflow approval, and a narrow inspectable revocable grant. | Host scheduler | Phase 14 | Pending |
| FR-68 | Bind delegated children to the parent's tools, credentials, context, targets, provider policy, budget, deadline, and cancellation lineage. | Host scheduler | Phase 14 | Pending |
| FR-69 | Prefer Hermes for intelligence subsystems; use another mature component or custom Lumen code only after supported extension fails and reproducible benchmarks justify it. | Runtime integration | Phase 02 | Complete |
| FR-70 | Map messaging identities and threads into canonical Space conversations and deliver through typed capabilities with deduplication and honest receipts. | Messaging adapter | Phase 11 | Pending |
| FR-71 | Expose consistent presence states for idle, wake detection, listening, understanding, thinking, acting, approval, speaking, interruption, offline, and degraded operation. | Client UX | Phase 08 | Pending |
| FR-72 | Provide post-activation streaming recognition, interruptible speech, barge-in, immediate stop, and unmistakable listening state. | Android companion | Phase 08 | Pending |
| FR-73 | Route models and speech engines only within approved provider, region, retention, locality, latency, cost, and data-class constraints. | Host policy | Phase 08 | Pending |
| FR-74 | Support combined, separated, managed, and hybrid topologies under one Space contract, independently of development, personal, and hardened assurance. | Distribution | Phase 15 | Pending |
| FR-75 | Keep canonical encrypted storage with the Space Authority and limit Hermes and workers to bounded operational state. | Host | Phase 03 | Pending |
| FR-76 | Provision a dedicated isolated data plane for each managed customer and minimize shared control-plane metadata. | Managed operations | Phase 16 | Pending |
| FR-77 | Offer managed and self-hosted Lumen as paid proprietary products with interoperable encrypted exports and equivalent semantics. | Distribution | Phase 15 | Pending |
| FR-78 | Publish versioned node, capability, runtime, and integration contracts, SDKs, conformance fixtures, and compatibility rules without publishing proprietary product code. | Protocol ecosystem | Phase 17 | Pending |
| FR-79 | Qualify every upstream component for provenance, commercial rights, security, maintenance, data flow, benchmarks, fallback, rollback, and disablement. | Distribution | Phase 15 | Pending |
| FR-80 | Use progressive disclosure so ordinary users avoid infrastructure complexity while power users and self-hosters can inspect advanced controls. | Distribution | Phase 15 | Pending |
| FR-81 | Persist Hermes clarification questions separately from action approvals; answering a question supplies input but grants no new authority. | Host | Phase 06 | Pending |
| FR-82 | Show approvals and clarifications in a synchronized attention inbox and their conversation, with one accepted resolution, expiry, and consistent results across surfaces. | Host | Phase 06 | Pending |
| FR-83 | Keep wake detection, recognition, and speech synthesis local by default; any remote speech processing requires explicit opt-in and must never be an automatic fallback. | Android companion | Phase 08 | Pending |
| FR-84 | Qualify shared mobile UI and native audio/security modules on physical Android and iPhone devices before committing to that client strategy; provide the documented native-Android plus phone-web fallback if it fails. | Mobile client | Phase 05 | Pending |

## Validation boundary

Published PRD rows do not provide individual acceptance tests. Phase success criteria in [ROADMAP.md](ROADMAP.md) and the gates in [docs/PLAN.md](../docs/PLAN.md) supply observable checks; they do not imply implementation. No requirement is marked complete from PR #20 draft work.
