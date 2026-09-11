# Lumen-Owned Setup Implementation Plan

> **Status:** Superseded historical task record. Completed tasks remain implementation evidence, but no task in this file is active. Remaining foundation work moved to the [Milestone 1 foundation-closure plan](./2026-09-11-milestone-1-foundation-closure.md).

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `lumen setup` as the one-command, resumable installation and configuration journey for a supervised Lumen Host and compatible Hermes runtime.

**Architecture:** A new public `cmd/lumen` binary calls a platform-independent `internal/setup` planner and journal. Narrow platform adapters perform artifact, configuration, and supervisor operations; the existing `lumen-host` binary remains the create-once authority and foreground service. Setup evidence and `doctor` are redacted views of observed state, never new Space authority.

**Tech Stack:** Go 1.27.1 standard library; JSON setup journal and manifest; existing encrypted Host/store, Hermes HTTP adapter, launchd, systemd, Docker Compose, Termux/runit, and contract-test conventions.

**Spec:** `docs/superpowers/specs/2026-09-10-lumen-owned-setup-design.md`

## Global constraints

- Lumen owns setup, public commands, policy, approvals, audit, and lifecycle; Hermes remains an untrusted replaceable adapter.
- `lumen setup` never reinitializes an existing Space, silently rotates identity/credentials, or weakens the requested profile.
- Secrets enter through protected files, file descriptors, credential stores, or hidden input; never flags, prompts, ordinary logs, or status JSON.
- Setup reports exactly `ready`, `degraded`, or `action_required` and includes only redacted next actions.
- Development loopback uses synthetic state; hardened deployments require enforceable Hermes isolation and pinned mutual TLS.
- Block 2 reserves `lumen connect` but does not trust external nodes or implement QR/manual pairing.
- Use standard library and existing platform facilities; add no installer framework or second state store.
- Each task follows red-green-refactor and preserves unrelated dirty worktree changes.

---

### Task 1: Freeze the public setup contract

**Files:**
- Create: `cmd/lumen/main.go`
- Create: `cmd/lumen/main_test.go`
- Create: `internal/setup/model.go`
- Create: `internal/setup/model_test.go`

**Interfaces:**
- Produce `setup.Stage`, `setup.Outcome`, `setup.Request`, `setup.Report`, and `setup.Action`.
- Public commands: `lumen setup`, `lumen doctor`, `lumen service start|stop|restart|status`, and reserved `lumen connect`.
- `lumen connect` returns `action_required` with `availableInBlock: 3`; it performs no pairing mutation.

- [x] **Step 1: Write failing model and CLI tests**

```go
func TestOutcomeValuesAreStable(t *testing.T) {
	for _, got := range []setup.Outcome{setup.Ready, setup.Degraded, setup.ActionRequired} {
		if err := got.Validate(); err != nil { t.Fatalf("%q: %v", got, err) }
	}
}

func TestConnectIsReservedWithoutPairing(t *testing.T) {
	report := runForTest([]string{"connect"})
	if report.Outcome != setup.ActionRequired || report.AvailableInBlock != 3 {
		t.Fatalf("unexpected report: %#v", report)
	}
}
```

- [x] **Step 2: Run tests and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./cmd/lumen -run 'TestOutcomeValues|TestConnectIsReserved' -count=1`

Expected: FAIL because the packages and public command do not exist.

- [x] **Step 3: Implement the minimum typed contract and strict command parser**

```go
type Outcome string
const (
	Ready Outcome = "ready"
	Degraded Outcome = "degraded"
	ActionRequired Outcome = "action_required"
)

type Report struct {
	Outcome Outcome `json:"outcome"`
	Stage Stage `json:"stage"`
	Actions []Action `json:"actions,omitempty"`
	AvailableInBlock int `json:"availableInBlock,omitempty"`
}
```

- [x] **Step 4: Run focused and package tests**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./cmd/lumen -count=1`

Expected: PASS; output contains no environment values or credentials.

- [x] **Step 5: Complete the contract change**

```bash
git add cmd/lumen internal/setup
git commit -m "feat(setup): define public Lumen setup contract"
```

### Task 2: Add deterministic planning and resumable evidence

