# Phase 1 Space core

## Scope

Phase 1 proves portable Space rules with fake nodes and a durable Host boundary. It does not implement device keys, encrypted serialization, transport, UI, adapters, or platform lifecycle. Those are native boundaries beginning in Phase 2.

## Frozen core contract

The portable `core:space` module exposes immutable `SpaceState` plus operations that return either a new state and value or a stable rejection reason.

- `createSpace(spaceId, ownerNodeId, hostNodeId)` creates epoch `1` and pairs the owner and Host.
- `pairNode` accepts a unique node; a revoked or unknown node cannot originate or receive work. Revocation cancels queued work and pending approvals for that node and rejects later submit, approval, and completion attempts.
- `advertiseCapability` records a capability/action pair for a paired node.
- `setGrant` records `deny`, `ask`, or `allow` for one target capability action.
- `submit` validates the Space, active Host epoch, origin, target, advertisement, action, and grant. Missing advertisement or grant is `deny`. It records one task by idempotency key and returns `rejected`, `awaiting_permission`, or `queued`.
- `approve` carries task, actor, target, Host epoch, action fingerprint, and expiry. It binds to one awaiting task; duplicate, expired, stale-epoch, or altered approvals are rejected.
- `complete` carries the current Host epoch and is callable only by the selected target for a queued task. It records `completed`, `failed`, or `unknown_outcome`. `unknown_outcome` is terminal until a future explicit reconciliation operation proves a replacement outcome.

Every submit, approval, and completion carries the current Host epoch and rejects a stale epoch before state changes. Every state-changing operation records a redacted audit event containing actor, active Host authority, epoch, operation, outcome, and idempotency key—never task arguments or content. Revocation also records each task it invalidates. A command fingerprint canonically binds Space, epoch, origin, target, capability, action, and argument or artifact digest. Repeating its idempotency key with the same fingerprint returns the first recorded result; reusing it with a different fingerprint is rejected.

`SpaceHost` is the serial portable Host boundary. `SpaceStateStore` creates a Space only when empty and serializes each transition with its atomic replacement. The Host returns an accepted, rejected, or replayed transition only after that write; a failed write returns `persistence_unavailable` while retaining the last committed state. On open, it first reads the committed state, applies restart recovery, and writes that recovery result before becoming ready. The platform adapter owns encryption, key access, schema migration, and storage errors. This preserves the KMP boundary in [ARCHITECTURE](./ARCHITECTURE.md#space-host).

## Observable success

`tools:space-scenario` creates fake Android, Mac, and iPhone nodes; grants one Mac capability; submits a task; demonstrates `deny`, `ask`, and `allow`; and exits nonzero if a policy or lifecycle invariant fails. Core tests cover policy, idempotency, revocation, exact approval, terminal outcomes, write failure, reopened state, and conservative recovery.

## Recovery behavior

`recoverAfterRestart` is a pure transition over restored state. It accepts a Space ID, current Host epoch, active Host actor, and operation ID. Only the paired active Host may invoke it; being the owner alone is insufficient. Wrong Space, stale epoch, and unauthorized actors are rejected. This operation is internal to Host startup, not a node-callable transport endpoint.

The current slice has no dispatch record, so every `queued` task may have executed before interruption. Recovery marks those tasks `unknown_outcome` with reason `HOST_RESTARTED`, preserves awaiting permissions and terminal tasks, and records one redacted event per changed task plus the operation receipt. It preserves grants, consumed approvals, membership, and the Host epoch. A restart is not Host migration.

The operation ID identifies one startup attempt. Retrying that exact operation returns its first receipt without touching newly submitted work; changed content under the same ID is rejected. A later startup uses a new ID. Submit receipts remain historical: replaying a submit may return its original `queued` receipt while the task is now `unknown_outcome`. Callers must read authoritative task state and must never dispatch based on a replayed receipt. Late completion cannot replace an unknown outcome through `complete`.

On write failure, startup remains unavailable and retries from the latest committed state; it must not dispatch work. No retry or target reconciliation is implemented here: unknown outcomes remain terminal pending an explicit evidence-based reconciliation contract.

Validation: `mise run phase1-check` covers mixed lifecycle recovery, Host authorization, Space/epoch mismatches, duplicate startup, key collisions, historical submit replay, late completion, failed writes, reopened state, and a simulated restart in the fake-node scenario.

## Phase boundary

This phase has no claim of encrypted serialization, real process lifecycle, authenticated transport, or target reconciliation. The Phase 2 Android adapter must implement the store contract with hardware-backed key access where available, atomic encrypted persistence, migrations, and physical restart evidence.
