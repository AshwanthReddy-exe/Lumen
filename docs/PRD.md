# Lumen V1 Product Requirements

## Vision

Lumen makes a person’s devices feel like one understandable, permission-controlled computing Space. The user can begin from any available surface and use capabilities provided by any authorized node without learning networking, agents, or infrastructure.

## Product principles

1. **The Space is the product.** Individual apps, pets, hosts, and runtimes are replaceable surfaces or components.
2. **One active Host coordinates V1.** It keeps canonical shared context and routes cross-node tasks.
3. **Local work stays local.** A node can use its own models and capabilities without a Host round trip, then synchronize permitted context.
4. **Capabilities, not devices, grant authority.** Every feature is separately configurable as deny, ask, or allow with scope.
5. **Autonomy is bounded.** Lumen proceeds within granted authority and asks for help or permission when required.
6. **Companions are optional.** A desk phone or Mac pet makes Lumen approachable but is never required.
7. **Invocation is reactive by default.** No model or tool work begins until the owner invokes Lumen or explicitly enables a bounded schedule or event rule.

## First reference deployment

The initial working Space starts with a headless Host service and a separately supervised Hermes runtime. A co-located plaintext loopback profile may prove compatibility only with synthetic state; the release proof isolates Hermes behind pinned mutual TLS. The Host runs from a terminal under an external service manager on a development computer, Linux/VPS, or Android Termux. The old Android phone is a companion and capability node; it is never the Host merely because the companion app is installed. Mac and iPhone nodes follow only after the Host-to-Hermes execution loop is complete.

## Core journeys

1. Initialize a Space through the local Host CLI, start the Host as a supervised service, and verify durable recovery.
2. Configure an isolated, mutually authenticated Hermes endpoint, submit one bounded task through the Host, and follow its normalized events to an honest terminal outcome. A plaintext loopback run is development evidence only.
3. Invoke Lumen on an iPhone, route an authorized coding or application task to a Mac, and receive progress and the verified result on the iPhone.
4. Invoke Lumen on the Mac, route an authorized reminder or notification action to the iPhone, and preserve one Host-owned task history across both devices.
5. Run a latency-sensitive Mac session through a local Hermes/model fast path under cached node policy, then synchronize the permitted outcome with the Host.
6. Pair the Android desk companion and use its face, display, microphone, speaker, and camera only through separately granted capabilities; the companion never owns canonical authority.
7. Inspect each node’s available capabilities and configure deny, ask, allow, target, resource, context, cost, and expiry scopes.
8. Receive an approval request when an action exceeds its grant, then approve once, reject, narrow the scope, or cancel.
9. Revoke a node and verify that it can no longer read context or execute work.

## Requirements

### Space and Host

| ID | Requirement |
| --- | --- |
| FR-01 | Create one private Space with one active Host and a recoverable owner identity. |
| FR-02 | Pair and revoke nodes using device-generated keys and explicit confirmation. |
| FR-03 | Let any eligible deployment environment run the documented Host service and be selected through an explicit migration flow; co-located node and Host identities remain separate. |
| FR-04 | Keep canonical shared context, policies, task history, and node registry on the active Host. |
| FR-05 | Authenticate signed, expiring messages and reject repeated nonces or revoked senders. |
| FR-06 | Run the Host as a headless foreground process whose lifecycle is supervised externally and whose executable contract is identical on Linux/VPS, macOS, and Android Termux. |
| FR-07 | Keep companion applications outside the Host authority boundary even when a companion and Host share one physical device. |

### Nodes and capabilities

| ID | Requirement |
| --- | --- |
| FR-10 | A node advertises versioned capabilities, health, constraints, and whether it can execute locally. |
| FR-11 | Configure each capability as `deny`, `ask`, or `allow`, with optional action, resource, time, and data scopes. |
| FR-12 | Validate capability, scope, user authority, expiry, node health, and required approval before execution. |
| FR-13 | Expose capability availability honestly; unsupported or offline work is queued, failed, or unavailable, never completed. |
| FR-14 | Support many feature adapters without giving an adapter access to unrelated capabilities or context. |

### Local and cross-node work

| ID | Requirement |
| --- | --- |
| FR-20 | Execute same-node work locally without routing model calls or tool traffic through the Host. |
| FR-21 | Synchronize the task outcome and user-permitted context delta with the Host when reachable. |
| FR-22 | Route cross-node work through the Host using idempotent commands and durable task state. |
| FR-23 | Allow local work during Host unavailability within cached grants; clearly show that shared context is not yet synchronized. |
| FR-24 | Reconcile duplicates, conflicts, and delayed events without silently overwriting newer context. |

### Tasks, approval, and context

| ID | Requirement |
| --- | --- |
| FR-30 | Persist an honest task lifecycle across restarts, disconnects, cancellation, and unknown outcomes. |
| FR-31 | Bind one-time approval to the exact task, action, arguments or artifact digest, actor, node, and expiry. |
| FR-32 | Support immediate and scheduled tasks; a schedule states target capability, authority, delivery behavior, and expiry. |
| FR-33 | Let users choose context synchronization per capability: none, metadata, summary, or content where supported. |
| FR-34 | Encrypt sensitive Host state and cross-node content, with export, backup, restore, and deletion controls. |
| FR-35 | Record redacted audit events for routing, policy, approval, execution, synchronization, and outcome. |
| FR-36 | Let a runtime propose typed memory records, but require Host validation and user inspection before a record becomes canonical context. |
| FR-37 | Keep browser profiles and their credentials outside shared context; browser actions require a capability grant and action-specific policy. |
| FR-38 | Integrate Hermes through a versioned adapter that discovers capabilities, submits and observes runs, forwards only exact approved runtime decisions, supports cancellation, and treats every Hermes result as untrusted evidence. |
| FR-39 | Use approved Hermes features—including model routing, built-in tools, skills, MCP, browser, voice, delegation, and remote execution—through immutable per-run profiles and Lumen capabilities without duplicating those subsystems. Delegation is disabled unless the parent grant covers its full transitive runtime surface. |
| FR-40 | Perform no background model or tool work by default; pre-activation wake-word and voice-activity detection run only on the node with zero outbound audio or model traffic, while schedules and event triggers require explicit, inspectable grants. |

## Initial capability set

The first contracts should prove different behaviors rather than maximize feature count. Hermes supplies intelligence and tool implementations where it already has them; Lumen supplies the stable capability, permission, task, and audit contracts:

- `coding.run`: local and remote Mac coding through a Hermes adapter, with reviewed changes.
- `reminder.manage`: create, list, complete, and delete reminders on an eligible Apple node.
- `schedule.manage`: create, pause, resume, and cancel Host-owned schedules.
- `notification.deliver`: deliver task and approval notifications to selected nodes.
- `browser.run`: initially research, navigate, and extract from allowlisted public sites; later draft and submit only with an exact preview and one-time approval.

## V1 success gate

A private alpha is ready when the reference Space completes at least 20 real tasks over 14 days; local Mac work succeeds without a Host round trip; cross-node commands route through the Host; deny/ask/allow rules behave correctly; revocation and replay tests pass; restarts recover an honest state; and no node receives context or authority outside its grant.

## Non-goals

V1 does not promise every possible device action, arbitrary screen control, multiple active Hosts, multi-user Spaces, a public plugin marketplace, billing, or automatic Host failover. New capabilities must use the same manifest, permission, task, context, and audit contracts.