**Files:**
- Create: `internal/setup/planner.go`
- Create: `internal/setup/planner_test.go`
- Create: `internal/setup/journal.go`
- Create: `internal/setup/journal_test.go`
- Modify: `internal/setup/model.go`

**Interfaces:**
- `Plan(context.Context, Request, Probe) (PlanResult, error)` performs no mutation.
- `Journal.Record(StageEvidence) error` atomically persists completed stage evidence beneath the private setup directory.
- Stages: `detected`, `artifacts_ready`, `directories_ready`, `credentials_ready`, `configuration_ready`, `host_initialized`, `services_installed`, `services_started`, `validated`.

- [x] **Step 1: Write failing planner fixtures**

```go
func TestHardenedNeverDowngrades(t *testing.T) {
	_, err := Plan(context.Background(), Request{Profile: Hardened}, fakeProbe{isolation: false})
	if !errors.Is(err, ErrIsolationUnavailable) { t.Fatalf("got %v", err) }
}

func TestCompletedEvidenceResumesAtNextStage(t *testing.T) {
	j := newTestJournal(t)
	mustRecord(t, j, StageEvidence{Stage: Detected, InputDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if got := j.Next(); got != ArtifactsReady { t.Fatalf("got %q", got) }
}
```

- [x] **Step 2: Run tests and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup -run 'TestHardenedNeverDowngrades|TestCompletedEvidence' -count=1`

Expected: FAIL because planner and journal are undefined.

- [x] **Step 3: Implement planner and atomic journal**

Use `runtime.GOOS`, `runtime.GOARCH`, explicit Termux `PREFIX` detection, `os.CreateTemp`, `File.Sync`, `os.Rename`, and parent-directory sync. Evidence stores digests and versions, never secret content.

```go
type StageEvidence struct {
	Stage Stage `json:"stage"`
	InputDigest string `json:"inputDigest"`
	CompletedAt int64 `json:"completedAt"`
}
```

- [x] **Step 4: Add interruption, changed-input, permissions, truncation, and rerun tests**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup -count=1`

Expected: PASS; a changed prior-stage digest returns `action_required` without mutation.

- [x] **Step 5: Complete planner and journal change**

```bash
git add internal/setup
git commit -m "feat(setup): plan and resume installation"
```

### Task 3: Verify and install compatible Lumen and Hermes artifacts

**Files:**
- Create: `deploy/manifest-v1.json`
- Create: `internal/setup/manifest.go`
- Create: `internal/setup/manifest_test.go`
- Create: `internal/setup/artifacts.go`
- Create: `internal/setup/artifacts_test.go`
- Create: `internal/setup/download.go`
- Create: `internal/setup/download_test.go`
- Create: `internal/setup/adoption.go`
- Create: `internal/setup/adoption_test.go`
- Modify: `internal/setup/model.go`

**Interfaces:**
- `LoadManifest(io.Reader) (Manifest, error)` strictly decodes a versioned compatibility manifest.
- `Installer.Stage(context.Context, Artifact, io.Reader) (StagedArtifact, error)` verifies size and SHA-256 before replacement.
- Existing Hermes may be adopted only when version, executable ownership, endpoint identity, authentication, and advertised Runs capability pass.

- [x] **Step 1: Write failing manifest and preservation tests**

```go
func TestChecksumFailurePreservesInstalledArtifact(t *testing.T) {
	path := installExisting(t, "old")
	err := installerFor(path).Install(context.Background(), artifact("sha256:wrong"), strings.NewReader("new"))
	if !errors.Is(err, ErrDigestMismatch) { t.Fatalf("got %v", err) }
	if got := readFile(t, path); got != "old" { t.Fatalf("replaced with %q", got) }
}
```

- [x] **Step 2: Run test and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup -run 'TestChecksumFailure|TestManifest' -count=1`

Expected: FAIL because artifact support is undefined.

- [x] **Step 3: Implement strict manifest parsing and staged replacement**

The manifest names exact OS/architecture/profile combinations, Lumen and Hermes versions, source URL, byte size, SHA-256, and compatibility contract version. Download through an injected `io.Reader`/HTTP client with bounded size; validate before rename.

- [x] **Step 4: Test adoption, unsupported targets, interrupted update, and rollback**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup -run 'TestArtifact|TestManifest|TestAdopt' -count=1`

