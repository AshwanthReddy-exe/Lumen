# Phase 2 headless Host and Hermes

## Current execution lock

**This is the first work for the next session. Begin with the required one-time Android authority quarantine, then finish this block before returning to pairing, node transport, dashboards, camera, microphone, Mac, iPhone, or new Android companion features.** A UI mock, network scaffold, or additional companion feature is not progress toward this exit gate.

## Goal

Run one durable Lumen Host as a terminal service, connect it to a separately supervised Hermes runtime, and complete one bounded task with honest policy, events, approval, cancellation, failure, and restart behavior. The same Host distribution must work on Linux/VPS, macOS, and Android Termux.

## Observable live proof

1. Initialize a Space with `lumen-host init` into a private data directory.
2. Run an optional synthetic compatibility check against authenticated loopback, then start an isolated Hermes endpoint with pinned mutual TLS for the release proof.
3. Start `lumen-host serve` in a terminal, then repeat under a service manager.
4. Submit a typed task through `lumen-host task submit` using an idempotency key.
5. Observe accepted, running, approval-required when applicable, and final events through Host-owned status commands.
6. Stop one run, interrupt another during event delivery, restart the Host, and observe a durable cancelled, failed, reconciled, or `unknown_outcome` result.
7. Repeat initialization, service lifecycle, and one real task on Android Termux using the same distribution and contracts, with Hermes isolated on another machine or container behind pinned mutual TLS.

Before step 1, stop the superseded Android foreground Host and explicitly discard its test state or archive it as non-authoritative evidence. Record what was removed and whether it is recoverable. Never import it into the new Host automatically.

### Android authority quarantine evidence

The `fix/android-authority-quarantine` slice removes the Android foreground service, its canonical-state store and lifecycle, all Host creation/start/stop actions, both foreground-service permissions, and the APK dependency on `core:space`. The remaining activity is a disabled companion shell that states the Host runs separately. An upgrade install leaves the old encrypted prototype file in private app storage but no shipped code can read or activate it. The owner may preserve it as explicitly non-authoritative historical evidence or clear the app's data; neither path permits automatic import into the production Host.

## Components

### Host-local execution core increment

Retain `core/space` as the only owner of Space transitions, policy, approval consumption, task outcomes, audit rules, and restart recovery. Extend it before service work with an explicit Host-local execution target, durable dispatch/run mapping, `dispatched`, `running`, `cancelling`, and `cancelled` states, timeout behavior, terminal compare-and-set rules, and an evidence-bound reconciliation operation that can resolve `unknown_outcome`. Hermes may propose events; only the active Host principal commits them. The state codec uses migration fixtures for the expanded schema. The core still imports no CLI, filesystem, HTTP, service-manager, or Hermes code.

Initialization creates distinct recoverable owner and active Host identities. The active Host identity is the first local execution target and advertises the Hermes-backed capability. The loopback operator credential authenticates control requests and maps them to the owner principal, but it is not reused as an identity or encryption key.

### Headless Host service

Create `services/host` as a Kotlin/JVM 21 application. Its foreground process has explicit initialization, serve, readiness, graceful shutdown, and exit-code behavior. It serializes state changes through `SpaceHost`, exposes an owner-restricted authenticated Unix-domain operator socket, and emits redacted structured diagnostics. It never backgrounds or restarts itself.

### Durable Host storage

The Host stores encrypted, versioned canonical state in a configured private data directory. Initialization is create-only. Version 1 uses an explicit envelope containing format version, key identifier, cipher suite, fresh random nonce, authenticated metadata, and ciphertext. AES-256-GCM uses a new 96-bit nonce for every write; nonce reuse is forbidden. Phase 2 reads only version 1. Older, unknown, or future versions fail closed without mutation. A future migration must write and verify a separate file, durably flush it, atomically replace the active file, and retain or quarantine the prior version for explicit rollback.

Commits use one serialized writer, atomic replacement, durable flush, and a process lock. The state key comes from a restricted secret file or service-manager credential and is distinct from the operator and Hermes credentials. Missing keys, permissive secret access, corrupt ciphertext, incompatible versions, lock contention, and failed writes leave the Host unavailable without overwriting the last committed state.

### Operator CLI

The CLI supports `init`, `serve`, `status`, `task submit`, `task show`, `task cancel`, `approval resolve`, and `shutdown`. Commands sent to a running Host use an owner-restricted Unix-domain socket, bounded request sizes, an independent operator credential, and pinned Host endpoint identity. Credentials never appear in flags, process listings, prompts sent to Hermes, or ordinary logs.

### Hermes Runtime Adapter

