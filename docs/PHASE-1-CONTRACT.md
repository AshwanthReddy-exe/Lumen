# Phase 1 executable slice

## Scope

The first executable slice proves the Space rules with in-memory state and fake nodes. It creates a Space, pairs nodes, advertises a capability, evaluates a grant, records a task, consumes an approval once, and returns an honest terminal result. It does not implement encryption, real keys, durable storage, transport, UI, adapters, or platform lifecycle. It starts Phase 1; the encrypted-store and restart-reconciliation requirements remain mandatory before the Phase 1 exit gate.

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

## Observable success

`tools:space-scenario` creates fake Android, Mac, and iPhone nodes; grants one Mac capability; submits a task; demonstrates `deny`, `ask`, and `allow`; and exits nonzero if a policy or lifecycle invariant fails. Core tests cover invalid origin/target, revoked nodes, missing advertisements, stale epoch, duplicate commands and collisions, mismatched or reused approvals, and invalid task transitions.

## Deferred boundaries

The slice has no claim of encrypted persistence or authenticated transport. Phase 1 expands those behind the same contract; Android, Apple, Hermes, and real local-network work start only in their designated phases.
