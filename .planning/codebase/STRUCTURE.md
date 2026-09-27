---
last_mapped_commit: ebe42cda0332c603644874b73ff920964ba853f1
last_mapped_at: 2026-09-26
---
# Codebase Structure

**Analysis Date:** 2026-09-26

## Directory Layout

```text
lumen/
├── apps/android-host/       # Quarantined Android companion UI
├── cmd/lumen/               # Owner-facing CLI
├── cmd/lumen-host/          # Headless Go Host entry point
├── deploy/                  # Docker, launchd, systemd, Termux service files
├── docs/                    # Canonical product, architecture, decisions, plan
├── internal/control/        # Credentialed local Unix socket
├── internal/hermes/         # Hermes HTTP/runtime adapter
├── internal/host/           # Host composition, lifecycle, execution
├── internal/setup/          # Platform-neutral setup/adoption workflow
├── internal/space/          # Pure Space state and policy transitions
├── internal/store/          # Encrypted durable state snapshot
├── protocol/                # Shared protocol fixtures
├── scripts/                 # Platform lifecycle/release checks and helpers
├── spikes/                  # Historical stack/protocol experiments
└── test/contract/           # Cross-package and deployment contract tests
```

## Directory Purposes

**`internal/space/`:**

- Purpose: Canonical domain model and authorization/state transitions.
- Contains: State and command records, grants, tasks, approvals, audit and pure transition rules.
- Key files: `internal/space/model.go`, `internal/space/apply.go`, `internal/space/policy.go`, `internal/space/execution.go`.

**`internal/host/`:**

- Purpose: Compose domain, storage, control socket, runtime adapter, and lifecycle.
- Contains: Config, initialization, service lifecycle, execution and recovery.
- Key files: `internal/host/service.go`, `internal/host/execution.go`.

**`internal/store/`:**

- Purpose: Persist the canonical Space snapshot under process and in-process locks.
- Contains: AES-GCM envelope, key/state file access, commits, state validation and fixtures.
- Key files: `internal/store/store.go`, `internal/store/envelope.go`, `internal/store/testdata/state-v1.json`.

**`internal/control/`:**

- Purpose: Local owner control protocol and credential handling.
- Contains: Unix server/client, wire framing, credentials, tests.
- Key files: `internal/control/server.go`, `internal/control/client.go`, `internal/control/protocol.go`.

**`internal/hermes/`:**

- Purpose: Hermes API contract and runtime adapter.
- Contains: HTTP/TLS client, capabilities, runs, SSE events, approvals and test fake.
- Key files: `internal/hermes/client.go`, `internal/hermes/events.go`, `internal/hermes/fake_test.go`.

**`internal/setup/`:**

- Purpose: Reusable platform-neutral setup, adoption, supervision, and recovery workflow.
- Contains: Models, planner, runner, journal, artifact/release checks, config, supervisor interfaces and platform probes.
- Key files: `internal/setup/model.go`, `internal/setup/planner.go`, `internal/setup/runner.go`, `internal/setup/journal.go`, `internal/setup/adoption.go`.

**`cmd/`:**

- Purpose: Executable boundaries.
- Contains: Owner CLI and internal Host daemon commands.
- Key files: `cmd/lumen/`, `cmd/lumen-host/main.go`.

**`deploy/` and `scripts/`:**

- Purpose: Adapt product-neutral commands to supported OS service managers and release workflows.
- Contains: Docker Compose/Dockerfiles, launchd/systemd units, Termux/runit scripts and lifecycle checks.
- Key files: `deploy/docker/compose.yaml`, `deploy/launchd/dev.lumen.host.plist`, `deploy/systemd/lumen-host.service`, `deploy/termux/`, `scripts/lumen-macos-check`, `scripts/lumen-linux-check`.

**`apps/android-host/`:**

- Purpose: Quarantined companion shell, not the canonical Space Host.
- Contains: Android manifest, Compose activity/theme, screen model, and UI tests.
- Key files: `apps/android-host/src/main/kotlin/dev/lumen/android/host/LumenHostActivity.kt`, `CompanionScreenModel.kt`.

**`protocol/`, `spikes/`, and `test/contract/`:**

- Purpose: Cross-language fixtures, historical experiments, and system/deployment contracts.
- Contains: JSON protocol fixture, isolated spike projects, Go contract tests.
- Key files: `protocol/fixtures/space-v1.json`, `spikes/o-001/`, `test/contract/`.

## Key File Locations

**Entry Points:**