Expected: PASS; unsupported targets fail before creating installation paths.

- [x] **Step 5: Complete artifact lifecycle change**

```bash
git add deploy/manifest-v1.json internal/setup
git commit -m "feat(setup): verify Lumen and Hermes artifacts"
```

### Task 4: Create private directories, credentials, and Lumen/Hermes configuration

**Checkpoint:** Implemented and task-reviewed through commit `b671c71`.

**Files:**
- Create: `internal/setup/config.go`
- Create: `internal/setup/config_test.go`
- Modify: `internal/host/service.go`
- Modify: `internal/host/service_test.go`

**Interfaces:**
- `WriteConfig(ConfigRequest) (ConfigPaths, error)` writes complete owner-only files with no unresolved placeholders.
- `host.ConfigFromFile(path string) (host.Config, error)` strictly loads generated non-secret settings; existing environment loading remains a compatibility path.
- Directory creation records `directories_ready` before credentials are created; credential creation records `credentials_ready` before configuration is generated.
- Secrets remain separate `0600` regular files owned by the service principal.

- [ ] **Step 1: Write failing configuration tests**

```go
func TestGeneratedConfigHasNoManualPlaceholders(t *testing.T) {
	paths, err := WriteConfig(validConfigRequest(t))
	if err != nil { t.Fatal(err) }
	for _, body := range readGenerated(t, paths) {
		if strings.Contains(body, "YOUR_") { t.Fatalf("placeholder in %q", body) }
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./internal/host -run 'TestGeneratedConfig|TestConfigFromFile' -count=1`

Expected: FAIL because typed configuration generation is missing.

- [ ] **Step 3: Implement strict config generation and loading**

Generate a stable JSON config for Lumen plus the pinned Hermes version's supported config format. Reject symlinks, permissive modes, unknown fields, relative secret paths, loopback in hardened mode, and missing TLS identity.

