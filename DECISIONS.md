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

## Superseded

| ID | Former decision | Replaced by |
| --- | --- | --- |
| D-004 | Make iPhone the defining first interaction adapter. | `D-001`, `D-017`, and `D-018`; every device is a possible surface. |
| D-005 | Defer Android. | `D-018`; Android is now the first reference Host. |
| D-010 | Predefine Home, VPS, and hybrid deployments. | `D-015`; any deployment must meet one Host contract. |

## Must decide before implementation

| ID | Question | Evidence required |
| --- | --- | --- |
| O-001 | Which portable core, Android, iOS, and macOS stack minimizes duplicated logic? | Small pairing, encrypted-store, background-service, and SSE prototypes. |
| O-002 | What exactly is shared context and what are the default sync levels? | Examples for coding, reminders, schedules, private content, conflicts, and deletion. |
| O-003 | How does local discovery work, and which remote relay or direct transport follows? | Reconnect, NAT, push, end-to-end encryption, latency, cost, and onboarding tests. |
| O-004 | What makes an old Android Host dependable enough? | Reboot, Doze, battery optimization, foreground-service, charging, thermal, and storage tests. |
| O-005 | How are Host backup and explicit migration performed without split brain? | Failed-migration, rollback, recovery-key, and stale-Host tests. |
| O-006 | Which exact actions comprise the first four capability contracts? | User journeys and permission boundaries for coding, reminders, schedules, and notifications. |

Do not decide an open question silently. Record the evidence, choice, consequence, and reversal trigger.

## Evidence in progress

| Decision | Evidence | Status |
| --- | --- | --- |
| `O-001` | [Headless stack comparison spike](./spikes/o-001/README.md) and [results](./spikes/o-001/EVIDENCE.md) | Contract parsing evidence passes for both candidates; platform lifecycle, storage, interop, packaging, and device checks remain pending. |
