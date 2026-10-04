---
phase: 02-close-the-existing-host-hermes-foundation
reviewed: 2026-09-27T20:14:59Z
depth: deep
files_reviewed: 6
files_reviewed_list:
  - cmd/lumen/main.go
  - cmd/lumen/main_test.go
  - internal/setup/planner.go
  - internal/setup/planner_test.go
  - internal/setup/supervisor.go
  - internal/setup/supervisor_test.go
findings:
  critical: 4
  warning: 1
  info: 0
  total: 5
status: issues_found
---

# Phase 02: macOS LaunchAgent code review

> Historical initial review snapshot (2026-09-27). Later implementation and disposition updates are tracked in [02-REVIEW.md](02-REVIEW.md) and the dated evidence follow-ups; do not treat this snapshot's finding counts as current.

## Summary

The revised code targets the current user's `gui/<uid>` domain, rejects combined macOS setup across supervisor choices, and emits plist string elements for program arguments. The remaining defects below prevent a safe claim that external macOS setup is durable, secure, or ready. This review did not register or alter a live launchd job. The scoped Go test attempt was inconclusive because the sandbox denied writes to the Go build cache and the tests' home-directory temporary paths.

## Narrative Findings (AI reviewer)

### CR-01 — Persistent service executes a replaceable artifact

**Classification:** BLOCKER
**File:** `cmd/lumen/main.go:1383-1395`
**Issue:** The LaunchAgent permanently references `LUMEN_LUMEN_ARTIFACT` after `checkArtifacts` checks its bytes. `setup.VerifyArtifact` hashes the file once, but neither function restricts ownership or writable ancestors. An artifact in a shared writable directory can be replaced after verification, so a later launch executes bytes the manifest never approved.
**Fix:** Install the verified artifact into a private, owner-controlled generation path before writing the plist, or reject paths whose file or ancestor ownership/modes allow another principal to replace the executable. Verify that exact path before starting or restarting.

### CR-02 — Bootstrap registration does not survive a new login

**Classification:** BLOCKER
**File:** `cmd/lumen/main.go:1391-1395`, `cmd/lumen/main.go:1441-1442`, `internal/setup/supervisor.go:470`, `internal/setup/supervisor.go:519-520`
**Issue:** The plist lives under `<dataDir>/setup/services`. `launchctl bootstrap gui/<uid> <path>` registers it for the current GUI session. `launchctl enable` changes a disabled-state override; it does not make an arbitrary plist path discoverable after logout or reboot. `BootStatus` can therefore report enabled while the Host does not return in a new session. A GUI agent also requires an active user login, which must be reflected in the supported Host profile.
**Fix:** Place an owned, private regular plist in a launchd auto-loaded user agent location, with exact collision and recovery checks. Verify a logout/login or reboot on a disposable Mac before claiming boot persistence, and document the logged-in-user requirement if this remains a GUI agent.

### CR-03 — Setup can report Ready while the Host failed to start

**Classification:** BLOCKER
**File:** `cmd/lumen/main.go:1062-1065`, `cmd/lumen/main.go:1114-1124`
**Issue:** `CommandSupervisor.Control(start)` returns an observed service state, but the caller ignores it and sets `s.started = true` as long as the launchctl commands return success. `verifyStage(ServicesStarted)` checks only that boolean, and `Validated` verifies stored Host initialization rather than live Host readiness. A job that starts and immediately exits can be journaled as validated and reported Ready. Reruns with a validated journal do not reobserve the service.
**Fix:** Require a running service and a bounded authenticated Host readiness check before recording the started/validated stages. On a validated rerun, observe live status and report degraded or action required when the Host is absent.

### CR-04 — The only shipped manifest cannot support the only allowed Mac topology

**Classification:** BLOCKER
**File:** `cmd/lumen/main.go:1240-1255`, `cmd/lumen/main.go:1375-1377`
**Issue:** macOS setup now accepts only external topology, but `selectedArtifacts` requires a manifest with exactly the requested topology. The only shipped `deploy/manifest-v1.json` declares `combined`. Consequently, default external Mac setup fails at the artifact stage and never reaches the new LaunchAgent implementation.
**Fix:** Ship and verify an external-topology manifest with the published Host artifact, or define a narrowly validated rule for selecting that Host artifact from the combined manifest. Add an end-to-end default external Mac setup test.

### WR-01 — Same plist path does not prove the loaded job has current contents

**Classification:** WARNING
**File:** `internal/setup/supervisor.go:440-475`
**Issue:** `launchdBootstrap` treats `path = <expected>` in `launchctl print` as proof that the loaded job matches the newly generated plist. launchd has already consumed its own copy of the job definition. If a file at that path is replaced while a prior job remains loaded, rerun accepts the stale executable and environment. This can occur during manual recovery or restored setup data.
**Fix:** Compare the loaded job's program and critical environment against the desired definition, or use an immutable definition path tied to its content and reject any loaded job whose identity differs. Cover the same-path/stale-content case in the bootstrap tests.

---

_Reviewer: independent Phase 02 review agent_
_Depth: deep_
_No source files modified._

## CR-03 readiness follow-up (2026-09-27)

**Verdict:** The current change addresses CR-03 for `lumen setup`; no new readiness blocker found in the reviewed diff. `verifyRuntimeReady` rejects supervisor errors, missing services, unexpected service names, and every state other than `running` (`cmd/lumen/main.go:1139-1158`). It then requires the authenticated Host control socket to return `status=ready` within a bounded wait (`cmd/lumen/main.go:1161-1181`, `cmd/lumen/main.go:772-788`). Both `ServicesStarted` and `Validated` use that check (`cmd/lumen/main.go:1124-1134`), and `SetupRunner.Run` repeats `Validated` verification before reporting Ready on a rerun (`internal/setup/runner.go:95-101`).

The test fixture's Linux-platform and Host-status overrides are package globals assigned and restored only by tests (`cmd/lumen/main_test.go:658-665`); no production environment variable or other production setter bypasses the default authenticated observer. The broad CLI journey tests remain synthetic because that fixture always reports Host running; the separate socket tests cover authenticated success and Host absence (`cmd/lumen/main_test.go:70-107`), while the runner test covers a failing recheck after prior validation (`internal/setup/runner_test.go:34-60`). A live supervisor/Host restart gate is still required for Phase 02. The coordinator reports that the full nine-package Go suite passed after this change; this reviewer did not rerun it.

## CR-04 coordinator disposition (2026-09-28)

**Implementation status:** addressed in `cmd/lumen/main.go`; external setup may select only the Host artifact from a validated combined manifest when the request and setup plan both specify external topology. Added `TestExternalSetupSelectsHostFromCombinedReleaseManifest`; focused regression passed. This is coordinator validation, not independent reviewer sign-off. Live external Mac setup remains unverified, and CR-01, CR-02, and WR-01 remain open.
