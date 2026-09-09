# Go Host Block 2 Implementation Plan

> Historical implementation plan. Canonical current status and release gates are in [the delivery plan](../../PLAN.md) and [the active Phase 2 document](../../PHASE-2-HOST-HERMES.md).

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship one native Go Lumen Host that durably authorizes and completes a bounded Hermes run on macOS/Linux and Android Termux with honest approval, cancellation, restart, and uncertain-outcome behavior.

**Architecture:** A pure `internal/space` state machine owns authority and immutable transitions. Thin `internal/host` orchestration commits through an encrypted `internal/store`, accepts local authenticated commands through `internal/control`, and treats the `internal/hermes` adapter as untrusted evidence. Native applications communicate through versioned protocol fixtures and never import Host internals.

**Tech Stack:** Go 1.27.1 standard library; JSON v1 fixtures; AES-256-GCM; Unix-domain sockets; HTTP/SSE with TLS 1.3; Kotlin/Compose retained only for Android.

**Spec:** `docs/superpowers/specs/2026-09-08-go-host-rearchitecture-design.md`

## Global constraints

- One active Host is the only canonical Space authority.
- `agent.run/execute` is advertised at initialization with grant `ask`; it grants no browser, shell, device, account, coding, or remote-execution authority.
- Hermes, its events, models, tools, MCP servers, and output are untrusted evidence.
- `internal/space` imports only the Go standard library and performs no I/O.
- Every mutating command has a non-empty idempotency key; exact replay returns the first receipt and changed content collides.
- State, operator, Host-identity, and Hermes credentials are distinct and never accepted through command-line flags.
- Hardened Hermes connections use TLS 1.3, a pinned server certificate/public-key identity, client authentication, no redirects, and one scoped bearer credential.
- Plain HTTP is allowed only for explicit synthetic loopback development.
- Tests follow red-green-refactor and assert real behavior with literal expected values.
- Do not delete the Kotlin reference core until every Go parity, scenario, build, and Android quarantine check passes.

---

### Task 1: Go toolchain, canonical fixtures, and state types

**Files:**
- Create: `go.mod`
- Create: `protocol/fixtures/space-v1.json`
- Create: `internal/space/model.go`
- Create: `internal/space/fixture_test.go`
- Modify: `mise.toml`

**Interfaces:**
- Produce `space.State`, `space.Command`, `space.Transition`, `space.Receipt`, and `space.Apply(State, Command) Transition`.
- Fixture top level is `{ "schemaVersion": 1, "cases": [...] }`; each case contains literal `initial`, `commands`, and `expected` projections.

- [ ] Write `fixture_test.go` first. It loads `protocol/fixtures/space-v1.json`, rejects an unsupported schema version, and table-runs cases through `Apply`; the first case creates `space`, separate `owner` and `host`, epoch `1`, paired identities, `agent.run/execute`, grant `ask`, and one redacted create audit event.
- [ ] Run `go test ./internal/space -run TestFixtures -count=1`; verify RED because the module/types/fixture do not exist.
- [ ] Add `go.mod` with module `github.com/AshwanthReddy-exe/Lumen` and `go 1.27.1`; add `go = "1.27.1"` to mise.
- [ ] Implement JSON-tagged state types with explicit string enums and a strict fixture decoder using `json.Decoder.DisallowUnknownFields()`.
- [ ] Implement only create-space validation and transition behavior needed by the first fixture.
- [ ] Run the targeted test, then `go test ./...`; verify GREEN.
- [ ] Commit `feat(space): establish Go authority fixtures`.

### Task 2: Phase 1 authority parity

**Files:**
- Modify: `protocol/fixtures/space-v1.json`
- Create: `internal/space/apply.go`
- Create: `internal/space/policy.go`
- Create: `internal/space/audit.go`
- Modify: `internal/space/fixture_test.go`
- Create: `test/contract/space_scenario_test.go`

**Interfaces:**
- `Apply` supports `pair_node`, `advertise_capability`, `set_grant`, `submit`, `approve`, `complete`, `revoke_node`, and `recover_after_restart`.
- A transition returns exactly one `Receipt` or stable `Rejection`; state includes recorded command content/outcome for replay.

- [ ] Add literal fixtures for pairing, deny/ask/allow, missing advertisement, exact approval, expired/mismatched approval, stale epoch, duplicate replay, changed-content collision, revocation, completion, and Phase 1 remote-task restart.
- [ ] Run `go test ./internal/space -run TestFixtures -count=1`; verify RED on the first unsupported command.
- [ ] Implement command dispatch and shared validation in `apply.go`; keep capability checks in `policy.go` and redacted event construction in `audit.go`.
- [ ] Add a black-box three-node test that executes deny, ask/approval, allow, idempotency, revocation, and simulated restart through exported APIs.
- [ ] Run targeted fixtures and `go test ./...`; verify GREEN and run `go test -race ./...`.
- [ ] Commit `feat(space): port authority contract to Go`.

