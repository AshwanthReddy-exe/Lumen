# Go Host rearchitecture design

## Goal

Replace the planned Kotlin/JVM Host and portable Kotlin Space core with a small native Go service while retaining native Kotlin Android and Swift Apple applications. Preserve every accepted authority, permission, durability, recovery, and Hermes boundary through executable cross-language fixtures before deleting the Kotlin reference implementation.

## Decision

Go owns the production Host executable, Space authority rules, local control socket, encrypted storage, and Hermes adapter. Android remains Kotlin with Jetpack Compose; Apple nodes remain Swift and SwiftUI. Nodes never link the Host implementation. They share versioned wire schemas, canonical fixtures, and conformance tests.

Go is preferred over Rust because Lumen needs a networked service and policy state machine rather than memory-unsafe native integration. Go provides native ARM64 and AMD64 binaries, a small operational surface, fast builds, simple concurrency, and standard-library HTTP, TLS, cryptography, JSON, filesystem, and Unix-socket support. Rust would improve low-level memory control but add implementation, review, and mobile-FFI cost without a demonstrated V1 requirement.

## Invariants

- One active Host owns canonical Space state, policy, task state, approvals, and audit.
- Hermes, models, tools, MCP servers, nodes, and external content remain untrusted evidence.
- The Host may orchestrate many capabilities but never receives a blanket authority grant.
- `agent.run/execute` is the first Host-local Hermes orchestration capability and defaults to `ask`.
- Browser, shell, coding, reminders, device hardware, remote execution, and other effects retain separately typed grants.
- Every state change has an authority, validation point, idempotency key, durable record, retry rule, cancellation behavior, and honest uncertain outcome.
- Android and Apple applications cannot create, open, or mutate canonical Host state directly.
- No Kotlin production authority code is deleted until Go parity checks pass.
- No background model or tool work occurs without explicit invocation or an explicitly granted trigger.

## Code boundaries

```text
cmd/lumen-host/          process entry point and operator CLI
internal/space/          pure authority state and transitions
internal/control/        authenticated local Unix-socket protocol
internal/store/          locked, versioned, encrypted atomic state
internal/hermes/         versioned Hermes Runs API adapter
internal/host/           thin orchestration across inward contracts
protocol/                schemas, canonical JSON, compatibility fixtures
test/contract/           black-box Host and Hermes contract scenarios
deploy/                  systemd, launchd, Docker, and Termux/runit examples
apps/android-host/       temporary package name; companion-only Kotlin app
```

Dependencies point inward. `internal/space` imports only the Go standard library and contains no filesystem, networking, CLI, platform, or Hermes code. `internal/host` composes narrow contracts; generic adapters are introduced only when two real implementations need them.

## Space and execution model

Initialization creates distinct owner and active-Host identities, even when they occupy one physical machine. The active Host advertises `agent.run/execute`; its initial grant is `ask`. A task cannot dispatch until its exact grant and any required one-time approval are durable.

Host-local execution uses these durable states:

```text
awaiting_permission -> queued -> dispatched -> running
                                      |           |
                                      +-> cancelling -> cancelled
                                      +-----------+-> completed | failed
                                      +-----------+-> unknown_outcome
unknown_outcome --matching runtime evidence--> completed | failed | cancelled | running
```

Before calling Hermes, the Host commits the task-to-runtime-run mapping, dispatch timestamp, reconciliation deadline, Host epoch, and immutable runtime-profile digest. Only the active Host identity may commit Hermes evidence. Only the owner or an explicitly authorized origin may request cancellation; the Host records `cancelling` before forwarding `/stop`. The first valid terminal transition wins. A disconnect or restart never implies success, failure, or cancellation.

## Compatibility migration

The existing Kotlin tests are behavioral evidence, not an API to translate class-for-class. The migration first freezes canonical JSON fixture cases containing command sequences and expected state, receipt, rejection, and redacted-audit projections. Existing Phase 1 behavior runs against both Kotlin and Go until parity is established. Newly designed dispatch, cancellation, timeout, and reconciliation behavior is frozen as literal fixtures before its Go implementation.

Required fixture families cover Space creation, separate owner/Host identity, pairing, capability advertisement, deny/ask/allow, exact approval, idempotent replay and collision, revocation, stale epochs, durable dispatch, run-ID collision, running progress, cancellation races, timeout, restart, unknown outcome, and evidence-bound reconciliation. Expected values are literal and must not be generated by either implementation.

After Go passes all fixtures and the black-box scenario, remove `core/space`, `tools/space-scenario`, the Kotlin Multiplatform root plugin, and obsolete Phase 0 build spikes from active checks. Preserve their Git history and record the removal in the changelog. The Android module remains a standalone Gradle application and never embeds Go or depends on Host internals.

## Host process and storage

`lumen-host` is one foreground binary with `init`, `serve`, `status`, `task submit`, `task show`, `task cancel`, `approval resolve`, and `shutdown`. It does not daemonize or restart itself. Supervisors own restart policy.

The running service accepts bounded requests through an owner-restricted Unix-domain socket. An independent operator credential maps authenticated requests to the owner identity. Operator, state-encryption, Host-identity, and Hermes credentials are distinct and never appear in command-line arguments or ordinary logs.

State uses a versioned authenticated envelope and AES-256-GCM with a fresh 96-bit nonce per write. Writes use one process lock, serialized mutation, a same-directory temporary file, file sync, atomic replacement, and parent-directory sync. Unknown versions, corrupt ciphertext, permissive secret permissions, missing keys, lock contention, and write failure fail closed without replacing the last committed state.

## Hermes boundary

The Go adapter uses Hermes's documented capabilities, health, runs, events, approval, steering, and stop endpoints. It rejects redirects, URL userinfo, endpoint pin changes, oversized responses, incompatible capabilities, and credential forwarding across origins. SSE is lossy progress evidence, never an authority log; one Host-owned consumer normalizes it and bounded status reconciliation establishes the best honest outcome.

Development may use explicit plaintext loopback with synthetic state. Hardened macOS, Linux, VPS, and Termux proofs require Hermes under a separate principal or container behind pinned mutual TLS. A Hermes process sharing the Termux UID is compatibility-only because it can access the Host's files.

## Verification and delivery

Development follows strict red-green-refactor TDD. Every behavior test must fail for the intended missing behavior before production code is added. Each block ends with unit, negative, recovery, race, and black-box checks appropriate to its boundary.

Block 2 delivery order is:

1. Freeze cross-language fixtures and implement Go Space parity.
2. Implement the foreground Host and authenticated CLI lifecycle.
3. Implement encrypted atomic storage and failure injection.
4. Implement the authenticated Hermes adapter and fake-server contracts.
5. Compose task submission, events, exact approval, cancellation, timeout, restart, and reconciliation.
6. Add systemd, launchd, Docker, and Termux/runit examples.
7. Prove the same binary contract on macOS/Linux and Android Termux ARM64.
8. Remove superseded Kotlin authority code and run the final combined gate.

The final combined pull request targets `main`. Intermediate branches are recoverable checkpoints and do not require separate pull requests.

## Non-goals

This rearchitecture does not implement node transport, Android companion pairing, camera, microphone, speaker, Mac coding, iPhone reminders, browser actions, schedules, remote relay, or multi-Host failover. It establishes the Host that those later blocks depend on.
