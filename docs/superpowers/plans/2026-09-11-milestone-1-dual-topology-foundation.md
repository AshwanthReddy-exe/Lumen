# Milestone 1 Dual Topology Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the Host foundation for both a combined Lumen and Hermes deployment and a Lumen Host connected to an existing independently managed Hermes service.

**Architecture:** One typed setup engine persists an explicit `combined` or `external` topology independently from assurance profile. The Host remains the only Space authority and gives Hermes only bounded task context through the existing authenticated Runs adapter; topology changes artifact and supervisor ownership, never Space behavior.

**Tech Stack:** Go 1.27.1 standard library, existing encrypted Host/store, Hermes HTTP Runs adapter, JSON manifest/journal/configuration, systemd and Docker Compose reference deployment, existing `mise` gates.

**Spec:** `docs/superpowers/specs/2026-09-11-milestone-1-dual-topology-design.md`

## Global Constraints

- Milestone 0 remains complete at its intended Space authority boundary; change only stale validation references unless a failing current Go contract proves a defect.
- Milestone 1 is the only active implementation milestone.
- `combined` owns and supervises both Lumen and Hermes; `external` owns and supervises only Lumen and never mutates the Hermes machine.
- Topology and assurance profile are independent persisted values.
- Host owns canonical Space state, authorization, approvals, task outcomes, and audit.
- Hermes receives only task-scoped context through the versioned Runs adapter and cannot create grants or access the encrypted store or owner control socket.
- Report only `ready`, `degraded`, or `action_required`; never expose secrets or raw untrusted errors.
- Never replace initialized Space identity, silently rotate credentials, weaken a requested profile, or retry an uncertain effect without durable evidence.
- Use existing primitives and the Go standard library; add no second runner, state store, installer framework, generic adapter, or remote-provisioning subsystem.
- One implementation agent mutates the worktree at a time. Tasks execute strictly in dependency order.
- Combined and external Linux/VPS evidence both block Milestone 1 exit. Later node, conversation, memory, messaging, voice, and companion features are not implemented here.

---

### Task 1: Reconcile the Milestone 0 validation record

**Files:**
- Modify: `docs/PHASE-1-CONTRACT.md`
- Modify if required by the same stale reference: `docs/CHANGELOG.md`
- Test: `internal/space/*_test.go`
- Test: `internal/store/*_test.go`

**Interfaces:**
- Consumes: current Go Space authority and encrypted store.
- Produces: an accurate historical contract that points to current executable checks without reopening Milestone 0 scope.

- [ ] **Step 1: Run the current contract tests**

  Run `rtk go test ./internal/space ./internal/store -count=1` and `rtk go test -race ./internal/space ./internal/store`. Record failures as implementation defects; do not infer defects from the retired Kotlin experiment.

- [ ] **Step 2: Prove the historical references are stale**

  Run `rg -n 'phase1-check|tools/space-scenario|core/space' docs/PHASE-1-CONTRACT.md docs/CHANGELOG.md mise.toml README.md`. Confirm `mise.toml` has no `phase1-check` task and the removed paths are not active source.

- [ ] **Step 3: Correct only the stale wording**

  Preserve the frozen Kotlin evidence, but label its command and removed paths historical. Name `go test ./internal/space ./internal/store` and the applicable contract tests as the current authority validation.

- [ ] **Step 4: Verify and commit**

  Run the focused tests again and `rtk git diff --check`. Commit `docs(space): reconcile foundation validation`.

### Task 2: Freeze topology in the setup contract

**Files:**
- Modify: `internal/setup/model.go`
- Modify: `internal/setup/planner.go`
- Modify: `internal/setup/planner_test.go`
- Modify: `internal/setup/journal.go`
- Modify: `internal/setup/journal_test.go`
- Modify: `internal/setup/runner.go`
- Modify: `internal/setup/runner_test.go`

**Interfaces:**
- Consumes: existing `setup.Request`, `PlanResult`, `Journal`, and `Runner`.
- Produces: `type Topology string`, constants `combined` and `external`, topology-bearing requests/reports/plans, and an immutable journal binding.

- [ ] **Step 1: Add failing topology tests**

  Add tests named `TestPlanRequiresExplicitSupportedTopology`, `TestPlanKeepsTopologyIndependentFromProfile`, `TestRunnerRejectsChangedTopologyBeforeMutation`, and `TestRunnerRejectsChangedEndpointIdentityBeforeMutation`. Assert that only `combined` and `external` validate and that reruns with a changed binding execute zero stage callbacks.

- [ ] **Step 2: Verify RED**

  Run `rtk go test ./internal/setup -run 'TestPlanRequiresExplicitSupportedTopology|TestPlanKeepsTopologyIndependentFromProfile|TestRunnerRejectsChangedTopology|TestRunnerRejectsChangedEndpointIdentity' -count=1`. Expected: fail because topology and endpoint identity are not part of the contract.