### Task 3: Host-local run lifecycle and recovery

**Files:**
- Modify: `protocol/fixtures/space-v1.json`
- Create: `internal/space/execution.go`
- Modify: `internal/space/model.go`
- Modify: `internal/space/apply.go`
- Create: `internal/space/execution_test.go`

**Interfaces:**
- Commands: `dispatch_host_run`, `reconcile_host_run`, `request_host_run_cancellation`.
- `HostRun` contains `taskId`, `runtimeRunId`, `runtimeProfileDigest`, `dispatchedAt`, and `reconcileBy`.
- States: `queued`, `dispatched`, `running`, `cancelling`, `cancelled`, `completed`, `failed`, `unknown_outcome`.
- Evidence outcomes: `running`, `completed`, `failed`, `cancelled`, `unavailable`.

- [ ] Add fixtures proving active-Host-only dispatch, exact unique run/profile mapping, invalid deadlines, owner-only cancellation, running/cancellation progression, timeout to unknown, restart recovery, evidence-bound unknown resolution, duplicate/reordered evidence, and first-terminal-wins.
- [ ] Run the fixture test; verify RED on `dispatch_host_run`.
- [ ] Implement the minimum lifecycle and compare-and-set rules. Commit mapping before external dispatch is represented by the `dispatched` transition; no adapter call exists in this package.
- [ ] Add focused table tests mutating actor, epoch, task, run ID, profile digest, timestamp, status, and terminal evidence.
- [ ] Run `go test ./internal/space -count=1` and `go test -race ./...`; verify GREEN.
- [ ] Commit `feat(space): add durable Host run lifecycle`.

### Task 4: Foreground Host, configuration, and authenticated local CLI

**Files:**
- Create: `cmd/lumen-host/main.go`
- Create: `internal/control/protocol.go`
- Create: `internal/control/server.go`
- Create: `internal/control/client.go`
- Create: `internal/control/control_test.go`
- Create: `internal/host/service.go`
- Create: `internal/host/service_test.go`

**Interfaces:**
- CLI: `lumen-host init|serve|status|task submit|task show|task cancel|approval resolve|shutdown`.
- `serve` stays foreground and exits `0` after authenticated shutdown, `2` for usage, `3` for unavailable state/configuration, and `4` for runtime incompatibility.
- Local requests are newline-delimited JSON capped at 64 KiB over a Unix socket with mode `0600`; authentication uses a constant-time comparison of a 32-byte credential read from a `0600` file.

- [ ] Write subprocess and socket tests first for usage errors, foreground readiness, unauthorized request, oversized request, status, authenticated shutdown, socket permissions, and secrets absent from process arguments/output.
- [ ] Run targeted tests and verify RED because the command/server do not exist.
- [ ] Implement configuration from environment and restricted files, never secret flags; implement one-request/one-response bounded framing.
- [ ] Implement the thin service lifecycle with explicit readiness and graceful context cancellation.
- [ ] Run `go test ./internal/control ./internal/host ./cmd/lumen-host -count=1`, `go test -race ./...`, and build `./cmd/lumen-host`.
- [ ] Commit `feat(host): add foreground service and operator CLI`.

### Task 5: Encrypted durable store