- `cmd/lumen/main.go`: Owner-facing setup and lifecycle CLI.
- `cmd/lumen-host/main.go`: Headless Host `doctor`, `init`, `serve`, local status/task/approval commands (`cmd/lumen-host/main.go:20-69`).

**Configuration:**

- `mise.toml`: Project command/task configuration.
- `go.mod`: Go module and dependency declarations.
- `internal/host/service.go`: Host runtime config validation/loading (`internal/host/service.go:38-125`).
- `deploy/`: Supervisor and service configuration.

**Core Logic:**

- `internal/space/`: Put product authority and pure state rules here.
- `internal/host/`: Put Host orchestration here when it coordinates domain state and adapters.
- `internal/store/`: Put durable-state mechanics here, not product policy.
- `internal/control/` and `internal/hermes/`: Keep local operator protocol and Hermes API mechanics at these adapter boundaries.
- `internal/setup/`: Keep portable setup decisions here; platform mutations belong in `deploy/` or narrowly scoped `scripts/`.

**Testing:**

- `internal/*/*_test.go`: Package unit/integration tests alongside implementation.
- `apps/android-host/src/test/` and `src/androidTest/`: Android model and UI tests.
- `test/contract/`: Cross-package, deployment, release and setup journey contracts.
- `protocol/fixtures/` and `internal/store/testdata/`: Shared and persisted-state fixtures.

## Naming Conventions

**Files:**

- Go source and tests use lowercase package-oriented names, with `_test.go` suffix for tests (for example, `internal/host/execution.go`, `internal/host/execution_test.go`).
- Deployment files name the managed service or profile (for example, `deploy/systemd/lumen-host.service`).
- Documentation uses uppercase canonical names in `docs/` and descriptive Markdown names in subfolders.

**Directories:**

- Go packages use lowercase names grouped by responsibility under `internal/`.
- Executable packages live under `cmd/<binary>/`.
- Platform-specific deploy files are grouped by supervisor/platform under `deploy/`.

## Where to Add New Code

**New Space rule or authority state:**

- Primary code: `internal/space/model.go` and `internal/space/apply.go`; use existing policy helpers in `internal/space/policy.go`.
- Tests: `internal/space/*_test.go`; add a `test/contract/` case when behavior crosses package or deployment boundaries.
- Keep state transitions deterministic and independent of I/O.

**New Host orchestration:**

- Implementation: `internal/host/`.
- Tests: corresponding `internal/host/*_test.go`.
- Route durable changes through `store.Store.Update` and domain transitions.

**New persistence behavior:**

- Implementation: `internal/store/`.
- Tests and fixtures: `internal/store/store_test.go`, `internal/store/testdata/`.
- Do not add storage-specific concepts to `internal/space`.

**New runtime API operation:**

- Contract and client: `internal/hermes/client.go` or `internal/hermes/events.go` according to the existing API area.
- Orchestration and authorization remain in `internal/host/`; add fake/runtime tests in the respective packages.

**New setup platform behavior:**

- Shared planning and lifecycle contract: `internal/setup/`.
- OS command/config adapter: matching subdirectory under `deploy/` or existing platform script under `scripts/`.
- Cross-platform acceptance: `test/contract/`.

**New companion UI:**

- Android-only presentation: `apps/android-host/src/main/kotlin/dev/lumen/android/host/`.
- Do not place canonical Host state or Hermes authority in the Android app.

## Special Directories

**`spikes/`:**

- Purpose: Isolated experiments and their fixtures.
- Generated: No general generated output detected.
- Committed: Yes; treat as historical/experimental and do not use as production authority without an explicit plan decision.

**`deploy/docker/data/`:**

- Purpose: Empty tracked mount placeholder for local deployment.
- Generated: Runtime data is generated when deploying.
- Committed: `.keep` only; runtime state and credentials must remain untracked.

**PR #20 branch-only paths (not present in current checkout):**

- `internal/conversation/`: Canonical conversation orchestration and projection (`feat/lumen-continuity` worktree, commit `0b67f81`).
- `internal/node/`: Signed read-only node status request, receipt, and dispatch.
- `integrations/hermes/`: Isolated Hermes chat config, probes, and patch.
- `scripts/lumen-mac-chat-check`: Mac real-chat check.
- These branch paths do not exist on current `main`; the branch worktree also has local uncommitted edits, including changes under `internal/control/` and `internal/host/`. Check the PR diff and worktree status separately before treating those edits as merged behavior.

---

*Structure analysis: 2026-09-26*