- [ ] **Step 3: Add the minimum contract**

  Define `TopologyCombined` and `TopologyExternal`. Add topology to `Request`, `PlanResult`, and `Report`. Replace `HermesAdopted bool` with explicit topology. Bind profile, topology, endpoint-origin digest, endpoint-identity digest, artifact digests, and plan digest in the journal before mutation. Keep secret values out of every digest input and report.

- [ ] **Step 4: Verify GREEN and compatibility**

  Run the focused tests, then `rtk go test ./internal/setup ./cmd/lumen -count=1` and `rtk git diff --check`.

- [ ] **Step 5: Commit**

  Commit `feat(setup): define deployment topology contract`.

### Task 3: Pin combined distribution inputs

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
- Consumes: immutable topology contract and existing manifest/artifact verifier.
- Produces: integrity-pinned combined artifacts and atomic staged replacement; external topology consumes no Hermes artifact.

- [ ] **Step 1: Add failing manifest and rollback tests**

  Add `TestManifestRejectsMutableSource`, `TestManifestRejectsPlaceholderDigest`, `TestManifestRequiresTopologyOwnership`, and `TestFailedReplacementPreservesInstalledArtifact`. Combined records require immutable Lumen and Hermes inputs; external records require only Lumen ownership.

- [ ] **Step 2: Verify RED**

  Run `rtk go test ./internal/setup -run 'TestManifestRejects|TestManifestRequiresTopologyOwnership|TestFailedReplacementPreservesInstalledArtifact' -count=1` and confirm the new validation paths fail.

- [ ] **Step 3: Implement immutable records**

  Record the official Hermes repository, selected immutable tag and peeled commit, Python version, official lockfile requirement, and frozen build command. Reject `example.invalid`, mutable references, zero digests, zero sizes, and topology-inconsistent ownership. Stage, hash, fsync, atomically replace, verify, and restore the prior installed artifact on failure using existing artifact helpers.

- [ ] **Step 4: Produce reproducible candidates**

  Build the Linux amd64 Lumen binary twice with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w'`. Build Hermes twice from the pinned source with `uv sync --frozen` through `deploy/docker/Dockerfile.hermes`. Put computed identities in the manifest; do not hand-write placeholder hashes.

- [ ] **Step 5: Verify and commit**

  Run focused tests, two-build digest comparisons, and `docker compose -f deploy/docker/compose.yaml config`. Commit `build(release): pin combined topology artifacts`.

### Task 4: Implement external Hermes adoption

**Files:**
- Modify: `internal/setup/adoption.go`
- Modify: `internal/setup/adoption_test.go`
- Modify: `internal/setup/config.go`
- Modify: `internal/setup/config_test.go`
- Modify: `internal/setup/doctor.go`
- Modify: `internal/setup/doctor_test.go`
- Modify only if a boundary defect is exposed: `internal/hermes/client.go`
- Modify only if a boundary defect is exposed: `internal/hermes/client_test.go`

**Interfaces:**
- Consumes: `TopologyExternal`, existing Hermes health/capabilities client, and TLS identity validation.
- Produces: a durable external-adoption record containing endpoint and compatibility identities but no secret values or remote lifecycle ownership.

- [ ] **Step 1: Add failing external-adoption tests**

  Add `TestExternalAdoptionNeverInstallsOrControlsHermes`, `TestExternalAdoptionRequiresAuthenticatedCompatibility`, `TestExternalAdoptionRejectsEndpointSubstitution`, `TestExternalAdoptionPreservesCredentialReferences`, and `TestHardenedExternalAdoptionRequiresPinnedMutualTLS`.

- [ ] **Step 2: Verify RED**

  Run `rtk go test ./internal/setup -run 'TestExternalAdoption|TestHardenedExternalAdoption' -count=1`. Expected: fail because existing adoption models an executable instead of an independent endpoint.

- [ ] **Step 3: Implement endpoint adoption**

  Validate and normalize the endpoint origin, credential-file permissions, CA/hostname/validity, optional profile requirements, exact leaf pin, client identity, authenticated health, capabilities, and supported Runs contract. Persist only version, capability/profile digest, origin digest, certificate identity digest, and credential paths. Do not write Hermes configuration or invoke a remote command.

- [ ] **Step 4: Implement truthful external observations**

  Doctor classifies an initialized safe Host with a temporarily unavailable previously validated external Hermes as `degraded`. Identity, authentication, compatibility, or binding mismatches are `action_required`. No context is sent before identity and compatibility pass.

- [ ] **Step 5: Verify and commit**

  Run focused setup and Hermes tests, `rtk go test -race ./internal/setup ./internal/hermes`, and `rtk git diff --check`. Commit `feat(setup): adopt existing external Hermes`.

