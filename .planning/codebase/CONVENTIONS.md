---
last_mapped_commit: ebe42cda0332c603644874b73ff920964ba853f1
last_mapped_at: 2026-09-26
---
# Coding Conventions

**Analysis Date:** 2026-09-26

## Naming Patterns

**Files:**

- Production Go files use lowercase snake-like names describing a domain or responsibility, such as `internal/host/execution.go`, `internal/store/store.go`, and `internal/setup/adoption.go`.
- Tests are co-located and named `<source>_test.go`, for example `internal/hermes/client_test.go`.
- Cross-package contract tests are under `test/contract/` and use descriptive scenario names such as `setup_journey_test.go`.

**Functions:**

- Exported Go API names use PascalCase; package-private functions use camelCase, as in `internal/store.Open` and `internal/store.decodeEnvelope`.
- Test names use `Test` plus a behavior statement, for example `TestCreateIntentRecoversToUnknownWithoutRetryAfterRestart` in `internal/host/execution_test.go`.

**Variables:**

- Use Go camelCase for locals and fields. Short names such as `s`, `err`, and `ctx` are common in narrow Go scopes; use domain-specific names for persisted fields and cross-layer values.

**Types:**

- Exported domain and boundary types use PascalCase (`space.State`, `hermes.Adapter`, `host.Service`). Keep package ownership clear rather than adding redundant type prefixes.
- String-backed enums use named types and package constants, as in `internal/space/model.go:3-9,18-37`.

## Code Style

**Formatting:**

- Use `gofmt`; the phase gate checks `gofmt -l` over `cmd`, `internal`, `protocol`, and `test` in `mise.toml:14-20`.
- Go version is pinned to `1.27.1` by `go.mod:1-3` and `mise.toml:1-5`.
- No separate Go lint or formatter configuration was detected.

**Linting:**

- `go vet ./...` is included in the Phase 2 verification task at `mise.toml:14-20`.
- Do not infer that the Phase 2 task has been run from its presence in `mise.toml`.

## Import Organization

**Order:**

1. Standard library imports.
2. Third-party imports, when present.
3. Lumen module imports, typically separated by a blank line.

See `internal/store/store.go:3-18` and `internal/host/execution_test.go:1-18`.

**Path Aliases:**

- No import aliases are used as a general convention. Prefer the package's natural name; qualify ambiguous identifiers explicitly.

## Error Handling

**Patterns:**

- Return errors from boundary and persistence operations; callers check and propagate them rather than hiding a failed state transition (`internal/store/store.go:180-212`).
- Use sentinel errors and `errors.Is` where callers need stable classification, as in `internal/hermes/client_test.go:400-422`.
- Persist uncertain external execution outcomes as `unknown_outcome` rather than claiming success or retrying ambiguously (`internal/host/execution_test.go:186-255`).
- Keep user-facing diagnostics bounded and actionable; do not include raw credentials or private prompt content. Repository security guidance is in `AGENTS.md:36-40` and `docs/THREAT_MODEL.md:87-90`.

## Logging

**Framework:** `log/slog` in Host services (`internal/host/service.go:1-20`).

**Patterns:**

- Log structured event names and bounded metadata. Conversation logging examples are in the PR20 branch's `internal/conversation/service.go:161-200`; that package is not in current `main`.
- Avoid message text, tokens, provider keys, or other private content in ordinary logs (`AGENTS.md:36-40`, `docs/THREAT_MODEL.md:87-90`).

## Comments

**When to Comment:**

- Explain security, recovery, and ordering invariants where they are not obvious from the code, for example the cleanup authority note in `internal/store/store.go:357-365`.
- Keep comments aligned with behavior and cite canonical architecture/security documents rather than duplicating policy (`AGENTS.md:54-58`).

**JSDoc/TSDoc:**

- Not applicable; the project code is Go, Swift, Kotlin, Python, and shell. Go doc comments are used on exported APIs where needed.

## Function Design

**Size:** Keep functions centered on one transition or adapter responsibility. Existing Go service methods may coordinate a full durable flow; factor helpers only when they name a real step, as `internal/host/execution.go` demonstrates.

**Parameters:** Use typed request/config structs at subsystem boundaries; avoid passing parallel positional values for a multi-field product operation (`internal/host/execution.go`, `internal/hermes/client.go`).

**Return Values:** Return concrete results plus errors. Use explicit outcome/status types for accepted, failed, and uncertain operations (`internal/space/model.go:39-63`).

## Module Design

**Exports:** Keep implementation details private to their owning package. `internal/` packages contain Host, store, Space, setup, Hermes, and control logic.

**Barrel Files:** Not used in Go packages; import the owning package directly.

---

*Convention analysis: 2026-09-26*
