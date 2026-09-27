---
last_mapped_commit: ebe42cda0332c603644874b73ff920964ba853f1
last_mapped_at: 2026-09-26
---
<!-- refreshed: 2026-09-26 -->

# Architecture

**Analysis Date:** 2026-09-26

## System Overview

```text
 Owner CLI (`cmd/lumen`, `cmd/lumen-host`)
                  │ credentialed local request
                  ▼
 Host service (`internal/host`)
      ┌───────────┼────────────┐
      ▼           ▼            ▼
 Space rules   Hermes       setup lifecycle
 (`space`)     adapter       (`setup`)
      │        (`hermes`)        │
      └──────┬──────┘            ▼
             ▼                OS supervisors
 encrypted whole-state       (`deploy/`)
 store (`internal/store`)
```

The checked-out `main` commit is `ebe42cd`. The production core is a single Go Host process with an inward dependency direction: `internal/space` defines deterministic authority transitions; `internal/host` composes those transitions with storage, operator control, lifecycle, and the Hermes adapter. `internal/store` persists the canonical `space.State`; `internal/control` is the local operator boundary. Platform service files in `deploy/` supervise the Host rather than defining product semantics.

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Space authority | Types, commands, grants, task lifecycle, approvals, audit, idempotency | `internal/space/model.go`, `internal/space/apply.go` |
| Host composition | Initialize canonical state, compose runtime and store, own service lifecycle and dispatch | `internal/host/service.go`, `internal/host/execution.go` |
| Encrypted persistence | Lock, validate, read, encrypt, atomically replace canonical state | `internal/store/store.go`, `internal/store/envelope.go` |
| Local operator boundary | Credentialed Unix socket request/response and owner command client | `internal/control/server.go`, `internal/control/client.go` |
| Hermes runtime adapter | Query capabilities/health, submit runs, consume events and relay evidence | `internal/hermes/client.go`, `internal/hermes/events.go` |
| Installation and adoption | Platform probes, setup plan/journal, configuration, artifact and supervisor lifecycle | `internal/setup/runner.go`, `internal/setup/adoption.go` |
| Executable entry points | Host binary commands and owner setup/lifecycle CLI | `cmd/lumen-host/main.go`, `cmd/lumen/` |
| Platform supervision | launchd, systemd, Docker Compose, Termux/runit integration | `deploy/launchd/`, `deploy/systemd/`, `deploy/docker/`, `deploy/termux/` |
| Android companion | Quarantined Compose UI; it does not own or start the Host | `apps/android-host/src/main/kotlin/dev/lumen/android/host/LumenHostActivity.kt`, `CompanionScreenModel.kt` |

## Pattern Overview

**Overall:** Modular monolith with a pure domain transition core and explicit I/O adapters.

**Key Characteristics:**

- Commands enter the pure `space.Apply` transition function, which clones state, enforces command idempotency, dispatches by command type, and records receipts or rejection (`internal/space/apply.go:12-76`).
- The Host applies transitions through the store’s read/transition/commit operation (`internal/store/store.go:180-213`).
- Hermes is a replaceable runtime adapter; its own package documents that it neither persists state nor authorizes Lumen operations (`internal/hermes/client.go:1-2,181-199`).

## Layers

**Domain / Space:**

- Purpose: Define authority, identities, node/capability records, grants, tasks, approvals, and valid state transitions.
- Location: `internal/space/`
- Contains: Domain model, transition rules, policy helpers, audit records, fixture contract tests.
- Depends on: Go standard library only; no Host, Hermes, storage, transport, or UI packages.
- Used by: `internal/host/`, `internal/store/`, and contract tests.

**Host application service:**

- Purpose: Compose durable state, operator requests, Hermes execution, and service lifecycle.
- Location: `internal/host/`
- Contains: Configuration, create-once initialization, service, execution/recovery, and adapters.
- Depends on: `internal/space`, `internal/store`, `internal/control`, `internal/hermes`.
- Used by: `cmd/lumen-host/` and setup integration.

**Persistence adapter:**

