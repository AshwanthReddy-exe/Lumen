# Milestone 1 Foundation Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Qualify one combined Linux/VPS Lumen and Hermes deployment through the complete Milestone 1 journey without manual environment plumbing.

**Architecture:** Reconcile and compose the existing Go setup, Host, store, Hermes Runs adapter, and supervisor primitives. The public `lumen` CLI owns lifecycle orchestration while the Host remains canonical authority and Hermes remains an untrusted execution adapter. Linux/VPS is the blocking reference profile; other platforms qualify separately.

**Tech Stack:** Go 1.27.1 standard library, existing encrypted Host/store, Hermes HTTP Runs adapter, JSON manifest/journal/configuration, systemd or Docker Compose reference supervision, existing `mise` gates.

**Spec:** `docs/PLAN.md` Milestone 1, `docs/PHASE-2-HOST-HERMES.md`, and `docs/superpowers/specs/2026-09-11-milestone-driven-development-design.md`

## Global constraints

- Milestone 1 is the only active implementation milestone.
- Host owns canonical Space state, authorization, approvals, task outcomes, and audit.
- Hermes is reused through the narrow authenticated Runs adapter and cannot create grants or authority.
- Report only `ready`, `degraded`, or `action_required`; never expose secrets or raw untrusted errors.
- Never replace initialized Space identity, silently rotate credentials, weaken a requested profile, or retry an uncertain effect without durable evidence.
- Use existing primitives and the Go standard library; add no second runner, state store, installer framework, or generic adapter layer.
- One implementation agent mutates the worktree at a time. Tasks execute strictly in dependency order.
- Combined Linux/VPS evidence blocks exit. macOS, Termux, separated, and hardened claims retain their own explicit gates.

---

## SDD preflight

Before Task 1, the coordinator runs:

```bash
/Users/ashwanthreddyboddireddy/.codex/plugins/cache/openai-curated-remote/superpowers/6.3.0/skills/subagent-driven-development/scripts/sdd-workspace docs/superpowers/plans/2026-09-11-milestone-1-foundation-closure.md
git rev-parse HEAD
git status --short --branch
```

Create the printed `progress.md` with `# SDD ledger — plan: docs/superpowers/plans/2026-09-11-milestone-1-foundation-closure.md` as its first line. Record the milestone base, dirty-state ruling, baseline command, one internal-consistency row for every task, and one producer/consumer row for every pair sharing a file or interface. Resolve every conflict in the ledger before extracting the Task 1 brief.

### Task 1: Reconcile the committed setup foundation

**Files:**
- Modify: `docs/PHASE-2-HOST-HERMES.md`
- Test: existing `internal/setup`, `cmd/lumen`, and contract packages

**Interfaces:**
- Consumes: committed setup work through current `HEAD` and historical setup evidence.
- Produces: an evidence-backed baseline matrix mapping every Milestone 1 deliverable to implemented, defective, or missing behavior.

- [ ] **Step 1: Run `rtk go test ./internal/setup ./cmd/lumen ./cmd/lumen-host ./internal/host ./internal/hermes ./test/contract -count=1`; record each failing test and its owning deliverable, or record the passing package list.**
- [ ] **Step 2: Inspect `cmd/lumen/main.go`, `internal/setup/{runner,doctor,manifest,supervisor}.go`, `deploy/manifest-v1.json`, and `test/contract/setup_hermes_run_test.go`; record deliverables 1–6 as `implemented`, `defective`, or `missing` with file evidence.**
- [ ] **Step 3: Update only the implementation checkpoint in `docs/PHASE-2-HOST-HERMES.md` to match committed evidence; retain fixture-manifest, clean-install, reboot, rollback, and external-runtime gaps.**
- [ ] **Step 4: Run `rtk go test ./... -count=1` and `rtk git diff --check`; both must pass.**
- [ ] **Step 5: Commit with `docs(plan): reconcile milestone one baseline`.**

### Task 2: Pin the Hermes and Lumen distribution inputs

**Files:**
- Create: `deploy/hermes-source.lock.json`
- Create: `deploy/docker/Dockerfile.hermes`
- Modify: `deploy/manifest-v1.json`
- Modify: `deploy/docker/compose.yaml`
- Modify: `internal/setup/manifest.go`
- Modify: `internal/setup/manifest_test.go`
- Modify: `internal/setup/artifacts.go`
- Modify: `internal/setup/artifacts_test.go`

**Interfaces:**
- Consumes: Hermes Agent tag `v2026.9.7`, peeled commit `2237be355906fbe6065ce1815711eee52b2d646e`, the official `uv.lock`, and Lumen commit `348cd3e` as the initial compatibility candidates.
- Produces: immutable source identity, locally reproducible build inputs, strict manifest records, and verified staged replacement/rollback behavior. Passing this task certifies artifact integrity only; Task 4 certifies Hermes behavior.