### Task 5: Complete public orchestration for both topologies

**Files:**
- Modify: `cmd/lumen/main.go`
- Modify: `cmd/lumen/main_test.go`
- Modify: `internal/setup/runner.go`
- Modify: `internal/setup/runner_test.go`
- Modify: `internal/setup/doctor.go`
- Modify: `internal/setup/doctor_test.go`
- Modify only for a proven defect: `internal/setup/supervisor.go`
- Modify only for a proven defect: `internal/setup/supervisor_test.go`

**Interfaces:**
- Consumes: Tasks 2-4.
- Produces: public `setup`, `doctor`, and `service start|stop|restart|status` behavior for combined and external topologies.

- [ ] **Step 1: Add failing public-journey tests**

  Add `TestSetupRunsCombinedPlan`, `TestSetupRunsExternalPlan`, `TestSetupRerunPreservesIdentityAndCredentials`, `TestSetupResumesAfterHostInitialization`, `TestDoctorUsesLiveTopologyEvidence`, `TestCombinedServiceControlsBothServices`, and `TestExternalServiceControlsHostOnly`.

- [ ] **Step 2: Verify RED**

  Run `rtk go test ./cmd/lumen ./internal/setup -run 'TestSetupRuns|TestSetupRerun|TestSetupResumes|TestDoctorUsesLiveTopologyEvidence|TestCombinedService|TestExternalService' -count=1`.

- [ ] **Step 3: Compose one runner**

  Parse topology into one typed request, create one plan, one journal, one setup state, and one runner. Combined stage callbacks acquire both verified artifacts, write both configurations, initialize Host once, and manage both services. External callbacks verify the Lumen artifact and adoption record, write only Host connection configuration, initialize Host once, and manage only Host. Remove environment markers, file presence, socket presence, and command discovery as readiness authority.

- [ ] **Step 4: Make doctor and service topology-aware**

  Doctor uses durable binding plus live Host and authenticated Hermes observations. Service actions derive the locally owned service set from topology and never send lifecycle requests to external Hermes.

- [ ] **Step 5: Verify and commit**

  Run focused tests, `rtk go test ./cmd/lumen ./internal/setup ./cmd/lumen-host -count=1`, `rtk go test -race ./cmd/lumen ./internal/setup`, and `rtk git diff --check`. Commit `feat(setup): complete dual topology lifecycle`.

### Task 6: Certify Host-mediated Hermes Space work

**Files:**
- Modify: `test/contract/setup_hermes_run_test.go`
- Modify: `internal/host/execution_test.go`
- Modify only when tests expose defects: `internal/host/execution.go`
- Modify only when tests expose defects: `internal/hermes/client.go`

**Interfaces:**
- Consumes: configured combined or external Hermes and the existing Host-owned task lifecycle.
- Produces: identical task-scoped Runs behavior across topologies, with no raw Space-store or control-socket access.

- [ ] **Step 1: Add failing topology contract cases**

  Add combined and external cases covering completion, exact `once` and `deny` approval, cancellation race, duplicate/reordered events, lost stream, Host restart, Hermes restart, runtime outage/recovery, incompatible capability, endpoint substitution, and proof that discovery never creates a grant.

- [ ] **Step 2: Add isolation assertions**

  Assert the runtime request contains only the task identity, authorized capability/target, runtime-profile digest, deadline/cancellation lineage, and bounded task input. Assert it contains no store path, state key, operator credential, unrelated audit record, or unrestricted Space snapshot.

- [ ] **Step 3: Verify RED**

  Run `rtk go test ./test/contract ./internal/host -run 'TestSetupCreatedDeployment|TestRuntimeRequestIsTaskScoped' -count=1` and confirm the first missing boundary fails.

- [ ] **Step 4: Fix only exposed boundary defects**

  Keep Host as the sole state-transition authority and event consumer. Persist intent and mapping before acknowledgement, reconcile within bounded deadlines, and record `unknown_outcome` when completion cannot be proved. Do not add a general Space-query API.

- [ ] **Step 5: Verify and commit**

  Run focused tests, `rtk go test ./test/contract ./internal/host ./internal/hermes -count=1`, `rtk go test -race ./internal/host ./internal/hermes`, and `rtk git diff --check`. Commit `test(host): certify dual topology Hermes execution`.

### Task 7: Add both automated Linux journeys

**Files:**
- Create: `scripts/lumen-combined-check`
- Create: `scripts/lumen-external-check`
- Create: `test/contract/setup_journey_test.go`
- Modify: `deploy/docker/compose.yaml`
- Modify: `mise.toml`

**Interfaces:**
- Consumes: Tasks 3-6.
- Produces: `milestone1-combined-check`, `milestone1-external-check`, and aggregate `milestone1-check` gates.

