# Phase 02: Close the existing Host/Hermes foundation - Pattern map

**Mapped:** 2026-09-27  
**Files analyzed:** 15 existing implementation, test, and probe files likely to be reused or narrowly changed  
**Analogs found:** 15 / 15

## File classification

| New/modified or reused file | Role | Data flow | Closest analog | Match quality |
|---|---|---|---|---|
| `internal/host/service.go` | service | CRUD / durable state | `internal/host/service_test.go` | role-match |
| `internal/host/service_test.go` | test | CRUD / recovery | `internal/host/service_test.go` | exact |
| `internal/setup/runner.go` | service | staged / resumable setup | `internal/setup/runner_test.go` | role-match |
| `internal/setup/adoption.go` | service | artifact I/O / validation | `internal/setup/adoption_test.go` | role-match |
| `internal/setup/doctor.go` | service | request-response / status | `internal/setup/doctor_test.go` | role-match |
| `internal/setup/supervisor.go` | service | process lifecycle | `internal/setup/supervisor_test.go` | role-match |
| `cmd/lumen/release.go` | controller | artifact I/O / recovery | `cmd/lumen/release_test.go` | role-match |
| `internal/hermes/client.go` | service / adapter | request-response / streaming | `internal/hermes/client_test.go` | role-match |
| `internal/host/execution.go` | service | event-driven / streaming | `internal/host/execution_test.go` | role-match |
| `test/contract/setup_hermes_run_test.go` | test | request-response / streaming | `test/contract/setup_hermes_run_test.go` | exact |
| `test/contract/supervisor_test.go` | test | process lifecycle | `test/contract/supervisor_test.go` | exact |
| `test/contract/setup_journey_test.go` | test | staged / resumable setup | `test/contract/setup_journey_test.go` | exact |
| `scripts/lumen-linux-check` | probe script | deployment / file I/O | `scripts/lumen-linux-check` | exact |
| `scripts/lumen-macos-check` | probe script | deployment / streaming | `scripts/lumen-macos-check` | exact |
| `scripts/lumen-external-check` | probe script | deployment / request-response | `scripts/lumen-external-check` | exact, synthetic endpoint only |

These are likely reuse points, not a mandate to modify every file. Phase 02 requires probe-first closure; extend a source file only when a qualifying probe finds a concrete gap. No Phase 04+ conversation, memory, or node-status files are in scope.

## Pattern assignments

### Host bootstrap and recovery

**Files:** `internal/host/service.go`, `internal/host/service_test.go`  
**Analog:** `internal/host/service.go` and its `internal/host/service_test.go` tests.

`Initialize` validates configuration, restricts the data directory, detects partially written state, and resumes only when both the state and key exist (`service.go:127-148`). It creates Space, owner, and Host IDs once, persists the initial state and credential, then writes the initialized marker (`service.go:191-231`). Reuse `VerifyInitialized` for stable identity evidence on reruns; do not add another identity store or make setup reruns recreate state. Extend the existing init/recovery tests for only the observed failure case.

### Setup, adoption, doctor, and supervisor

**Files:** `internal/setup/runner.go`, `internal/setup/adoption.go`, `internal/setup/doctor.go`, `internal/setup/supervisor.go`, `cmd/lumen/main.go`  
**Analogs:** `internal/setup/runner_test.go`, `internal/setup/adoption_test.go`, `internal/setup/doctor_test.go`, `internal/setup/supervisor_test.go`, and CLI coverage in `cmd/lumen/main_test.go`.

- `SetupRunner.Run` validates profile/topology and journal binding, then advances the existing ordered stages. Host initialization is verified even after an ambiguous initializer error before evidence is journaled (`runner.go:34-141`). Preserve this durable-resume behavior; add a stage or alternate journal only if an existing contract cannot represent a demonstrated state.
- Adoption probes and validates candidates through the current `AdoptExternalHermes` and endpoint-file validation path (`adoption.go`). Extend its candidate tests for a proven incompatibility; do not copy artifacts into a new helper layer.
- `Doctor.Check` builds its result from observed evidence and emits only the defined outcomes (`doctor.go:42-160`; exact outcome values are in `internal/setup/model.go:5-19`). Reuse this evidence-to-status mapping in probes. Missing Hermes may be degraded; invalid binding, credentials, supervisor, boot, or Host readiness stays action-required.
- Reuse `CommandSupervisor` and its platform-specific tests for install/start/status behavior. Boot and reboot claims still require the named live platform; contract tests do not establish them.

### Artifact update and rollback

**File:** `cmd/lumen/release.go`  
**Analog:** `cmd/lumen/release_test.go`.

`releaseChangeCommand` locks deployment state, requires a validated setup journal, retains the current executable, writes a pending record, installs or restores the generation, verifies its digest, commits the record, clears pending state, and restarts (`release.go:80-149`). Reuse `RetainExecutableGeneration`, `SaveReleaseRecord`, and `recoverPendingRelease` with existing tests. Image updates explicitly return `image_update_requires_deployment` (`release.go:108-113`); preserve this restriction unless a tested, safe container replacement path exists.