- [ ] **Step 1: Add failing tests `TestManifestRejectsMutableHermesSource`, `TestManifestRejectsPlaceholderDigest`, and `TestFailedReplacementPreservesInstalledArtifact`; run `rtk go test ./internal/setup -run 'TestManifestRejects|TestFailedReplacement' -count=1` and expect rejection paths to be missing.**
- [ ] **Step 2: Record schema version, official repository, tag, peeled commit, Python `3.11`, and lockfile requirement in `deploy/hermes-source.lock.json`; fetch/build logic must verify the peeled commit and use `uv sync --frozen`.**
- [ ] **Step 3: Build the Lumen binary twice with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w'`; build Hermes from the pinned source twice with `deploy/docker/Dockerfile.hermes`; record computed sizes and SHA-256 identities in the manifest/Compose inputs rather than hand-written placeholders.**
- [ ] **Step 4: Run `rtk go test ./internal/setup -count=1`, two-build digest comparisons, `docker compose -f deploy/docker/compose.yaml config`, and the staged update/rollback tests; all must pass.**
- [ ] **Step 5: Commit with `build(release): pin milestone one artifacts`.**

### Task 3: Complete the public combined setup journey

**Files:**
- Modify: `cmd/lumen/main.go`
- Modify: `cmd/lumen/main_test.go`
- Modify: `internal/setup/runner.go`
- Modify: `internal/setup/runner_test.go`
- Modify: `internal/setup/doctor.go`
- Modify: `internal/setup/doctor_test.go`
- Modify only if a demonstrated defect requires it: `internal/setup/config.go`, `internal/setup/supervisor.go`, and their tests

**Interfaces:**
- Consumes: planner, journal, verified artifact, configuration, Host initialization, and supervisor primitives.
- Produces: truthful `lumen setup`, `lumen doctor`, and `lumen service start|stop|restart|status` for the combined Linux/VPS reference journey.

- [ ] **Step 1: Add failing tests named `TestSetupCommandRunsCompletePlan`, `TestSetupRerunPreservesIdentityAndCredentials`, `TestSetupResumesAfterHostInitialization`, `TestDoctorReportsReadyDegradedAndActionRequired`, and `TestServiceCommandsUseSelectedSupervisor`.**
- [ ] **Step 2: Run `rtk go test ./cmd/lumen ./internal/setup -run 'TestSetupCommand|TestSetupRerun|TestSetupResumes|TestDoctorReports|TestServiceCommands' -count=1`; expect failures at the first missing orchestration behavior.**
- [ ] **Step 3: In `setupCommand`, construct `cliProbe`, call `setup.Plan`, open the selected setup journal, create one `setupState`, and pass its `runStage`/`verifyStage` callbacks plus `HostInitializerFuncs` into one `setup.SetupRunner`. Make `doctorCommand` pass `setupState.observe` into `setup.Doctor`, and make `serviceCommand` call only the supervisor selected by `cliProbe`. `SetupRunner.validateJournalBinding` must reject a changed profile or plan digest before mutation; `runStage` must implement artifact verification/adoption, `WriteConfig`, create-once Host initialization, and `InstallServices` exactly once; `verifyStage` must observe the completed invariant before journal recording.**
- [ ] **Step 4: Run `rtk go test ./cmd/lumen ./internal/setup ./cmd/lumen-host -count=1`, `rtk go test ./... -count=1`, and `rtk go test -race ./cmd/lumen ./internal/setup`; all must pass.**
- [ ] **Step 5: Commit with `feat(setup): complete combined setup journey`.**

### Task 4: Certify the configured Hermes Runs boundary

**Files:**
- Modify: `test/contract/setup_hermes_run_test.go`
- Modify: `internal/setup/doctor_test.go`
- Modify only when the test exposes a defect: `internal/hermes/*` or `internal/host/*`

**Interfaces:**
- Consumes: Task 3 deployment output, Task 2 compatibility pin, and the existing Host-owned `agent.run/execute` lifecycle. Hermetic tests may use a fake server; release claims require the pinned runtime probe in Task 6.
- Produces: configured capability, authentication, run, event, approval, cancellation, event-loss, and restart-recovery evidence.

- [ ] **Step 1: Extend `TestSetupCreatedDeploymentCompletesBoundedRun` with subtests `approval_once`, `cancel`, `duplicate_reordered_events`, `lost_stream`, and `host_restart`; each asserts the durable Host outcome and that discovery never creates a grant.**
- [ ] **Step 2: Run `rtk go test ./test/contract -run TestSetupCreatedDeploymentCompletesBoundedRun -count=1`; expect the first unimplemented boundary to fail without requiring network access.**
- [ ] **Step 3: Fix only the exposed Host/Hermes boundary defects; do not create a second execution path or enable discovered capabilities.**
- [ ] **Step 4: Run `rtk go test ./test/contract ./internal/host ./internal/hermes -count=1`, `rtk go test ./... -count=1`, and `rtk go test -race ./internal/host ./internal/hermes`; all must pass.**
- [ ] **Step 5: Commit with `test(setup): certify configured Hermes runs`.**