- Purpose: Serialize access and protect canonical state on disk.
- Location: `internal/store/`
- Contains: Encrypted envelope, key and state file operations, atomic commits, migrations/tests for persisted format.
- Depends on: `internal/space` state types and Go filesystem/crypto primitives.
- Used by: `internal/host/`.

**Runtime adapter:**

- Purpose: Translate between Lumen execution requests and the documented Hermes API, returning bounded evidence.
- Location: `internal/hermes/`
- Contains: HTTP/TLS client, capability negotiation, run/event/approval/stop types.
- Depends on: Go HTTP, TLS, JSON, and context libraries.
- Used by: Host execution; it must not become the authority or persistence layer.

**Setup and platform layer:**

- Purpose: Plan and resume host installation, adoption, and supervision while keeping platform operations outside product rules.
- Location: `internal/setup/`, `cmd/lumen/`, `deploy/`, `scripts/`
- Contains: Platform-neutral setup stages/journal plus launchd, systemd, Docker, and Termux commands/configuration.
- Depends on: Host command/config contracts and OS tools at adapter edges.
- Used by: Owner-operated setup and release checks.

## Data Flow

### Host-local task path on main

1. `cmd/lumen-host/main.go:21-65` parses `doctor`, `init`, `serve`, status, task, and approval commands.
2. `internal/control/client.go:11-55` reads the operator credential and sends a bounded request to the Unix socket.
3. `internal/control/server.go:34-117,273-...` binds the private socket; request handling authenticates the credential before calling the Host handler.
4. `internal/host/service.go:457-492,518-539` creates and runs the service and routes the authenticated local command.
5. `internal/host/execution.go` validates task/approval arguments, applies Space transitions through `internal/store`, and invokes the Hermes adapter for runtime evidence.
6. `internal/store/store.go:180-213` commits the resulting state before a successful transition is returned.

### State persistence

1. Host code supplies a pure transition closure to `Store.Update` (`internal/store/store.go:180-213`).
2. Store reads and validates the current encrypted envelope, computes the next state, and commits it while holding its mutex and process lock (`internal/store/store.go:28-39,42-105`).
3. `internal/store/envelope.go:62-89,92-128` encodes state with AES-256-GCM and authenticates envelope metadata as additional data; decryption rejects invalid or unauthenticated envelopes.

**State Management:**

- The complete canonical `space.State` is stored as one encrypted JSON snapshot in `state.json`, with key material in a separate file; it is not a relational database (`internal/host/service.go:141-154`, `internal/store/store.go:42-105`).
- `space.Apply` returns a new state value and receipt/rejection rather than mutating the input in place (`internal/space/apply.go:5-28`).
- Host initialization bootstraps Space, owner, Host identity, capability advertisement, and the initial ask grant (`internal/host/service.go:407-423`).

## Key Abstractions

**`space.State` and `space.Command`:**

- Purpose: Canonical authority state and the typed inputs that can change it.
- Examples: `internal/space/model.go:74-92,176-224`.
- Pattern: Pure transition through `space.Apply`; use stable request IDs for replay protection.

**`store.Store`:**

- Purpose: Single-writer durable boundary for the complete Space state.
- Examples: `internal/store/store.go:28-39,180-213`.
- Pattern: `Update(func(space.State) space.Transition)`; never write authority state outside this boundary.

**`hermes.Adapter`:**

- Purpose: Versioned interface separating runtime mechanics from Host policy.
- Examples: `internal/hermes/client.go:181-199`.
- Pattern: Adapter methods return runtime evidence; Host owns authorization, durable state, and final outcomes.

## Entry Points

**Host daemon:**

- Location: `cmd/lumen-host/main.go:20-65,94-113`
- Triggers: Owner-run command or platform service manager.
- Responsibilities: Parse command, load configuration, initialize or serve the Host, install signal-driven shutdown.

**Owner CLI:**

- Location: `cmd/lumen/`
- Triggers: Owner installation, diagnosis, lifecycle, or task commands.
- Responsibilities: User-facing lifecycle and setup operations; inspect subcommands here before changing the CLI.