### Hermes Runs and Host reconciliation

**Files:** `internal/hermes/client.go`, `internal/hermes/events.go`, `internal/host/execution.go`  
**Analogs:** `internal/hermes/client_test.go`, `internal/host/execution_test.go`, and `test/contract/setup_hermes_run_test.go`.

- Keep `hermes.Adapter` a narrow transport/evidence boundary; it exposes capabilities, health, create/status/events, approval, steer, and stop while leaving policy and Space mutation to the Host (`client.go:181-199`). Extend request decoding or validation at this boundary only for an observed pinned-runtime incompatibility.
- The Host consumes events within the task reconciliation deadline, persists approval/event evidence, deduplicates seen event IDs, and reconciles after stream loss (`execution.go:828-870`, `932-970`). Reuse the existing durable task and first-terminal-wins tests when validating lost streams, stop, and restart; never interpret adapter output alone as task authority.
- Contract tests already exercise configured runtime approval once/deny and exact create counts (`setup_hermes_run_test.go:370-410`). Fakes establish control flow, not pinned Hermes behavior. Ordinary clarification stays disabled if the pinned API does not expose a certified answer path.

### Host-local capability boundary

**File:** `internal/host/execution.go`  
**Analog:** `internal/host/execution_test.go`.

Reuse existing local capability authorization and action-bound approval tests, especially `TestAskRequiresExactApprovalAndStartsOnlyOnce` and `TestInitialApprovalDenyRequiresExactPendingBinding`. Preserve capability ID `agent.run/execute`, one-time approval binding, and no transitive grant. Do not introduce new capability plumbing for E1.

### Deployment probes and evidence

**Files:** `scripts/lumen-linux-check`, `scripts/lumen-macos-check`, `scripts/lumen-external-check`  
**Analogs:** each script's existing private work directory, pinned-artifact verification, cleanup trap, redaction, readiness polling, and JSON outcome checks.

- For named Linux combined-profile checks, extend `lumen-linux-check`: it already builds the pinned gateway, provisions clean-volume setup, and redacts failure logs. Keep cleanup scoped to resources created by this invocation.
- For native macOS Host against pinned Hermes, extend `lumen-macos-check`: it verifies the source archive SHA-256, starts the Host foreground, checks status, and exercises run/approval/denial/cancellation paths. Reuse its test environment setup rather than copying a second journey script.
- `lumen-external-check` creates a Python compatibility endpoint inside Docker (`scripts/lumen-external-check:78-126`); its evidence is synthetic. Reuse it for repeatable contract/control-flow checks only. Actual cross-machine Runs must use the existing Host adapter against pinned Hermes on the second machine; do not relabel this endpoint as live proof.
- Record missing Linux/update host, second machine, pinned runtime, or containment controls as unverified. Reuse current output/redaction conventions; do not add an evidence framework before a concrete probe demonstrates the need.

## Shared patterns

### Durable intent and recovery

**Sources:** `internal/host/execution.go`, `internal/setup/runner.go`, `cmd/lumen/release.go`  
**Apply to:** setup reruns, ambiguous runtime dispatch, and executable replacement. Persist intent before external effects; after interruption, verify durable state and reconcile under a deadline. Never blindly redispatch ambiguous work.

### Honest operational outcomes

**Sources:** `internal/setup/model.go:5-19`, `internal/setup/doctor.go:42-160`  
**Apply to:** doctor and deployment probes. Emit exactly `ready`, `degraded`, or `action_required`; make absent prerequisites unverified instead of converting synthetic results into live evidence.

### Credential and evidence handling

**Sources:** `scripts/lumen-linux-check:41-48,50-77`, `scripts/lumen-macos-check:37-74`  
**Apply to:** any extended probe. Keep temporary state private, use cleanup traps, redact tokens and keys in failure output, and store credentials by role only in reports.

### Authority boundary

**Sources:** `internal/hermes/client.go:181-199`, `internal/host/execution.go:828-870`  
**Apply to:** all Hermes interactions. Host policy and durable task state remain authoritative; Hermes responses and events are untrusted evidence.

## No analog found

| File or evidence | Role | Data flow | Reason |
|---|---|---|---|
| New physical E2 containment probe, if planning identifies one | probe script | adversarial file/network/tool/memory attempts | Existing profile checks do not prove OS/container isolation; choose the probe from the actually available restricted environment. No probe filename or harness is assumed here. |
| New pinned-runtime E1 runner, if needed | probe script | streaming / recovery | Existing macOS journey covers a pinned gateway, but does not itself qualify every required lost-stream/restart interaction. Extend it if possible; otherwise name a new runner only after the environment and interaction are concrete. |

## Metadata

**Analog search scope:** `internal/host/`, `internal/setup/`, `internal/hermes/`, `cmd/lumen/`, `scripts/`, `test/contract/`  
**Tracked-source check:** every analog path above is tracked by Git.  
**Pattern extraction date:** 2026-09-27
