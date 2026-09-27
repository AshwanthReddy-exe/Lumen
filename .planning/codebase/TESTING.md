---
last_mapped_commit: ebe42cda0332c603644874b73ff920964ba853f1
last_mapped_at: 2026-09-26
---
# Testing Patterns

**Analysis Date:** 2026-09-26

## Test Framework

**Runner:**

- Go standard-library `testing` package; module version is `go 1.27.1` in `go.mod:1-3`.
- No separate Go test framework or assertion dependency is configured.
- Contract tests use the same runner under `test/contract/`.

**Assertion Library:**

- `testing.T` methods (`Fatal`, `Fatalf`, `Errorf`); use standard-library error inspection such as `errors.Is` when error identity is part of the contract.

**Run Commands:**

```bash
go test ./...                         # Go unit and contract suites
go test -race ./...                   # race-enabled Go suite
go test ./test/contract -run <Pattern> -count=1  # focused contract check
mise run phase0-check                 # KMP and Swift schema baseline
```

`go test`, `go test -race`, and `go vet` are listed in the Phase 2 task in `mise.toml:14-20`. `phase0-check` runs the Kotlin and Swift schema spikes in `mise.toml:7-12`; it is not a Go test command. The commands above describe configured or standard runner usage, not checks run during this mapping.

## Test File Organization

**Location:**

- Package tests are co-located with source files in `cmd/` and `internal/`.
- Cross-package/deployment contract checks live in `test/contract/`.

**Naming:**

- Go test files end in `_test.go`; functions use `Test<Behavior>` names.
- The current `main` tree contains 36 Go test files. This count is a file inventory, not a test execution result.

**Structure:**

```text
internal/<package>/<source>_test.go
cmd/<command>/<behavior>_test.go
test/contract/<journey>_test.go
```

## Test Structure

**Suite Organization:**

```go
func TestReadRejectsMissingWrongKeyAndInvalidEnvelope(t *testing.T) {
    s, statePath, keyPath := testStore(t)
    if err := s.Initialize(testState()); err != nil {
        t.Fatal(err)
    }
    // Corrupt one input at a time and assert Read rejects it.
}
```

Adapted from `internal/store/store_test.go:158-200`; preserve one test's focused behavior and use `t.Run` for named variations where useful.

**Patterns:**

- Create isolated state with `t.TempDir()` and test helpers (`internal/store/store_test.go:32-45`, `internal/host/execution_test.go:115-134`).
- Use table-driven subtests for related variants (`internal/setup/doctor_test.go:99-123`).
- Use `t.Cleanup`/`defer` to close services and servers (`internal/host/execution_test.go:115-134`, `internal/hermes/fake_test.go:26-51`).
- Assert durable state after the operation, not only its immediate return (`internal/host/execution_test.go:141-163`).

## Mocking

**Framework:**

- Hand-written fake adapters and test servers; no mocking framework detected.

**Patterns:**

```go
type fakeRuntime struct {
    create hermes.Run
    createErr error
    events []hermes.Event
}
```

`internal/host/execution_test.go:20-35` contains the Host fake runtime pattern. `internal/hermes/fake_test.go:26-124` builds local HTTP and TLS test servers with test certificates.

**What to Mock:**

- Replace external runtime, clock, filesystem fault points, or operating-system supervisor calls when checking Host decisions in isolation (`internal/host/execution_test.go:115-138`, `internal/store/store_test.go:220-260`, `internal/setup/supervisor_test.go`).
- Use `httptest` for Hermes HTTP/SSE boundary behavior and local fixtures.

**What NOT to Mock:**

- Keep domain transitions, encryption envelope checks, durable recovery behavior, and serialization real in the relevant package tests.
- A fake container/runtime probe is not evidence of a live deployment; record such results as synthetic and separate from owner/device acceptance (`docs/PLAN.md:379-419`).

## Fixtures and Factories

**Test Data:**

```go
func executionService(t *testing.T, runtime hermes.Adapter) *Service {
    t.Helper()
    d := t.TempDir()
    // Initialize a real private store and inject only the Hermes boundary.
    // Return a ready service with cleanup registered.
}
```

Pattern from `internal/host/execution_test.go:115-134`.

**Location:**

- Small package fixtures/helpers stay in `*_test.go`, such as `internal/store/store_test.go` and `internal/hermes/fake_test.go`.
- Shared cross-platform JSON fixtures are under `spikes/o-001/fixtures/` and are exercised by `mise.toml:7-12`.

## Coverage

**Requirements:** No coverage percentage target was detected in `mise.toml` or Go test configuration.

**View Coverage:**

```bash
go test ./... -cover
```

## Test Types

**Unit Tests:**

- Validate package invariants, malformed inputs, policy checks, persistence boundaries, retries, and state transitions. Examples: `internal/store/store_test.go`, `internal/space/execution_test.go`, `internal/host/execution_test.go`.

**Integration Tests:**

- Use real temporary files, local HTTP/TLS servers, command subprocesses, and fake system services. Examples: `internal/hermes/client_test.go`, `cmd/lumen-host/main_test.go`, `cmd/lumen/release_test.go`.

**Contract Tests:**

- `test/contract/` checks deployment scripts, install journeys, release manifests, and cross-package behavior. Some checks require external tools or platform setup; see `mise.toml:38-72`.

**E2E Tests:**

- Live Linux/macOS/Hermes journeys are invoked by scripts listed in `mise.toml:38-72`. Do not count script presence, synthetic provider probes, or prior plan prose as a current live pass. Phase 2 still requires explicit install, restart/reconciliation, and cross-machine evidence (`docs/PLAN.md:402-419`).

## Common Patterns

**Async Testing:**

```go
results := make(chan error, 2)
go func() { _, err := service.SubmitTask(context.Background(), req); results <- err }()
for i := 0; i < 2; i++ {
    if err := <-results; err != nil { t.Fatal(err) }
}
```

Adapted from `internal/host/execution_test.go:276-290`. Ensure goroutines are bounded and report through the test; use explicit channels, contexts, and deadlines for concurrency behavior.

**Error Testing:**

```go
if _, err := client.Health(context.Background()); !errors.Is(err, ErrRedirectRejected) {
    t.Fatalf("expected redirect rejection, got %v", err)
}
```

Pattern from `internal/hermes/client_test.go:408-422`. Prefer asserting error class and durable outcome over matching unstable full error text.

---

*Testing analysis: 2026-09-26*