**Setup runner:**

- Location: `internal/setup/runner.go:12-43`
- Triggers: Setup CLI workflow.
- Responsibilities: Order durable setup stages and delegate platform mutation/verification through supplied callbacks.

## Architectural Constraints

- **Dependency direction:** Keep product state and rules in `internal/space`; adapters depend inward on its types, not vice versa.
- **Authority:** Host validates owner requests, applies grants/approvals, and records canonical outcomes. Hermes and runtime output are evidence only (`internal/hermes/client.go:1-2,181-199`).
- **Persistence:** One Host owns the store lock; store operations serialize through a mutex and OS flock (`internal/store/store.go:28-39,42-105,180-213`).
- **Local operator trust:** Unix socket requests require the independent operator credential; private parent and socket permissions are checked (`internal/control/server.go:34-117`).
- **Hermes separation:** Store no provider/runtime secrets in Space state; runtime credentials are loaded from restricted files by the Host (`internal/host/execution.go:92-126,128-...`).
- **Platform boundary:** Android companion is not the Host; the same headless Host artifact is supervised by OS-specific integration (`apps/android-host/src/main/kotlin/dev/lumen/android/host/CompanionScreenModel.kt:11-17`, `deploy/`).

## Anti-Patterns

### Putting policy in a runtime adapter

**What happens:** A Hermes client, helper, or model response is treated as permission to perform an operation.
**Why it's wrong:** Runtime observations are not canonical authority and may be stale, malformed, or untrusted.
**Do this instead:** Validate grants and apply state changes through `internal/space/apply.go` and `internal/host/`; keep `internal/hermes/` evidence-only.

### Letting platform setup define product behavior

**What happens:** launchd, systemd, Docker, or Termux branches implement distinct Space semantics.
**Why it's wrong:** Different deployment forms would produce different authority and recovery behavior.
**Do this instead:** Keep ordered, platform-neutral stages in `internal/setup/runner.go`; isolate OS commands in `deploy/` and `scripts/`.

## Error Handling

**Strategy:** Return errors at adapter boundaries, convert expected domain rejection to stable receipts/reasons, and fail closed when durable state or authenticated dependencies are unavailable.

**Patterns:**

- `space.Apply` represents policy rejection in `Transition.Rejection` and retains command replay information (`internal/space/apply.go:12-28,71-76`).
- Storage update returns no successful transition if reading or committing fails (`internal/store/store.go:180-213`).
- Host startup reports configuration/runtime and socket errors and closes owned resources (`internal/host/service.go:457-492`).
- Control handlers avoid returning state when the store read fails (`internal/host/service.go:518-525`).

## Cross-Cutting Concerns

**Logging:** Host runtime events use Go logging in execution paths; avoid recording secrets or user content (`internal/host/execution.go`).
**Validation:** Strict config, path, command and state checks live at the relevant boundaries (`internal/host/service.go:62-125`, `internal/control/`, `internal/store/store.go`).
**Authentication:** Owner control uses a restricted local credential; hardened Hermes communication uses TLS identity and bearer credentials (`internal/control/`, `internal/host/execution.go:92-126`).

## PR #20 branch-only architecture (not present on main)

The separate `feat/lumen-continuity` worktree at commit `0b67f81` adds schema-v2 conversations, persona/profile/certification records, migration, a conversation service, and a limited HTTPS signed node-status request/receipt. Evidence in that worktree: `internal/space/types.go:9-18,41-125`, `internal/space/state.go:9-40,84-150`, `internal/conversation/service.go:28-70,106-179`, `internal/host/conversation.go:12-23,25-89`, `internal/node/status.go:17-31,91-134,136-183`, and `internal/node/dispatch.go:16-46,48-69`. This status action is read-only and does not implement pairing or general remote device execution. The worktree also contains uncommitted edits to control timeouts and chat Docker/provider certification; these are not committed PR behavior and must be reviewed separately.

---

*Architecture analysis: 2026-09-26*
