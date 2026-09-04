# Decisions

## Accepted

| ID | Decision | Consequence |
| --- | --- | --- |
| D-001 | Build Lumen around a personal multi-device Space. | Platforms and companions are implementations, not the product definition. |
| D-002 | Keep one active Host as the canonical V1 coordinator. | Cross-node routing and shared state have one authority. |
| D-003 | Use scoped capabilities, not device roles, for authority. | Each feature supports deny, ask, or allow with restrictions. |
| D-006 | Put Hermes behind a versioned Runtime Adapter. | Hermes is optional and never owns Lumen state. |
| D-007 | Encrypt sensitive state and support export, restore, and deletion. | Host storage includes explicit key and recovery behavior. |
| D-008 | Never put raw secrets in prompts or ordinary logs. | Adapters receive scoped credential handles. |
| D-009 | Route deterministically using capability, policy, and health. | Ambiguity is shown to the user instead of guessed. |
| D-011 | Give Hermes only reviewed tools and context. | Profiles help configuration but are not treated as sandboxes. |
| D-012 | Isolate `coding.run` from the canonical project. | Lumen validates and permission-checks generated patches before apply. |
| D-013 | Use Hermes’s documented Runs API. | Capability discovery and contract tests protect upgrades. |
| D-014 | Require Git workspaces for the initial coding capability. | Patch identity and stale-base checks remain deterministic. |
| D-015 | Allow any eligible node to implement the Host contract. | Host placement is configurable rather than tied to a product tier. |
| D-016 | Use a local fast path for same-device work. | Model and tool traffic stay local; context synchronizes afterward. |
| D-017 | Make companion interfaces optional. | Desk companion and Mac pet share core contracts with ordinary apps. |
| D-018 | Use an old Android phone as the first Host. | The initial topology proves low-cost, always-present personal hosting. |
| D-019 | Keep the Host core independent of Hermes. | Android/Termux limitations cannot break Space coordination. |
| D-020 | Use a Kotlin Multiplatform core with native Kotlin/Android and Swift/SwiftUI applications. | Portable rules are shared; UI, lifecycle, storage, keys, and permissions remain native. [Evidence](./spikes/o-001/EVIDENCE.md) |
| D-021 | Default initial capability context to the least useful level that preserves continuity. | `coding.run` and `reminder.manage` use metadata, `schedule.manage` separates Host-owned schedules from context, and `notification.deliver` uses none. [Contract](./spikes/o-002/CONTEXT_SYNC_PROPOSAL.md) |
| D-022 | Start with paired-node, local-network transport only. | mDNS is discovery only; a mutually authenticated encrypted channel carries versioned envelopes. Remote relay, push, and NAT traversal remain later work. [Contract](./spikes/o-003/TRANSPORT_DECISION.md) |
| D-023 | Make the first Host a native Android foreground service for a charged local-network phone. | Lumen reports degraded/offline states honestly and does not promise server-grade uptime. [Contract](./spikes/o-004/HOST_DECISION.md) |
| D-024 | Use manual encrypted export and explicit Host migration with a monotonic epoch. | There is one active Host; no automatic failover, cloud escrow, or multi-master synchronization. [Contract](./spikes/o-005/RECOVERY_DECISION.md) |
| D-025 | Freeze four small V1 capability contracts. | `coding.run`, `reminder.manage`, `schedule.manage`, and `notification.deliver` have typed actions and capability-scoped policy. [Contract](./spikes/o-006/CAPABILITY_DECISION.md) |

## Superseded

| ID | Former decision | Replaced by |
| --- | --- | --- |
| D-004 | Make iPhone the defining first interaction adapter. | `D-001`, `D-017`, and `D-018`; every device is a possible surface. |
| D-005 | Defer Android. | `D-018`; Android is now the first reference Host. |
| D-010 | Predefine Home, VPS, and hybrid deployments. | `D-015`; any deployment must meet one Host contract. |

## Phase 0 closure

`O-001`–`O-006` are resolved by `D-020`–`D-025`. The decision contracts state their reversal triggers and phase-specific acceptance checks. They do not claim that later Android, Apple, migration, or remote-access code has already been implemented or physically tested.
