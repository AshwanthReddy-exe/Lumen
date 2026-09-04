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

## First reference deployment

The initial working Space has one old Android phone acting as Host and desk companion, one Mac acting as a pet surface and coding node, and one iPhone acting as an interaction and personal-capability node. This topology validates the model; it does not define the product’s platform boundary.

## Core journeys

1. Create a Space on the old phone and pair the Mac and iPhone.
2. Inspect each node’s available capabilities and configure their authority.
3. Ask the desk companion to execute a coding task on the Mac and follow its progress.
4. Talk to the Mac pet and perform a Mac-local task directly; synchronize the useful task context to the Host afterward.
5. Ask from one node to create a reminder or scheduled task using an eligible node.
6. Receive an approval request when an action exceeds its grant, then approve once, reject, narrow the scope, or cancel.
7. Revoke a node and verify that it can no longer read context or execute work.

## Requirements

### Space and Host

| ID | Requirement |
| --- | --- |
| FR-01 | Create one private Space with one active Host and a recoverable owner identity. |
| FR-02 | Pair and revoke nodes using device-generated keys and explicit confirmation. |
| FR-03 | Let any eligible node satisfy the documented Host contract and be selected through an explicit migration flow. |
| FR-04 | Keep canonical shared context, policies, task history, and node registry on the active Host. |
| FR-05 | Authenticate signed, expiring messages and reject repeated nonces or revoked senders. |

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

## Initial capability set

The first contracts should prove different behaviors rather than maximize feature count:

- `coding.run`: local and remote Mac coding through a Hermes adapter, with reviewed changes.
- `reminder.manage`: create, list, complete, and delete reminders on an eligible Apple node.
- `schedule.manage`: create, pause, resume, and cancel Host-owned schedules.
- `notification.deliver`: deliver task and approval notifications to selected nodes.

## V1 success gate

A private alpha is ready when the reference Space completes at least 20 real tasks over 14 days; local Mac work succeeds without a Host round trip; cross-node commands route through the Host; deny/ask/allow rules behave correctly; revocation and replay tests pass; restarts recover an honest state; and no node receives context or authority outside its grant.

## Non-goals

V1 does not promise every possible device action, arbitrary screen control, multiple active Hosts, multi-user Spaces, a public plugin marketplace, billing, or automatic Host failover. New capabilities must use the same manifest, permission, task, context, and audit contracts.
