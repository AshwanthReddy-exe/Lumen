# O-006 capability decision

## Status

Accepted for V1 contract freeze. These are the smallest actions that support the
reference journeys. The actions are versioned, typed, and evaluated by Lumen
policy before an adapter is called. An adapter cannot add actions, widen scopes,
or convert a runtime approval into a Lumen grant.

## Common contract

Every invocation has `space_id`, `task_id`, `capability_id`, `action`, typed
`arguments`, `origin_node_id`, `target_node_id`, an idempotency key, issue time,
expiry, and the current grant reference. The Host is authoritative for
cross-node and scheduled work; a node is authoritative only for its local
execution policy and result. The Host persists the command before dispatch and
persists the result or `unknown_outcome` before reporting completion.

Scopes are restrictions, never additional authority. A grant may restrict:

- `actions`: an explicit subset of the actions below;
- `resources`: named project/workspace, reminder list, schedule ID, or node ID;
- `data`: context synchronization level and allowed classification;
- `time`: validity window and, for schedules, the schedule expiry;
- `frequency`: maximum invocations per time window.

The effective grant is the intersection of capability, node, resource, data,
time, and frequency scopes. Missing or unknown fields, expired grants, revoked
nodes, unhealthy or ineligible targets, invalid arguments, and stale grant
references fail closed.

## Actions

### `coding.run`

`run({ repository, base_ref, task, allowed_paths, artifact_mode })`

- `repository` is a named Git workspace already registered on the target Mac.
- `base_ref` identifies the expected commit; `task` is bounded user input.
- `allowed_paths` is a non-empty allowlist relative to that workspace.
- `artifact_mode` is `patch` (the only V1 value).
- Scope is the registered repository and allowed path set; shell, network,
  credentials, and unrelated repositories are not implied.
- The target uses an isolated worktree. The result is a patch digest and
  redacted status, not an applied change.
- Approval is required before applying a patch to the registered workspace,
  and is bound to task, repository, base commit, allowed paths, and digest.

No separate `apply` capability ships in V1; applying a reviewed result is the
terminal step of this action and cannot be approved against a changed digest.

### `reminder.manage`

`create({ list_id, title, notes? , due_at? })`,
`list({ list_id })`, `complete({ reminder_id })`, and
`delete({ reminder_id })`.

- `list_id` must be an explicitly granted reminder list; IDs are opaque.
- `title` and `notes` are user content and remain on the Apple node at the
  default context level.
- `due_at` is an absolute timestamp; invalid or ambiguous local times fail.
- `reminder_id` must have originated from the granted list and target node.
- `create` and `delete` are `ask` by default; `list` and `complete` are
  `allow` only when explicitly granted. No action is allowed by default.

### `schedule.manage`

`create({ schedule_id, target_capability, target_action, target_arguments,
timezone, start_at, expiry_at, retry_policy, delivery_policy })`,
`list({})`, `pause({ schedule_id })`, `resume({ schedule_id })`, and
`cancel({ schedule_id })`.

- Schedules are Host-owned and target exactly one capability action and a
  canonical argument digest; they cannot target `schedule.manage` itself.
- `target_arguments` are validated against the target capability at creation
  and revalidated at each run; the schedule can never widen its grant.
- `timezone` is an explicit IANA zone; `start_at` and `expiry_at` are absolute
  instants, with expiry required for every schedule.
- `retry_policy` is bounded (maximum attempts and backoff); no unbounded retry.
- `delivery_policy` is `local_only` or `host_when_reachable`; it does not grant
  notification authority.
- `create`, `resume`, and `cancel` require `ask`; `list` and `pause` require an
  explicit `allow`. A scheduled run is a new invocation and is rechecked then.

### `notification.deliver`

`deliver({ recipient_node_id, kind, title, body, task_id?, deep_link? })`.

- `recipient_node_id` must be an explicitly granted notification target.
- `kind` is one of `task_update`, `approval_request`, or `system_status`.
- `body` is a privacy-safe preview capped by the node contract; secrets, raw
  prompts, credentials, and unsanitized external content are rejected.
- `deep_link`, when present, references a Lumen task or approval only.
- Delivery is ephemeral by default: retain only redacted delivery status as an
  operational record, not the notification body.
- `approval_request` requires `ask` and action-bound approval; task updates
  and system status require explicit `allow`. Default is deny.

## Policy outcomes

- `deny`: reject before dispatch with a stable reason; do not call the adapter.
- `ask`: persist `awaiting_permission`, show the exact action, target, scope,
  arguments digest, and expiry, then continue only after one matching approval.
- `allow`: execute only within the current grant and still perform argument,
  health, expiry, and target checks.

An offline node may execute only a locally cached, unexpired grant for actions
marked local-capable. It cannot perform cross-node or scheduled work, upload
context, or consume a newly issued approval until it reconnects to the Host.

## Acceptance checks

1. Unknown actions, malformed arguments, missing scopes, wrong target/resource,
   expired grants, revoked nodes, and stale schedules are rejected before the
   adapter runs.
2. For each action, fixtures prove deny, ask, and allow, including a reused,
   expired, or argument-mutated approval; only the exact one-time approval is
   accepted.
3. Duplicate commands return the original durable outcome and do not repeat the
   side effect; interrupted execution reports `unknown_outcome` until reconciled.
4. Coding fixtures reject path traversal, symlink escape, stale base refs,
   changed patch digests, and any apply outside the approved workspace.
5. Reminder fixtures reject ungranted lists and foreign IDs; schedule fixtures
   prove target grants are rechecked at run time and retries stop at expiry.
6. Notification fixtures prove redaction, recipient scoping, ephemeral body
   handling, and no authority through deep links.
7. Context fixtures prove the O-002 level is enforced before serialization and
   at Host persistence for all four capabilities.