Create a narrow, versioned adapter around [Hermes’s documented API server](https://hermes-agent.nousresearch.com/docs/developer-guide/programmatic-integration):

- `GET /v1/capabilities` for compatibility negotiation;
- `GET /health` and authenticated `/health/detailed` for liveness and readiness;
- `POST /v1/runs` for idempotent run submission;
- `GET /v1/runs/{id}` and `/events` for status and bounded SSE events;
- `/approval`, `/steer`, and `/stop` only after Host policy authorizes the exact operation.

This first slice freezes the common boundary used later by Hermes model routing, built-in tools, skills, MCP, browser, voice adapters, delegated runs, and remote execution. It does not recreate or enable every feature in Block 2. Capability metadata is imported as untrusted discovery data; every enabled Hermes surface still requires a typed Lumen capability, target, context scope, budget, cancellation, and audit contract.

Hardened Linux, macOS, and VPS deployments isolate Host and Hermes with separate OS principals or containers. Hermes is reached through a protected TLS endpoint or proxy with a pinned server identity, client authentication, and an independently scoped Hermes bearer credential. The adapter rejects redirects, URL userinfo, pin changes, and credential forwarding to another origin. Plain loopback HTTP is allowed only in an explicit development profile with synthetic state and cannot satisfy the security exit gate.

Android Termux runs the same Host artifact and contracts. Because processes inside one Termux installation share an Android UID, runit and proot do not isolate Hermes from Host secret files. A co-located Termux Hermes run is compatibility-only and uses synthetic, non-sensitive state. The security-valid Termux proof connects to Hermes isolated on another machine or container through pinned mutual TLS; compromise of the Termux UID remains Host compromise.

Capability flags are necessary but not sufficient. The release records a tested Hermes version or commit plus behavioral probes for run creation, the sole Host-owned SSE consumer, status reconciliation, exact approval, stop, authentication, and bounded retention. At design time, upstream reports document [approval bypass under safe defaults](https://github.com/NousResearch/hermes-agent/issues/98728) and [single-consumer run events](https://github.com/NousResearch/hermes-agent/issues/103262); the adapter must fail its real-Hermes gate if those behaviors violate the pinned contract. The Host stores its own task-to-run mapping and normalized events. Hermes responses, tool requests, artifacts, and completion claims are untrusted evidence.

### Process supervision

Ship examples for systemd, launchd, Docker, and Termux/runit. All invoke the same `lumen-host serve` contract, supply secrets without command-line exposure, use a private data directory, stop gracefully, and apply bounded restart policy. Supervisor state is operational evidence, not Space authority.

## Failure contract

| Condition | Host behavior | Task outcome |
| --- | --- | --- |
| State/key unavailable | Readiness fails; no task is accepted. | No new task exists. |
| Hermes unavailable or incompatible before dispatch | Persist rejection/degraded evidence. | Failed or unavailable, never dispatched. |
| Duplicate submit with same key and digest | Return the durable prior result. | Unchanged. |
| Duplicate key with different digest | Reject as collision. | Unchanged. |
| SSE duplicates or reordering | Apply only Host-owned lifecycle transition keys and compare-and-set state rules; retain repeated user-visible progress without assuming Hermes supplies a durable event identity. | No repeated authority transition. |
| SSE disconnect after dispatch | Reconcile through run status within a bounded deadline. | Reconciled terminal result or `unknown_outcome`. |
| Host dies after dispatch | Restart recovery uses durable dispatch evidence before retry. | Never redispatched without an idempotent contract. |
| Approval request | Match exact task, action, arguments/digest, actor, runtime request, and expiry. Forward only Hermes `once` or `deny`; reject unknown, `session`, and `always` choices. | Remains awaiting approval until a valid one-time decision or expiry. |
| Cancellation races completion | Persist the first valid terminal evidence under compare-and-set rules. | One honest terminal outcome. |

## Validation gates

- Unit tests cover core execution transitions and migration fixtures; configuration; create-only initialization; encrypted envelopes, fresh nonces, corruption, permissions, locking, and atomic write failure; lifecycle reducers; adapter normalization; bounds; and secret redaction.
- Contract tests use a fake Hermes server for capability mismatch, authentication failure, run creation, SSE duplication/reordering/disconnect, approval, cancellation, timeout, and recovery.
- An opt-in real-Hermes test verifies the documented capability, run, event, exact `once`/`deny` approval, stop, and health surface under deny-by-default configuration without relying on Hermes internals.
- `mise run phase1-check` remains green.
- Gradle must compile with an explicit Java 21 toolchain and release target. A new `mise run phase2-check` must run on Java 21, build the distribution, and run core, Host, adapter, persistence, CLI, and scenario tests. Android Termux aarch64 is a required compatibility run with its exact Termux and OpenJDK versions recorded; it is not assumed equivalent to desktop Linux.
- Owner evidence records one supervised desktop/server run and one Android Termux run with no manual state repair.

## Delivery sequence and PR stack

1. `docs/host-first-plan`: canonical architecture correction and this plan.
2. `fix/android-authority-quarantine`, based on the planning branch: complete; remove the Android foreground Host entry points and prevent the superseded test Space from presenting as authority.
3. `feat/host-execution-contract`, based on the prior slice: freeze owner/Host-executor identity, dispatch/run evidence, running/cancellation states, reconciliation, codec migration, and core tests.
4. `feat/host-service-bootstrap`, based on the prior slice: JVM 21 distribution, CLI lifecycle, configuration, health, and supervisor examples.
5. `feat/host-durable-store`, based on the prior slice: version 1 encrypted envelope, key separation, locking, atomic commits, recovery, and failure tests.
6. `feat/hermes-runtime-adapter`, based on the prior slice: mutual endpoint identity, isolated deployment profile, pinned behavioral compatibility, authenticated Runs API client, normalized events, and fake-server contract tests.
7. `feat/host-hermes-loop`, based on the prior slice: task submission, approval, cancellation, reconciliation, end-to-end scenario, `phase2-check`, and real-Hermes smoke evidence.

The owner explicitly requested a stacked series that can be reviewed before any merge. Each PR targets the preceding branch and is merged bottom-up. After a lower PR merges, fetch and fast-forward `main`, retarget the next PR to `main`, and update it without force-pushing or rewriting shared history. Protocol, persistence, authorization, and migration changes require independent review before merge.

## Exit

Block 2 is complete only when the same Host distribution passes the automated gate, survives forced restart with honest task state, completes a real Hermes task from the Host CLI, and repeats the lifecycle on Android Termux. Until then, nodes and companions remain intentionally blocked.
