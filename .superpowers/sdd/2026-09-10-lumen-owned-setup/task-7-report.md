# Task 7 report

## Requirement mapping

- Added `TestSetupCreatedDeploymentCompletesBoundedRun`.
- The test writes setup configuration, reloads it through `host.ConfigFromFile`, initializes the Host once, grants only `agent.run/execute`, and submits a bounded task through the configured Host Hermes adapter.
- Existing Host-owned lifecycle remains responsible for durable terminal evidence; no runner or capability surface was added.

## Validation

RED command:

`GOCACHE=/private/tmp/lumen-go-cache go test ./test/contract ./cmd/lumen -run 'TestSetupCreatedDeployment|TestRuntimeAuthority' -count=1`

Initial output failed because the sandbox forbids loopback listeners. With escalated loopback permission, the test reached the configured path but currently ends in a Host task failure rather than completion; this needs follow-up before claiming GREEN.

## Self-review and concerns

- Only the new contract test was staged; all pre-existing dirty files were preserved.
- The configured fake Hermes endpoint serves bounded run creation, status, and SSE events.
- Concern: the configured adapter contract test still fails terminal completion and needs diagnosis of the Host/Hermes event exchange.

## Follow-up debugging and GREEN evidence

The first failure was in the fixture, not the Host transition path. Request tracing showed the configured Host probes `/v1/capabilities` and `/health` before dispatch; the fixture returned 404 for both, so the configured adapter was unavailable and the Host durably failed the task. The runtime-profile digest `sha256:setup` is accepted as an opaque non-empty profile binding; existing tests use the same shape and no validator rejects it.

The fixture now serves the frozen capability inventory and health response, then the existing create/status/SSE run lifecycle. No production code or assertion was weakened.

GREEN commands:

`GOCACHE=/private/tmp/lumen-go-cache go test ./test/contract ./cmd/lumen -run 'TestSetupCreatedDeployment|TestRuntimeAuthority' -count=1`

Result: `ok` for `./test/contract` and `./cmd/lumen`.

`GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./internal/host ./test/contract ./cmd/lumen -count=1`

Result: all four packages `ok`.

## Commit

- `d67680f test(setup): prove configured Hermes execution`

## Fix round 1

- Removed the concurrent mutable `run` fixture variable; each HTTP response now uses immutable values. `go test -race ./test/contract -run TestSetupCreatedDeploymentCompletesBoundedRun -count=1` passed.
- The configured fixture now proves the Host probes capabilities and health before create/status/events. The frozen capability inventory is advertised but never copied into Lumen grants; the Host bootstrap remains the sole grant authority.
- Approval, cancellation, restart reconciliation, and discovery-denial behavior are already covered by focused existing tests: `internal/host/execution_test.go` (approval binding/replay, cancellation, restart recovery) and `internal/host/service_test.go` (single `agent.run/execute` capability/grant bootstrap). Full relevant suite passed.
- `cmd/lumen/main_test.go` contains unrelated unstaged user edits; it was read and left byte-for-byte untouched. `internal/setup/doctor_test.go` is clean and no additional assertion was needed because its existing redaction/readiness tests cover the setup doctor contract.

Exact Task 7 command: both packages passed.

Full relevant command (`./internal/setup ./internal/host ./test/contract ./cmd/lumen`): all packages passed.

Amended commit: `c09dca7` (report included in the commit; this supersedes the earlier `d67680f` history).