- [ ] **Step 1: Add failing script-contract tests**

  Add `TestCombinedJourneyScriptContract` and `TestExternalJourneyScriptContract`. The combined script must prove clean setup, rerun, forced interruption, doctor, both-service restart, task, failed update, rollback, and final state comparison. The external script must prove independent service ownership, adoption, Host-only lifecycle, task, outage/degraded/recovery, identity rejection, and final state comparison.

- [ ] **Step 2: Verify RED**

  Run `rtk go test ./test/contract -run 'TestCombinedJourneyScriptContract|TestExternalJourneyScriptContract' -count=1` and expect missing scripts/gates.

- [ ] **Step 3: Implement bounded journey scripts**

  Use `set -eu`, isolated Compose project names and volumes, synthetic identities/content, bounded waits, and cleanup traps. Preserve redacted failure logs, emit no secret values, and never claim host reboot from a process restart.

- [ ] **Step 4: Register gates**

  Add separate mise tasks and an aggregate `milestone1-check` containing formatting, vet, unit, race, reproducible Linux builds, Compose validation, both journeys, secret scanning, and deployment checks. Keep physical platform checks separate.

- [ ] **Step 5: Verify and commit**

  Run both gates twice from clean volumes, then the aggregate gate. Commit `test(deploy): add dual topology foundation gates`.

### Task 8: Record real deployment and reboot evidence

**Files:**
- Modify: `deploy/README.md`
- Modify: evidence section in `docs/PHASE-2-HOST-HERMES.md`

**Interfaces:**
- Consumes: Tasks 3-7 and owner-controlled environments.
- Produces: dated, redacted combined and external topology evidence; unrun checks remain explicitly open.

- [ ] **Step 1: Record combined evidence**

  On clean Ubuntu 24.04 LTS amd64, run `mise run milestone1-combined-check`, real pinned Hermes completion/approval/cancellation/recovery probes, reboot, doctor, failed update, rollback, and canonical state comparison.

- [ ] **Step 2: Record external evidence**

  Use independently managed Host and Hermes machines. Record versions, topology, assurance profile, certificate preconditions, authenticated Runs behavior, Host reboot, Hermes reboot, outage/reconnection, Host-only service control, and proof that Lumen did not mutate the Hermes machine.

- [ ] **Step 3: Preserve evidence boundaries**

  Record date, commit, versions, topology, preconditions, exact journey, expected/actual outcome, pass/fail, and redacted reference. Mark unavailable credentials, machines, reboots, or failed checks open; never replace them with synthetic claims.

- [ ] **Step 4: Commit when evidence exists**

  Run `rtk git diff --check`. Commit `docs(evidence): record dual topology foundation proof`. If owner access is unavailable, stop this task with the precise evidence blocker and do not execute Task 9.

### Task 9: Close Milestone 1

**Files:**
- Modify: `docs/PLAN.md`
- Modify: `docs/PHASE-2-HOST-HERMES.md`
- Modify: `docs/ARCHITECTURE.md` only if implemented boundaries changed
- Modify: `docs/DECISIONS.md` only for new durable decisions
- Modify: `docs/CHANGELOG.md`
- Modify: `README.md` and `deploy/README.md` only where owner instructions changed

**Interfaces:**
- Consumes: all task reviews, automated gates, and owner evidence.
- Produces: an honest Milestone 1 exit and the entry condition for the Conversation and Memory nucleus.

- [ ] **Step 1: Run independent final review**

  Package the entire plan diff and dispatch a reviewer for architecture, authority, security, recovery, privacy, topology separation, and over-engineering. Fix every Critical or Important finding through one bounded fix wave and one scoped re-review.

- [ ] **Step 2: Run the complete gate**

  Run `rtk mise run milestone1-check`, `rtk go test ./... -count=1`, `rtk go test -race ./...`, `rtk git diff --check`, and `rtk graphify update .`. Capture fresh output.

- [ ] **Step 3: Decide from evidence**

  Mark Milestone 1 complete only when both topologies passed automated and real-runtime/reboot/rollback gates and no unresolved Critical or Important finding remains. Otherwise keep it active and list each precise open check.

- [ ] **Step 4: Align the product roadmap**

  Preserve the later journey order: canonical conversations and common memory, secure generic node fabric, cross-node continuity/action, Jarvis voice/presence, messaging, managed portability, earned autonomy, and ecosystem launch. Describe old-phone desk companion and Mac desk pet only as optional node/companion surfaces, never as authority or platform assumptions.

- [ ] **Step 5: Commit**

  Commit `docs(milestone): close dual topology foundation` only if the exit gate passed; otherwise commit an evidence checkpoint that keeps Milestone 1 active.