- [ ] **Step 4: Test development, personal-alpha, hardened, and Termux boundaries**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./internal/host -count=1`

Expected: PASS; same-UID Termux cannot produce a hardened co-located report.

- [ ] **Step 5: Commit configuration generation**

```bash
git add internal/setup internal/host
git commit -m "feat(setup): generate Host and Hermes configuration"
```

### Task 5: Initialize the Host, then install and control platform supervisors

**Checkpoint:** Implementation commits exist through `a8f54bf`, but the five-round task review breaker left concrete create-once integration, lifecycle/boot coverage, and mandatory manifest-bound Termux digests to be closed before acceptance. Task 6 was not started.

**Files:**
- Create: `internal/setup/supervisor.go`
- Create: `internal/setup/supervisor_test.go`
- Modify: `deploy/systemd/lumen-host.service`
- Create: `deploy/systemd/lumen-hermes.service`
- Modify: `deploy/launchd/dev.lumen.host.plist`
- Create: `deploy/launchd/dev.lumen.hermes.plist`
- Modify: `deploy/docker/compose.yaml`
- Modify: `deploy/termux/install-lumen-host`
- Create: `deploy/termux/hermes-run`
- Modify: `test/contract/supervisor_test.go`

**Interfaces:**
- `Supervisor.Install(context.Context, ServicePlan) error`
- `Supervisor.Enable(context.Context, []ServiceName) error`
- `Supervisor.Control(context.Context, Action, []ServiceName) ([]ServiceState, error)`
- Native implementations invoke only allowlisted fixed commands with explicit arguments; no shell interpolation or secret flags.
- The existing create-once Host initialization runs and verifies durable state before supervisor installation; reruns never replace Host identity.

- [ ] **Step 1: Write failing fake-supervisor and definition tests**

```go
func TestInstallEnablesHermesAndHostInOrder(t *testing.T) {
	f := &fakeSupervisor{}
	err := InstallServices(context.Background(), f, validServicePlan())
	if err != nil { t.Fatal(err) }
	want := []string{"install:hermes", "install:host", "enable:hermes", "enable:host"}
	if !slices.Equal(f.calls, want) { t.Fatalf("got %#v", f.calls) }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./test/contract -run 'TestInstallEnables|TestSupervisorDefinitions' -count=1`

Expected: FAIL because Hermes supervisors and installer integration are absent.

- [ ] **Step 3: Implement the minimal supervisor adapters and definitions**

Use `launchctl`, `systemctl`, Docker Compose, and `sv`/Termux:Boot directly. Host may start degraded while Hermes recovers; dependency failure must not make Space state unavailable.

- [ ] **Step 4: Test install, rerun, bounded restart, graceful stop, missing manager, and boot definition**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./test/contract -run 'TestSupervisor|TestInstall' -count=1`

Expected: PASS; missing boot support returns `action_required`.

- [ ] **Step 5: Commit supervision**

```bash
git add internal/setup deploy test/contract/supervisor_test.go
git commit -m "feat(setup): supervise Host and Hermes"
```

### Task 6: Orchestrate setup and whole-deployment doctor

**Files:**
- Create: `internal/setup/runner.go`
- Create: `internal/setup/runner_test.go`
- Create: `internal/setup/doctor.go`
- Create: `internal/setup/doctor_test.go`
- Modify: `cmd/lumen/main.go`
- Modify: `cmd/lumen/main_test.go`
- Modify: `cmd/lumen-host/main.go`

**Interfaces:**
- `Runner.Run(context.Context, Request) (Report, error)` executes each planned stage once and records evidence after verification.
- `Doctor.Check(context.Context) Report` observes artifacts, Host state, Hermes health/capabilities, supervisor/boot state, credential readiness, and isolation.
- Public JSON includes versions and states but no paths containing usernames, tokens, certificate contents, prompts, or raw adapter errors.

- [ ] **Step 1: Write failing clean-run, interruption, and doctor tests**

```go
func TestInterruptedSetupResumesWithoutReinitializingHost(t *testing.T) {
	f := newFakeEnvironment(t, failOnceAt(ServicesStarted))
	_, _ = f.Runner.Run(context.Background(), f.Request)
	report, err := f.Runner.Run(context.Background(), f.Request)
	if err != nil || report.Outcome != Ready { t.Fatalf("%#v %v", report, err) }
	if f.hostInitCalls != 1 { t.Fatalf("init calls=%d", f.hostInitCalls) }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./cmd/lumen -run 'TestInterruptedSetup|TestDoctor' -count=1`

Expected: FAIL because orchestration and full doctor are missing.

- [ ] **Step 3: Implement orchestration and redacted diagnostic aggregation**

Map Host healthy plus Hermes unavailable to `degraded`; map missing/insecure credentials, unsupported isolation, incomplete initialization, or unavailable supervisor to `action_required`; return `ready` only when every selected-profile invariant passes.

- [ ] **Step 4: Add subprocess contract tests for public commands and secret scanning**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./cmd/lumen ./cmd/lumen-host -count=1`

Expected: PASS; setup rerun performs no duplicate initialization or service installation.

- [ ] **Step 5: Commit orchestration**

```bash
git add cmd/lumen cmd/lumen-host internal/setup
git commit -m "feat(setup): complete one-command bootstrap"
```

### Task 7: Prove the configured Hermes execution boundary

**Files:**
- Create: `test/contract/setup_hermes_run_test.go`
- Modify: `internal/setup/doctor_test.go`
- Modify: `cmd/lumen/main_test.go`

**Interfaces:**
- The setup-created deployment completes one bounded `agent.run/execute` task through the existing Host-owned Hermes adapter.
- Approval, cancellation, restart reconciliation, and terminal evidence remain owned by the Host.
- Discovered Hermes features never become Lumen grants; integration inventory and lifecycle remain deferred until their Block 5 capability contracts exist.

- [ ] **Step 1: Write the failing configured-runtime proof**

```go
func TestSetupCreatedDeploymentCompletesBoundedRun(t *testing.T) {
	env := newConfiguredSetupEnvironment(t)
	got := env.SubmitAndWait("agent.run/execute", "return the synthetic marker")
	if got.Status != "completed" { t.Fatalf("got %#v", got) }
}
```

- [ ] **Step 2: Run test and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./test/contract ./cmd/lumen -run 'TestSetupCreatedDeployment|TestRuntimeAuthority' -count=1`

Expected: FAIL until setup produces a runnable Host/Hermes deployment.

- [ ] **Step 3: Connect setup output to the existing bounded execution contract**

Use the existing Hermes capability client and Host execution lifecycle. Do not add a second task runner or expose arbitrary discovered capabilities.

- [ ] **Step 4: Test completion, approval, cancellation, restart recovery, and discovery denial**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./internal/setup ./internal/host ./test/contract ./cmd/lumen -count=1`

Expected: PASS; only the frozen Runs capability is executable and discovery cannot self-enable anything.

- [ ] **Step 5: Commit integration inventory**

```bash
git add cmd/lumen internal/setup internal/host test/contract
git commit -m "test(setup): prove configured Hermes execution"
```

### Task 8: Complete automated gates and owner evidence

**Files:**
- Create: `test/contract/setup_journey_test.go`
- Modify: `mise.toml`
- Modify: `deploy/README.md`
- Modify: `README.md`
- Modify: `docs/PLAN.md`
- Modify: `docs/PHASE-2-HOST-HERMES.md`
- Modify: `docs/CHANGELOG.md`

**Interfaces:**
- `mise run phase2-check` builds both `lumen` and `lumen-host`, runs setup/secret/supervisor contracts, and retains existing Go, race, reproducibility, Termux PIE, and Android companion checks.
- Owner evidence uses the journey `clean environment -> setup -> doctor -> rerun -> reboot -> doctor -> real task -> approval -> cancellation -> interruption -> recovery`.

- [ ] **Step 1: Write the failing clean-environment contract test**

```go
func TestSetupJourneyHasNoManualHermesEdits(t *testing.T) {
	env := newHermeticSetupEnvironment(t)
	report := env.RunLumen("setup", "--profile", "development")
	if report.Outcome != "ready" { t.Fatalf("%#v", report) }
	if env.ContainsGeneratedPlaceholder("YOUR_") { t.Fatal("manual placeholder remains") }
}
```

- [ ] **Step 2: Run the contract and verify RED**

Run: `GOCACHE=/private/tmp/lumen-go-cache go test ./test/contract -run TestSetupJourney -count=1`

Expected: FAIL until the distribution invokes the complete setup runner.

- [ ] **Step 3: Extend the Phase 2 gate and deployment documentation**

Add deterministic builds for both binaries, shell/plist/Compose validation, complete setup contracts, and a scan proving generated output contains no test secrets or unresolved placeholders.

- [ ] **Step 4: Run automated verification**

Run: `ANDROID_HOME=/Users/ashwanthreddyboddireddy/Library/Android/sdk GOCACHE=/private/tmp/lumen-go-cache mise run phase2-check`

Expected: PASS with no skipped required platform contract.

- [ ] **Step 5: Record separate owner proofs**

Record exact versions and redacted evidence for macOS development, Linux/VPS hardened, physical Android Termux compatibility, and Termux Host to isolated Hermes hardened. A compile, emulator, template, or same-UID Termux run cannot substitute for its named proof.

- [ ] **Step 6: Run Graphify and independent security review**

Run: `graphify update .`

Expected: graph rebuild succeeds. Independent review covers setup journal integrity, artifact verification, command execution, secret handling, profile downgrade, supervisor permissions, and rollback.

- [ ] **Step 7: Commit verified Block 2 setup**

```bash
git add cmd internal deploy test mise.toml README.md docs
git commit -m "feat(setup): deliver Lumen-owned Host bootstrap"
```

## Later block handoff

- Block 3 implements single-use short-lived QR/manual pairing, node-generated keys, explicit Host authorization, authenticated transport, reconnect, revocation, and device management.
- Block 4 turns the Redmi/Xiaomi Android app into a companion and capability node without Host authority.
- Block 5 exposes Telegram, WhatsApp, browser, MCP, tools, skills, voice, delegation, and remote execution incrementally through typed Lumen capability contracts.
- Later capability-first milestones in [PLAN.md](../../PLAN.md) own conversation, node, voice, messaging, autonomy, migration, and launch sequencing.