### Task 5: Add the automated combined Linux journey gate

**Files:**
- Create: `scripts/lumen-linux-check`
- Create: `test/contract/setup_journey_test.go`
- Modify: `deploy/docker/compose.yaml`
- Modify: `mise.toml`

**Interfaces:**
- Consumes: Tasks 2–4.
- Produces: `mise run milestone1-linux-check`, a hermetic Docker Compose journey proving clean setup, rerun, interruption recovery, truthful doctor, service restart, configured synthetic task, failed update, rollback, and state preservation without host reboot claims.

- [ ] **Step 1: Add `TestLinuxSetupJourneyScriptContract` asserting the script performs clean-volume setup, rerun, forced interruption, doctor, restart, task, failed update, rollback, and final state comparison; run `rtk go test ./test/contract -run TestLinuxSetupJourneyScriptContract -count=1` and expect RED.**
- [ ] **Step 2: Implement `scripts/lumen-linux-check` with `set -eu`, an isolated Compose project name, synthetic identities/content, bounded waits, traps that preserve failure logs but remove test containers, and no secret values in output.**
- [ ] **Step 3: Add `milestone1-linux-check` to `mise.toml` containing only Go, Linux artifact, Docker Compose, setup journey, secret scan, and deployment checks; leave Android, launchd, and physical Termux checks in `phase2-check`.**
- [ ] **Step 4: Run `rtk mise run milestone1-linux-check` twice from clean volumes; both runs must pass and emit redacted evidence metadata.**
- [ ] **Step 5: Commit with `test(deploy): add combined Linux journey gate`.**

### Task 6: Record the reboot and pinned-runtime owner proof

**Files:**
- Modify: `deploy/README.md`
- Modify: evidence section in `docs/PHASE-2-HOST-HERMES.md`

**Interfaces:**
- Consumes: Tasks 2–5.
- Produces: dated evidence for Ubuntu 24.04 LTS amd64 with Docker Engine and Compose, the combined topology, pinned Hermes `v2026.9.7`/`2237be355906fbe6065ce1815711eee52b2d646e`, reboot survival, and real Runs behavior.

- [ ] **Step 1: Provision or identify a clean Ubuntu 24.04 LTS amd64 VPS with Docker Engine boot-enabled and Compose available; record versions, topology, commit, profile, and preconditions without credentials.**
- [ ] **Step 2: Run `mise run milestone1-linux-check`, then execute the configured pinned-runtime completion, exact `once`/`deny` approval, cancellation, lost-stream, and Host-restart probes; record expected and actual outcomes.**
- [ ] **Step 3: Reboot the VPS, verify both services and `lumen doctor`, compare canonical Space identity/state, then exercise failed update and rollback without manual state repair.**
- [ ] **Step 4: Record a redacted evidence reference and explicitly mark every unrun or failed check open; never substitute a local process restart for reboot.**
- [ ] **Step 5: Commit with `docs(evidence): record combined foundation proof`.**

### Task 7: Close Milestone 1

**Files:**
- Modify: `docs/PLAN.md`
- Modify: `docs/PHASE-2-HOST-HERMES.md`
- Modify: `docs/CHANGELOG.md`
- Modify: `README.md` and `deploy/README.md` only where owner instructions changed

**Interfaces:**
- Consumes: all task reviews and automated/owner evidence.
- Produces: one honest Milestone 1 exit decision and the entry condition for Milestone 2.

- [ ] **Step 1: Read `BASE` from the ledger and run `/Users/ashwanthreddyboddireddy/.codex/plugins/cache/openai-curated-remote/superpowers/6.3.0/skills/subagent-driven-development/scripts/review-package docs/superpowers/plans/2026-09-11-milestone-1-foundation-closure.md "$BASE" HEAD`; dispatch an independent architecture, authority, security, recovery, and over-engineering reviewer with the printed package path.**
- [ ] **Step 2: For any Critical or Important finding, dispatch one bounded fix wave, regenerate a package for only that fix range, and dispatch one scoped re-review. Record every finding, fix commit, verdict, and residual ruling in the ledger; no unresolved Critical or Important finding may pass the gate.**
- [ ] **Step 3: Run `rtk mise run milestone1-linux-check`, `rtk go test ./... -count=1`, `rtk go test -race ./...`, `rtk git diff --check`, and `rtk graphify update .`; capture fresh output. Run `phase2-check` separately when its Android/macOS prerequisites are available.**
- [ ] **Step 4: Update canonical status and changelog only if the combined Linux/VPS exit gate passed; list macOS/Termux/separated/hardened evidence independently.**
- [ ] **Step 5: Commit with `docs(milestone): close reliable combined foundation`.**
