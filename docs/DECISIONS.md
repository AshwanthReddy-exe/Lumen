# Decisions

## Accepted

| ID | Decision | Consequence |
| --- | --- | --- |
| D-001 | Build Lumen around a personal multi-device Space. | Platforms and companions are implementations, not the product definition. |
| D-002 | Keep one active Host as the canonical V1 coordinator. | Cross-node routing and shared state have one authority. |
| D-003 | Use scoped capabilities, not device roles, for authority. | Each feature supports deny, ask, or allow with restrictions. |
| D-006 | Put Hermes behind a versioned Runtime Adapter. | Hermes is the native intelligence platform and remains replaceable at the boundary; its absence may make intelligent work unavailable but never compromises or owns Lumen state. |
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
| D-021 | Default initial capability context to the least useful level that preserves continuity. | `coding.run` and `reminder.manage` use metadata, `schedule.manage` separates Host-owned schedules from context, and `notification.deliver` uses none. [Contract](./O-002-CONTEXT.md) |
| D-022 | Use paired-node local transport as the first transport baseline. | mDNS is discovery only and a mutually authenticated encrypted channel carries versioned envelopes; `D-048` extends the same envelope and authority contract to remote connectivity. [Baseline](./O-003-TRANSPORT.md) |
| D-024 | Use manual encrypted export and explicit Host migration with a monotonic epoch. | There is one active Host; no automatic failover, cloud escrow, or multi-master synchronization. [Contract](./O-005-RECOVERY.md) |
| D-025 | Preserve four small Phase 0 capability contracts as the common capability baseline. | `coding.run`, `reminder.manage`, `schedule.manage`, and `notification.deliver` proved typed actions and scoped policy; they no longer define the complete launch catalog. [Historical contract](./O-006-CAPABILITIES.md) |
| D-026 | Use Hermes as Lumen's preferred intelligence runtime, but keep Lumen as the authority. | Lumen consumes Hermes model routing, tools, skills, MCP, browser, voice, delegation, and remote execution through scoped adapters; Hermes cannot grant authority, persist canonical state, select an unapproved target, or approve itself. [Hermes tools](https://hermes-agent.nousresearch.com/docs/user-guide/features/tools/) |
| D-027 | Add `browser.run` in two stages. | Read-only research begins with allowlisted public sites and isolated profiles. Draft and submit actions require an exact preview and one-time approval; credential, payment, upload, download, and 2FA actions remain out of the first action slice. [Browser profile guidance](https://docs.browser-use.com/open-source/customize/browser/authentication) |
| D-028 | Keep memory as Host-owned typed context. | Runtimes may propose records but cannot persist them directly. Each record is inspectable and has scope, provenance, classification, retention, expiry, and deletion behavior. |
| D-029 | Run the Host as a headless native service, supervised outside the process. | One foreground Go CLI artifact can run under systemd, launchd, Docker, or Termux/runit; companion apps never own canonical Space state. [Contract](./O-004-HOST.md) |
| D-030 | Complete the minimum reliable Host-to-Hermes foundation before the capability-first product slices. | The foundation proves durable authority, runtime isolation, events, approval, cancellation, failure, recovery, and setup without forcing all platform deployment evidence ahead of product validation. [Foundation gate](./PHASE-2-HOST-HERMES.md) |
| D-031 | Require enforceable runtime isolation for hardened Host deployments. | Linux/macOS/VPS use separate principals or containers plus pinned endpoint identity. Android Termux runs the same Host contract, but same-UID Hermes is compatibility-only; security-valid use requires an isolated Hermes endpoint with pinned mutual TLS. |
| D-032 | Import Hermes capabilities instead of rebuilding parallel agent subsystems in Lumen. | Lumen defines stable authority and lifecycle contracts around immutable, verifiable runtime profiles. A Hermes feature remains disabled when its complete transitive tools, credentials, models, remote backend, and context cannot be bounded to the task. |
| D-033 | Make Lumen locally present and suggestive by default, with earned autonomy. | Wake-word detection is node-local with zero outbound audio; suggestions have no side effects, and schedules, event triggers, and automatic actions require explicit narrow revocable grants. |
| D-034 | Implement the production Host and Space authority core in Go; keep mobile applications native. | The Host ships as a native macOS/Linux/Termux binary. Android stays Kotlin and Apple stays Swift; versioned protocols and conformance fixtures replace shared implementation code. [Design](./superpowers/specs/2026-09-08-go-host-rearchitecture-design.md) |
| D-035 | Advertise `agent.run/execute` as the first Host-local orchestration capability with an `ask` default. | The Host can invoke approved Hermes reasoning without receiving blanket authority; browser, shell, device, coding, and account effects remain separate capabilities. |
| D-036 | Make `lumen setup` the public, resumable installer and keep `lumen-host` as the internal foreground service. | Lumen owns platform detection, Lumen/Hermes acquisition and configuration, create-once initialization, supervisor installation, boot enablement, and whole-deployment readiness without exposing manual environment plumbing. |
| D-037 | Expose Hermes integrations only through Lumen-owned lifecycle and capability contracts. | Telegram, WhatsApp, browser, MCP, skills, voice, delegation, and remote execution reuse Hermes implementations, but discovery cannot self-enable them and Hermes cannot own credentials, grants, approvals, audit, or canonical connection state. |
| D-038 | Make Hermes-first reuse a product principle. | Adopt Hermes subsystems first, extend through supported Hermes surfaces second, integrate benchmark-winning upstream components for measured gaps third, and build only Space-specific or otherwise missing behavior. |
| D-039 | Make Jarvis-like continuity, presence, agency, and restraint the defining experience. | Web, messaging, voice, native, and ambient surfaces share one Space conversation; presence states are explicit and pre-activation sensing remains local. |
| D-040 | Make Host-to-node capability execution a first-class protocol. | Nodes advertise typed, restricted capabilities; the Host selects, authorizes, dispatches, observes, cancels, and records work while every node revalidates locally. |
| D-041 | Support combined, separated, managed, and hybrid deployment topologies under one Space contract. | Topology is independent of development, personal, or hardened assurance; canonical storage stays with the Space Authority in every topology. |
| D-042 | Ship Lumen as a proprietary paid product with public interoperability contracts. | Managed and self-hosted offerings are commercial; node protocols, capability schemas, SDKs, fixtures, and compatibility rules are public without opening the proprietary product implementation. |
| D-043 | Deliver capability-first vertical journeys rather than device-completion blocks. | The critical path is conversation and memory, node coordination and action, natural presence, messaging continuity, portability, and earned autonomy; devices are adapters and reference evidence. |
| D-044 | Use a dedicated data plane for each managed Space. | A shared control plane may provision, bill, and observe minimal operational health, but each customer receives isolated Host, storage, Hermes, credentials, backup, and deletion boundaries. |
| D-045 | Route models and media engines under data-aware policy. | Hermes may optimize quality, latency, and cost only within approved provider, region, retention, locality, and data-class constraints. |
| D-046 | Accept low-risk useful memory automatically but keep it inspectable. | Hermes may propose memory; the Host records provenance and retention, permits correction and deletion, and requires confirmation for sensitive or consequential records. |
| D-048 | Extend paired-node transport through outbound persistent sessions and a content-blind relay. | Managed and remote nodes can traverse NAT without making the relay a peer, authority, key holder, or source of completion truth; local-network transport remains the first implementation baseline. |

## Superseded

| ID | Former decision | Replaced by |
| --- | --- | --- |
| D-004 | Make iPhone the defining first interaction adapter. | `D-001` and `D-017`; every device is a possible surface. |
| D-005 | Defer Android. | `D-017` and `D-029`; Android is an early companion/node, while Host placement is independent. |
| D-010 | Predefine Home, VPS, and hybrid deployments. | `D-015`; any deployment must meet one Host contract. |
| D-018 | Use an old Android phone as the first Host. | `D-029`; the phone may run the headless Host under Termux, but the Android companion app is never the Host. |
| D-023 | Make the first Host a native Android foreground service. | `D-029` and `D-030`; the Host is a portable terminal service and Hermes integration precedes node work. |
| D-020 | Use a Kotlin Multiplatform core with native Kotlin/Android and Swift/SwiftUI applications. | `D-034`; only applications remain platform-native while the production Host and authority core move to Go. |

## Phase 0 closure

`O-001`–`O-006` were initially resolved by `D-020`–`D-025`. `D-023` was later superseded by `D-029` after the Android prototype exposed the wrong Host boundary. The decision contracts state their reversal triggers and phase-specific acceptance checks. They do not claim that later node, Apple, migration, or remote-access code has already been implemented or physically tested.