**Files:**
- Create: `internal/store/envelope.go`
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`
- Create: `internal/store/testdata/state-v1.json`
- Modify: `internal/host/service.go`

**Interfaces:**
- Envelope fields: `formatVersion`, `keyId`, `cipherSuite`, `nonce`, `metadata`, `ciphertext`.
- `Store.Initialize(State)`, `Store.Read()`, and `Store.Update(func(State) Transition)` serialize one writer under a non-blocking process lock.
- Cipher suite string is `AES-256-GCM`; nonce is 12 random bytes per write; key file is exactly 32 bytes with no group/other permissions.

- [ ] Write tests first for create-only initialization, round trip, fresh nonce, authenticated metadata, wrong/missing key, permissive key mode, corrupt/truncated/future envelope, lock contention, injected write/sync/rename failure, and preservation of the last committed state.
- [ ] Run `go test ./internal/store -count=1`; verify RED.
- [ ] Implement strict envelope parsing, AES-GCM, `0600` files, same-directory temp write, file sync, atomic rename, and parent-directory sync using standard library/syscall facilities supported by macOS and Linux/Termux.
- [ ] Integrate the store behind the Host service without exposing storage types to `internal/space`.
- [ ] Run store tests, `go test -race ./...`, and `go vet ./...`.
- [ ] Commit `feat(store): persist encrypted Host authority`.

### Task 6: Authenticated Hermes runtime adapter

**Files:**
- Create: `internal/hermes/client.go`
- Create: `internal/hermes/events.go`
- Create: `internal/hermes/client_test.go`
- Create: `internal/hermes/fake_test.go`

**Interfaces:**
- Methods: `Capabilities`, `Health`, `CreateRun`, `RunStatus`, `Events`, `ResolveApproval`, `Steer`, `Stop`.
- `Config` requires base URL, profile mode, maximum response/event sizes, timeouts, bearer credential source, and hardened TLS identity material.
- The HTTP client sets `CheckRedirect` to reject every redirect and never follows a new origin.

- [ ] Write fake-server tests first for capability mismatch, health authentication, exact headers and idempotency key, run creation, bounded SSE parsing, duplicate/reordered events, disconnect, approval values limited to `once` or `deny`, stop, redirect, URL userinfo, oversized data, timeout, and TLS pin/client-certificate mismatch.
- [ ] Run `go test ./internal/hermes -count=1`; verify RED.
- [ ] Implement the smallest standard-library HTTP/SSE client and strict JSON decoders. Do not persist state or make policy decisions in the adapter.
- [ ] Run adapter tests, `go test -race ./...`, and `go vet ./...`.
- [ ] Commit `feat(hermes): add constrained Runs API adapter`.

### Task 7: Durable Host-to-Hermes loop

**Files:**
- Modify: `internal/host/service.go`
- Create: `internal/host/execution.go`
- Create: `internal/host/execution_test.go`
- Modify: `internal/control/server.go`
- Create: `test/contract/host_hermes_test.go`

**Interfaces:**
- The service persists submit/approval, creates a Hermes run, persists `dispatch_host_run`, owns one SSE consumer, normalizes evidence, and persists every authority transition before replying.
- On disconnect/restart it polls bounded run status until `reconcileBy`, then records a proven terminal result or `unknown_outcome`.
- Cancellation persists `cancelling` before calling `Stop`; a race commits the first valid terminal evidence.

- [ ] Write fake-runtime end-to-end tests first for allow/ask/deny, create failure before dispatch, crash after dispatch, SSE progress, exact approval, stale approval, cancellation race, stop failure, duplicate events, disconnect, status reconciliation, restart, and unknown outcome.
- [ ] Run `go test ./test/contract -run TestHostHermes -count=1`; verify RED.
- [ ] Implement orchestration only in `internal/host/execution.go`; adapters return evidence and never mutate Space state.
- [ ] Wire CLI task/status/approval/cancel commands to the authenticated control server.
- [ ] Run all Go tests, race tests, vet, and a subprocess scenario using the fake Hermes server.
- [ ] Commit `feat(host): complete durable Hermes execution loop`.

### Task 8: Distribution, supervisors, Termux proof, and Kotlin-core retirement

**Files:**
- Create: `deploy/systemd/lumen-host.service`
- Create: `deploy/launchd/dev.lumen.host.plist`
- Create: `deploy/docker/Dockerfile`
- Create: `deploy/termux/run`
- Create: `test/contract/supervisor_test.go`
- Modify: `mise.toml`
- Modify: `README.md`
- Modify: `docs/PLAN.md`
- Modify: `docs/PHASE-2-HOST-HERMES.md`
- Modify: `docs/CHANGELOG.md`
- Modify: `settings.gradle.kts`
- Modify: `build.gradle.kts`
- Delete after parity: `core/space/`
- Delete after parity: `tools/space-scenario/`

**Interfaces:**
- `mise run phase2-check` runs format verification, `go vet`, all Go tests, race tests where supported, native build, cross-build for `linux/arm64`, Android companion unit/build checks, and fixture/scenario checks.
- Supervisor definitions call only `lumen-host serve`, load secrets from restricted files/environment, use bounded restart, and send graceful termination.

- [ ] Write tests that parse each supervisor definition and launch the foreground binary under a temporary directory; verify no secret flags, private paths, graceful shutdown, and bounded restart settings.
- [ ] Run supervisor tests and verify RED before definitions exist.
- [ ] Add supervisor/container definitions and reproducible macOS/Linux/ARM64 builds.
- [ ] Add `phase2-check`; run it before deleting Kotlin authority code.
- [ ] Delete Kotlin `core/space` and `tools/space-scenario`, remove unused KMP/serialization/JVM root plugins and includes, and keep the Android Compose build independent.
- [ ] Run `phase2-check` again and `graphify update .`.
- [ ] On connected Android/Termux, record exact Android, Termux, architecture, and Go versions; run `lumen-host init`, foreground and runit lifecycle, authenticated status/shutdown, restart recovery, and one real isolated-Hermes task. Do not treat same-UID Hermes as security evidence.
- [ ] Update canonical docs with exact automated and owner evidence and remaining risks.
- [ ] Dispatch independent protocol/persistence/authorization review, resolve findings, and create the combined Block 2 pull request against `main` only after every exit criterion passes.
- [ ] Commit `feat(host): complete Block 2 native distribution`.
