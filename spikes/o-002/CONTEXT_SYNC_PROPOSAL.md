# O-002 context synchronization proposal

## Status

This contract resolves `O-002` as `D-021`. It freezes the V1 defaults; implementation and physical behavior are verified in the relevant delivery phases.

## Design goals

- Preserve useful continuity without turning the Host into a copy of every conversation, file, or notification.
- Make the user-selected level enforceable before serialization and again before Host persistence.
- Separate operational records needed to run the Space from user context supplied to a capability.
- Preserve origin, classification, retention, and deletion behavior through offline sync and conflict handling.

## Boundary and vocabulary

**Operational records** are the minimum redacted records required for routing, policy, approvals, task lifecycle, audit, and schedule ownership. They are not context and are retained under their own policies. A task result may record `completed`, `failed`, or `unknown_outcome` without retaining its private text or artifact content.

**Context** is user or capability information retained for later continuity: preferences, project notes, reminders, task summaries, and capability-specific memory. Every context record has a stable record ID, namespace, schema version, origin node, logical version, classification, retention deadline, digest, and payload only when the selected level permits it.

The Host is the canonical record owner after accepting a delta. A node retains only its local cache and journal. No capability may read a context namespace unless its declared input contract and current grant permit it.

## Synchronization levels

| Level | What leaves the node | Host behavior | What it must never include |
| --- | --- | --- | --- |
| `none` | No context record or free-text result. | Retain only the separate operational record. | User content, prompts, source code, reminder text, attachments, or model output. |
| `metadata` | Record header, namespace, classification, timestamps, retention, origin, content digest, and coarse size/type indicators. | Index the header for inspection and synchronization status; no payload is persisted. | Any recoverable content or preview. |
| `summary` | Metadata plus a bounded, explicitly labelled summary payload produced under the capability's data policy. | Persist and expose the summary only to authorized readers. | Credentials, tokens, private keys, raw prompts, raw source, attachments, or text excluded by the record classification. |
| `content` | Metadata plus the declared typed payload. | Encrypt at rest and expose it only through the declared namespace and grant. | Secrets or data whose classification forbids synchronization. |

The selected level is a maximum, not a request to fill every field. A classification rule can further reduce it; it can never raise it. Unknown levels, record types, classifications, or schema fields fail closed.

## Proposed V1 defaults

| Capability | Default | Rationale | Explicit opt-in can raise to |
| --- | --- | --- | --- |
| `coding.run` | `metadata` | Preserve task provenance and project identity without copying source, prompts, patches, or command output into the Space. | `summary`, then scoped `content` for a named project and retention period. |
| `reminder.manage` | `metadata` | Reminder titles, notes, and recipients can be sensitive. The Host retains operational state needed to route the action, not reminder text. | `content` only for the named reminder namespace. |
| `schedule.manage` | `metadata` for capability context | The Host necessarily stores the typed schedule definition as an operational, Host-owned entity; it is not general context. | `summary` for explanatory history; content only for a declared schedule note namespace. |
| `notification.deliver` | `none` | Delivery receipts and redacted status are operational records. Notification payloads are not durable context by default. | `metadata` only for a named notification history namespace. |

No initial capability defaults to `content`. An approval to perform an action is not approval to retain its content; raising a context level requires a separate, visible per-capability choice that names namespace, scope, retention, and affected nodes.

## Examples

### Local coding task

A Mac executes `coding.run` locally with the default `metadata` setting. The Host receives the task's operational lifecycle and a context header such as project namespace, origin, digest, classification, and expiry. It does not receive a prompt, diff, source file, shell output, or model transcript. If the user selects `summary`, the Mac may upload a bounded result such as “updated tests for pairing validation,” subject to the project policy.

### Reminder created from another node

An Android companion asks an iPhone node to create a reminder. The Host routes and records the approved action. Under the default `metadata` setting, reminder title and notes remain on the eligible Apple node. The owner can opt in to a specific reminder namespace at `content` when cross-device recall is desired.

### Private content

Secrets, authentication material, recovery material, payment data, raw private prompts, and content marked `local-only` have an effective level of `none`, regardless of a broader capability setting. The UI must explain the reduction rather than silently synchronize less data.

### Conflicting offline edits

Two disconnected nodes update the same mutable context record. The Host stores both causally distinct versions and marks a conflict; it does not choose based on arrival order. Immutable records deduplicate by stable ID and digest. A type-specific resolver may be added only with an accepted decision and tests.

### Deletion and retention

An owner deletion creates a signed, versioned tombstone that propagates to authorized node caches and blocks later uploads of an older version. The Host retains only a redacted audit fact that deletion occurred, not the deleted payload. Retention expiry follows the same tombstone path. An offline node must reconcile with the tombstone before it can resynchronize the deleted record.

## Required implementation evidence

1. Golden fixtures prove each selected level at both the outbound envelope and Host-store boundary.
2. Negative fixtures prove payload leakage, unknown fields, disallowed classifications, stale versions, and tombstoned uploads are rejected.
3. A local/offline scenario proves no context is uploaded before the selected policy permits it.
4. Owner review confirms the four defaults, the operational-record boundary, retention defaults, and the conflict experience.
