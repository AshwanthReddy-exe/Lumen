# Lumen design principles

This document is the engineering lens for Lumen. It complements the product behavior in [PRD](./PRD.md), the system boundaries in [ARCHITECTURE](./ARCHITECTURE.md), and the durable choices in [DECISIONS](./DECISIONS.md).

## 1. Design around the Space

Lumen is a private, multi-device Space. A device is only a node; a companion is only an interface. The active Host owns canonical state, policy, routing, and audit history. No UI, model, Hermes process, relay, or node may become an accidental second authority.

Start each feature with its user journey, authority, contract, lifecycle, failure behavior, and observable success. Keep the first implementation a small vertical slice with a real boundary and a useful end-to-end result.

## 2. Keep the dependency direction honest

Dependencies point inward:

```text
platform app / UI
        ↓
platform adapters, transport, runtime, storage
        ↓
protocol contracts
        ↓
portable Space domain
```

The portable domain owns policy, task transitions, context rules, and invariants. It must not import UI, Android/iOS APIs, Hermes, a transport, a database, or a vendor SDK. Adapters implement narrow, versioned contracts and cannot add authority. Do not create a generic abstraction until two real callers need the same stable behavior.

## 3. Make authority and trust explicit

- The Host is the authority for cross-node work, grants, canonical context, task state, approvals, and audit records.
- Local execution may avoid the Host latency path, but never bypass local capability policy.
- Cross-node execution requires Host authorization and an eligible, paired target.
- Discovery identifies a possible peer; authentication establishes a peer; authorization permits one action. These are separate checks.
- Hermes, models, tools, MCP servers, relays, nodes, and external content are untrusted inputs. Their responses are evidence, never authority.

## 4. Treat capabilities as the security boundary

Every action has a stable capability ID, schema version, typed arguments, declared context, required permissions, risk class, and execution modes. Grants default to deny and may allow or ask only for declared actions and scopes. Approvals bind to the exact action and arguments (or artifact digest), are single-use, and expire. Credentials are scoped handles, not prompt text or ordinary log content.

## 5. Design state changes before code

For every state-changing operation, document:

1. **Authority:** which Host, node, or local policy may accept it.
2. **Validation:** the invariant, schema, grant, epoch, and freshness checks.
3. **Identity:** the idempotency key and stable correlation/task ID.
4. **Durability:** the record written before acknowledging acceptance.
5. **Retry:** what may be retried and how duplicates are absorbed.
6. **Timeout/cancellation:** who can cancel, when it stops, and what remains recorded.
7. **Uncertainty:** the honest result when a dependency or connection fails.

Never report success from a model response, adapter callback, or log line. Acceptance means the authoritative record says so.

## 6. Assume failure and disorder

Process death, restart, duplicated or reordered messages, stale grants, clock changes, offline nodes, partial writes, unavailable Hermes, network loss, and dependency upgrades are normal states. Use explicit lifecycle states, compare-and-set transitions, monotonic Host epochs, expiry checks, durable events, and reconciliation. Unknown outcome is a first-class result; retry only when the operation is idempotent and policy permits it.

## 7. Minimize context and protect user control

Collect the least data needed for the action and retain it for the shortest useful period. Context is typed records with classification, retention, origin, version, and digest—not an unbounded prompt. Synchronization is capability-specific and defaults to the least sharing level that preserves continuity. Redact secrets and unnecessary personal content from events, diagnostics, exports, and error messages. Deletion and revocation must have durable records and clear scope.

## 8. Make operations observable without making them invasive

Emit structured, redacted events for authorization, dispatch, lifecycle transitions, synchronization, recovery, and failure. Health signals must distinguish unavailable, degraded, stale, denied, and unknown outcomes and provide an actionable next step. Logs support diagnosis only; audit state remains in the Host’s durable record. Metrics and traces must not become a covert context channel.

## 9. Prefer reversible change and tested contracts

Public schemas, persistence formats, and adapter contracts are versioned. Migrations are explicit, reversible where practical, and tested against old and new data. Configuration is explicit and safe by default. Compatibility fixtures cover valid and rejected messages. Production readiness requires acceptance, negative, recovery, and operational checks on the target platform—not only a happy-path unit test.

## Code organization rule

Prefer small, cohesive files with one responsibility: domain policy, a state transition, a protocol type, an adapter, or a platform integration. Keep orchestration thin and name the boundary it crosses. Split a file when it mixes authority, I/O, and policy or when its tests require unrelated setup. Avoid framework-shaped domain objects, speculative layers, and utility buckets; the code structure should reveal who owns each decision.

## Review gate

A change is ready when a reviewer can answer, from the code and tests: who owns the decision, what is trusted, what is persisted, what happens twice or out of order, what happens after restart or timeout, what data crosses each boundary, and how the user learns the outcome. If any answer is unclear, the design is not finished.
