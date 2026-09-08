# Decisions

## Accepted

| ID | Decision | Consequence |
| --- | --- | --- |
| D-001 | Build Lumen around a personal multi-device Space. | Platforms and companions are implementations, not the product definition. |
| D-002 | Keep one active Host as the canonical V1 coordinator. | Cross-node routing and shared state have one authority. |
| D-003 | Use scoped capabilities, not device roles, for authority. | Each feature supports deny, ask, or allow with restrictions. |
| D-006 | Put Hermes behind a versioned Runtime Adapter. | Hermes is the preferred intelligence runtime but remains replaceable; its absence may make intelligent work unavailable but never compromises or owns Lumen state. |
| D-007 | Encrypt sensitive state and support export, restore, and deletion. | Host storage includes explicit key and recovery behavior. |
| D-008 | Never put raw secrets in prompts or ordinary logs. | Adapters receive scoped credential handles. |
| D-009 | Route deterministically using capability, policy, and health. | Ambiguity is shown to the user instead of guessed. |
| D-011 | Give Hermes only reviewed tools and context. | Profiles help configuration but are not treated as sandboxes. |
| D-012 | Isolate `coding.run` from the canonical project. | Lumen validates and permission-checks generated patches before apply. |
| D-013 | Use Hermes’s documented Runs API. | Capability discovery and contract tests protect upgrades. |
| D-014 | Require Git workspaces for the initial coding capability. | Patch identity and stale-base checks remain deterministic. |
| D-015 | Allow any eligible device environment to run the Host contract. | Physical placement is configurable; the Host service identity and state remain distinct from co-located companion/node processes. |
| D-016 | Use a local fast path for same-device work. | Model and tool traffic stay local; context synchronizes afterward. |
| D-017 | Make companion interfaces optional. | Desk companion and Mac pet share core contracts with ordinary apps. |
| D-019 | Keep the Host core independent of Hermes. | Android/Termux limitations cannot break Space coordination. |
| D-020 | Use a Kotlin Multiplatform core with native Kotlin/Android and Swift/SwiftUI applications. | Portable rules are shared; UI, lifecycle, storage, keys, and permissions remain native. [Evidence](./O-001-EVIDENCE.md) |
| D-021 | Default initial capability context to the least useful level that preserves continuity. | `coding.run` and `reminder.manage` use metadata, `schedule.manage` separates Host-owned schedules from context, and `notification.deliver` uses none. [Contract](./O-002-CONTEXT.md) |
| D-022 | Start with paired-node, local-network transport only. | mDNS is discovery only; a mutually authenticated encrypted channel carries versioned envelopes. Remote relay, push, and NAT traversal remain later work. [Contract](./O-003-TRANSPORT.md) |
| D-024 | Use manual encrypted export and explicit Host migration with a monotonic epoch. | There is one active Host; no automatic failover, cloud escrow, or multi-master synchronization. [Contract](./O-005-RECOVERY.md) |
| D-025 | Freeze four small V1 capability contracts. | `coding.run`, `reminder.manage`, `schedule.manage`, and `notification.deliver` have typed actions and capability-scoped policy. [Contract](./O-006-CAPABILITIES.md) |
| D-026 | Use Hermes as Lumen's preferred intelligence runtime, but keep Lumen as the authority. | Lumen consumes Hermes model routing, tools, skills, MCP, browser, voice, delegation, and remote execution through scoped adapters; Hermes cannot grant authority, persist canonical state, select an unapproved target, or approve itself. [Hermes tools](https://hermes-agent.nousresearch.com/docs/user-guide/features/tools/) |
| D-027 | Add `browser.run` in two stages. | Read-only research begins with allowlisted public sites and isolated profiles. Draft and submit actions require an exact preview and one-time approval; credential, payment, upload, download, and 2FA actions remain out of the first action slice. [Browser profile guidance](https://docs.browser-use.com/open-source/customize/browser/authentication) |
| D-028 | Keep memory as Host-owned typed context. | Runtimes may propose records but cannot persist them directly. Each record is inspectable and has scope, provenance, classification, retention, expiry, and deletion behavior. |
| D-029 | Run the Host as a headless Kotlin/JVM service, supervised outside the process. | One foreground CLI artifact can run under systemd, launchd, Docker, or Termux/runit; companion apps never own canonical Space state. [Contract](./O-004-HOST.md) |
| D-030 | Complete the Host-to-Hermes execution loop before node networking or companion work. | The first live slice proves durable authority, runtime isolation, events, approval, cancellation, failure, and recovery through an authenticated Hermes Runs API. [Plan](./PHASE-2-HOST-HERMES.md) |
| D-031 | Require enforceable runtime isolation for hardened Host deployments. | Linux/macOS/VPS use separate principals or containers plus pinned endpoint identity. Android Termux runs the same Host contract, but same-UID Hermes is compatibility-only; security-valid use requires an isolated Hermes endpoint with pinned mutual TLS. |
| D-032 | Import Hermes capabilities instead of rebuilding parallel agent subsystems in Lumen. | Lumen defines stable authority and lifecycle contracts around immutable, verifiable runtime profiles. A Hermes feature remains disabled when its complete transitive tools, credentials, models, remote backend, and context cannot be bounded to the task. |
| D-033 | Make Lumen reactive by default. | Text, voice, button, or shortcut invocation starts work; pre-activation wake-word detection is node-local with zero outbound audio, while schedules and event triggers require explicit revocable grants. |

## Superseded

| ID | Former decision | Replaced by |
| --- | --- | --- |
| D-004 | Make iPhone the defining first interaction adapter. | `D-001` and `D-017`; every device is a possible surface. |
| D-005 | Defer Android. | `D-017` and `D-029`; Android is an early companion/node, while Host placement is independent. |
| D-010 | Predefine Home, VPS, and hybrid deployments. | `D-015`; any deployment must meet one Host contract. |
| D-018 | Use an old Android phone as the first Host. | `D-029`; the phone may run the headless Host under Termux, but the Android companion app is never the Host. |
| D-023 | Make the first Host a native Android foreground service. | `D-029` and `D-030`; the Host is a portable terminal service and Hermes integration precedes node work. |

## Phase 0 closure

`O-001`–`O-006` were initially resolved by `D-020`–`D-025`. `D-023` was later superseded by `D-029` after the Android prototype exposed the wrong Host boundary. The decision contracts state their reversal triggers and phase-specific acceptance checks. They do not claim that later node, Apple, migration, or remote-access code has already been implemented or physically tested.
