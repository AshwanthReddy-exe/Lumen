---
phase: 02-close-the-existing-host-hermes-foundation
reviewed: 2026-09-28T01:39:01Z
depth: deep
files_reviewed: 10
files_reviewed_list:
  - cmd/lumen/main.go
  - cmd/lumen/main_test.go
  - cmd/lumen/account_home.go
  - cmd/lumen/account_home_osusergo.go
  - internal/setup/supervisor.go
  - internal/setup/supervisor_test.go
  - internal/setup/journal.go
  - internal/setup/runner.go
  - cmd/lumen-host/main.go
  - cmd/lumen-host/main_test.go
findings:
  critical: 2
  warning: 2
  info: 0
  total: 4
status: issues_found
---

# Phase 02: LaunchAgent code review

## 2026-09-28 follow-up: uncertain publication outcome

**Verdict:** The narrow post-publication outcome warning is closed. When directory durability is uncertain, an extant regular plist or an unreadable/unknown final path now reports `launchagent_activation_pending`; only a positively absent path remains a plain durability failure. Both the activation-pending and durability sentinels remain discoverable. An independent re-review found no bypass in this change. The focused regression passed. This does not close CR-02/CR-03: legacy ungated plist handling, the absent-gate behavior, GUI-login scope, and live login/reboot remain open. Full `publishLaunchAgent` command-order/readiness fault injection also remains unverified.

## 2026-09-28 independent re-review of journal-backed activation

**Verdict:** CR-03 remains open. The new staged/gated flow and journal ordering pass focused tests, but the following gaps prevent safe live setup:

- **Blocker:** an existing ungated auto-load plist is rejected as different content but is not migrated or disabled; the Host permits `serve` when the gate environment variable is absent.
- **Warning (WR-01):** rerun treats any running label as ready without checking the loaded arguments or gate environment, so a stale ungated definition may remain active.
- **Warning:** if parent-directory sync fails after the final hard link, the plist remains but setup returns a generic stage failure rather than an uncertain outcome.

Missing regression coverage: positive activation with matching evidence, mismatched journal/binding rejection, stale loaded definition, and live login/reboot transition. No files or launchctl state were changed by the reviewer. Focused implementation tests passed, but neither they nor this review establish live startup safety.

The journal-binding gap was fixed and independently re-reviewed afterward: terminal evidence contains a digest of the complete binding, the activation gate verifies it, legacy missing digests are upgraded only after runner verification, and nonempty mismatches cannot be rebound. Positive and endpoint-substitution tests pass. This closes that subfinding only; legacy plist migration, stale loaded definitions, uncertain publication, and live login/reboot evidence remain unresolved.

## 2026-09-28 independent re-review of LaunchAgent re-registration

**Verdict:** the specific WR-01 same-path acceptance bypass is closed. `launchdBootstrap` now boots out only a loaded label whose reported path is the owned staged or published definition, bootouts with uncertain results fail closed, and every bootstrap error remains an error rather than being accepted from path equality. Published finalization bootstraps the gated auto-load definition and verifies both Launchd running state and authenticated Host readiness. The reviewer confirmed that wrong-path jobs are preserved and the staged-to-published retry path is covered by focused tests.

Remaining limitations: there is no end-to-end `publishLaunchAgent` test for the gated staged-to-published command sequence, wrong-path publication collision, or readiness-failure retry. Since the auto-load plist is published before bootstrap/readiness, a later login can start the validated Host after setup has reported an error; this uncertain/deferred-start outcome remains part of CR-03 and requires explicit recovery semantics plus live macOS verification. CR-02, CR-03 and WR-02 remain open. No LaunchAgent or launchctl state was changed.

## Summary

The generated plist is private and uses the current user's launchd domain, and setup checks authenticated Host readiness. CR-01 (cross-principal artifact replacement) has been independently re-reviewed and resolved. Existing plist leaf ACL and same-content collision checks are independently reviewed and tested. Journal-bound activation evidence is independently reviewed and exact-binding authorization is closed. CR-02 login persistence and CR-03 legacy-plist migration/recovery remain blockers; the specific WR-01 same-path stale-definition bypass is closed after re-registration changes, while post-publication failure outcomes and WR-02 readiness timing remain open. The separate false-Ready setup finding in the earlier `02-01-REVIEW.md` has been resolved by the authenticated readiness checks. CR-04 was also resolved. No live LaunchAgent was installed or changed during this review.

## Narrative Findings (AI reviewer)

### Critical issues

#### CR-01 (resolved 2026-09-28): The manifest digest is checked before launchd opens a replaceable executable

**Classification:** RESOLVED
**File:** `cmd/lumen/main.go:1515-1533` (also `cmd/lumen/main.go:1448-1457`, `internal/setup/artifacts.go:205-231`)
**Issue:** `verifiedLaunchAgentArtifactPath` hashes a pathname, closes the file, and writes that same pathname into a persistent plist. The checks reject group/other writes, but allow the artifact or any ancestor to be writable by the current user. That user can replace the file by rename after the digest check, including after setup returns and before a later login or restart. A competing replacement between `Lstat`, `EvalSymlinks`, `VerifyArtifact`, and launchd's eventual exec has the same result: the Host starts with bytes that never matched the manifest. A root-owned leaf under a user-writable parent is also replaceable by that user. The test at `cmd/lumen/main_test.go:199-235` tests only other-user mode and symlink rejection; it cannot establish launch-time identity.
**Fix:** Copy from a verified open file into a dedicated generation path, verify the installed copy, and make the executable path nonreplaceable by principals outside the explicitly trusted owner boundary. If the manifest digest is promised at every restart, verify it through a protected pre-exec path or use an OS-owned immutable installation location; a one-time setup hash alone cannot provide that guarantee.

**Resolution:** `validateProtectedPath` checks ownership, symlinks and group/other write bits for every component; on Darwin, `rejectPathACL` inspects each component with `/bin/ls -lde`, fails closed on errors or malformed/allow entries, and accepts deny-only entries. The independent reviewer confirmed this prevents a different UID from gaining write/delete/child-entry replacement rights after verification. Same-UID processes and root remain trusted by the documented boundary; no claim is made against them or that launchd re-hashes the artifact at every later exec. The macOS regression adds `everyone allow add_file,delete_child` while POSIX group/other write bits remain clear and verifies rejection. Review verdict: no other-principal TOCTOU bypass found.

#### CR-02: Boot readiness does not establish that the agent will load at the next login

**Classification:** BLOCKER
**File:** `cmd/lumen/main.go:1489-1512`, `internal/setup/supervisor.go:389-402` (also `internal/setup/supervisor.go:428-430`)
**Issue:** `os.UserHomeDir` uses the process's `$HOME` on macOS. An owner invoking setup with an alternate private `HOME` passes every current ownership/mode check, writes the plist under that alternate tree, and successfully bootstraps the current `gui/<uid>` session. At the next normal login, launchd scans the account's real `~/Library/LaunchAgents`, not the alternate tree. `BootStatus` still returns true from the `print-disabled` override without checking the plist's canonical location, existence, contents, or fresh-login load. Even with the correct home, `gui/<uid>` is a logged-in-user service, so it cannot provide the project's always-on Mac Host while that user is logged out. The tests at `cmd/lumen/main_test.go:173-197` set `HOME` to a temporary directory and at `internal/setup/supervisor_test.go:257-275` mock an enabled entry; neither proves login persistence.
**Fix:** Resolve and validate the account's canonical home independently of `$HOME`; make boot evidence include the expected on-disk definition and its loaded target, and qualify it with a real logout/login or reboot. Explicitly scope this profile to logged-in-user availability, or use a properly permissioned LaunchDaemon if unattended pre-login hosting is required.

#### CR-03: Failed or interrupted setup leaves an auto-starting plist behind

**Classification:** BLOCKER
**File:** `cmd/lumen/main.go:1435-1458` (also `internal/setup/supervisor.go:451-482`, `internal/setup/runner.go:124-130`)
**Issue:** Setup now keeps the generated plist under the private setup directory while installing and starting the service, and checks owned-service plus authenticated Host readiness before publishing the final LaunchAgents path. However, the final-path hard link is created before its parent-directory sync succeeds. If that sync fails, setup reports an error while the autoload plist remains; process death after the link has the same uncertain outcome. The journal has no durable validation/activation record, and there is no load-time gate or explicit recovery rule for that publication boundary. A later login may start the Host after the caller observed an incomplete/uncertain setup result.
**Fix:** Define a durable publication/activation commit and its recovery semantics so a post-publication error or crash cannot leave an ambiguous setup result. This may require a load-time activation gate or an equivalent journal/reconciliation protocol. Inject failure after publication and verify the next-login behavior on a disposable Mac. Do not claim closure based only on staging before readiness; readiness narrows the window but does not resolve the durable commit boundary.

#### CR-04 (resolved 2026-09-27T23:55:15Z): A path containing “running” could make a stopped launchd job report running

**Classification:** BLOCKER
**File:** `internal/setup/supervisor.go:490-516` (also `internal/setup/supervisor.go:582-599`)
**Issue:** `observe` searches the entire `launchctl print` output for the substring `running`, rather than the job's `state =` field. For a stopped job whose printed executable or plist path is `/Users/alice/running/lumen-host`, output containing `state = waiting` still produces `StateRunning`. `lumen service status` can then report `ready` for a stopped service, and the supervisor component of `lumen doctor` is false. The current tests only feed `state = running`; they do not cover a stopped state with unrelated `running` text.
**Fix:** Parse the exact launchd state line for the requested job and map `state = running` only to `StateRunning`; retain manager-specific parsing for systemd and runit. Add a stopped-job fixture whose path contains `running`.

**Resolution:** `internal/setup/supervisor.go:498-537` now routes launchd status through `launchdServiceState`, which reads only a `state =` line; `waiting` maps to stopped even when an unrelated path includes `/running/`. `TestLaunchdStatusUsesOnlyTheStateField` covers that case and a genuine running state. CR-04 is excluded from the frontmatter's open counts.

### Warnings

#### WR-01 (resolved 2026-09-28): The same plist path does not identify the loaded definition

**Classification:** RESOLVED
**File:** `internal/setup/supervisor.go:441-482`
**Issue:** On rerun, `launchdBootstrap` accepts a loaded label solely because `launchctl print` says it came from the expected path. Launchd can retain an earlier executable and environment after the on-disk file at that path is restored or replaced; `writePrivateLaunchAgent` only checks current file bytes. The existing same-path test at `internal/setup/supervisor_test.go:285-294` supplies only `path = ...`, so it codifies this false identity check.
**Resolution:** `launchdBootstrap` no longer treats same-path output as identity. It refuses a different path, then bootouts only the accepted Lumen staged/published label and bootstraps the currently verified requested definition. Failed bootout/bootstrap is not converted to success by a subsequent same-path print. Tests cover same-path reload, staged-to-published retry after an interrupted bootstrap, uncertain bootout, and wrong-path collision preservation. Independent review confirms the specific bypass is closed. Live macOS behavior and end-to-end publication failure recovery remain open under CR-03.

#### WR-02: The supervisor state is sampled only once before the readiness wait

**Classification:** WARNING
**File:** `cmd/lumen/main.go:1139-1158`
**Issue:** After `Control(start)`, `verifyRuntimeReady` immediately returns `owned services are not ready` if the first status sample is `unknown` or `stopped`. Its 10-second retry loop applies only to the Host socket after supervisor state has already passed. A service that transitions from starting to running within that interval causes setup to return `action_required` and leaves the service stage uncommitted, including on Linux/Docker paths using this shared function.
**Fix:** Poll both owned supervisor states and authenticated Host readiness under one bounded deadline, failing immediately only for a terminal manager error. Cover an initial non-running state followed by running/ready.

## Verification and live limits

Focused tests for generated plists, the LaunchAgents directory, launchd bootstrap, boot-status parsing, and lifecycle command routing passed: `rtk go test ./cmd/lumen ./internal/setup -run 'TestWriteLaunchAgentIsPrivateEscapedAndIdempotent|TestLaunchAgentDirectoryIsAutoLoadedAndOwnerControlled|TestSupervisorBootStatusRequiresExactLaunchdEnabledEntry|TestLaunchdBootstrapResumesOnlyTheSameDefinition|TestLaunchdLifecycleUsesCurrentUserDomain' -count=1` (8 tests). A full two-package run was inconclusive in this sandbox because many test helpers create temporary directories under the unwritable real home (`operation not permitted`); this is not counted as a product failure. Read-only `launchctl print-disabled gui/<uid>` confirmed the expected enabled-entry output shape, but no Lumen job was registered.

The following remain unproven without a disposable live macOS logout/login or reboot test: whether this exact generated plist is auto-loaded, whether the Host returns with its bound identity and credentials, whether a failed/interrupted setup leaves a deferred job, whether a modified artifact is rejected at the next launch, and whether `lumen doctor` reports the correct logged-out/relogged-in state. The GUI domain cannot establish pre-login availability by a unit test.

## CR-01 independent security re-review — 2026-09-28

**Verdict:** Closed for replacement by a different OS principal, within the stated same-UID/root trust boundary. Apple documents that filesystem ACLs supplement BSD mode permissions; path validation now rejects ACL entries that grant rights and fails closed on unknown output, while preserving deny-only ACLs already present on the account home. The reviewer found no remaining different-UID replacement route through accepted components.

**Checks run:** `GOCACHE=/tmp/lumen-phase2-go-cache go test ./cmd/lumen -run 'TestLaunchAgentArtifact(MustBeUnreplaceableByOtherUsers|RejectsMacOSACLOnParent)' -count=1 -v` passed on macOS. `GOCACHE=/tmp/lumen-phase2-go-cache go test ./cmd/lumen -count=1` passed. `git diff --check` passed. `graphify update .` completed. No live launchctl state was changed. The test fixture adds an ACL granting `add_file,delete_child` while POSIX group/other write bits are clear, and confirms the artifact path is rejected.

---

_Reviewed: 2026-09-27T23:35:44Z_
_Reviewer: independent Phase 02 code reviewer_
_Depth: deep_
_No source files modified._

## Account-home lookup follow-up — 2026-09-27T23:44:24Z

**Verdict:** The `$HOME` redirection subcase of CR-02 is closed for the normal macOS build. `launchAgentDirectory` now obtains `user.Current().HomeDir` at `cmd/lumen/main.go:1490-1504`; Go's Darwin user lookup reads the current real UID's account record, and `launchAgentDirectoryForHome` still resolves symlinks and rejects unexpected ownership or writable path components at `cmd/lumen/main.go:1506-1525`. An ordinary `HOME` override no longer chooses the plist location. This does **not** close CR-02: the job still lives only in `gui/<uid>`, `BootStatus` still checks an enabled override rather than a future login, and no logout/login or reboot proof exists. Running the CLI under another account or `sudo` selects that account's UID/home/domain, not the intended logged-in owner; the caller must use the correct account.

### WR-03: The HOME-independence test primes the cached user before changing HOME

**Classification:** WARNING
**File:** `cmd/lumen/main_test.go:200-216`
**Issue:** The test calls `user.Current()` before `t.Setenv("HOME", ...)`. Go caches `user.Current()` on its first call, so the subsequent call inside `canonicalAccountHome` can only return the pre-change value. The test passes even under the `osusergo` build path, whose user lookup may fall back to `$HOME` when the account database has no entry. It therefore does not prove the intended behavior for a fresh process started with a redirected `HOME`. Separately, `canonicalAccountHome` does not reject an empty or relative account `HomeDir`: `filepath.Abs` would convert it to a path under the current working directory, which can again be a non-login directory if an account record is malformed.
**Fix:** Test the first lookup in a child process launched with altered `HOME`, compare with an independently obtained account home, and reject empty or relative `account.HomeDir` before `filepath.Abs`. Keep the stock Darwin and any supported `osusergo` build modes distinct in the test claim.

**Tests run:** `rtk proxy env GOCACHE=/private/tmp/lumen-launchagent-review-cache go test ./cmd/lumen -run 'TestCanonicalAccountHomeIgnoresHomeEnvironment|TestLaunchAgentDirectoryIsAutoLoadedAndOwnerControlled' -count=1` passed; `rtk proxy env GOCACHE=/private/tmp/lumen-launchagent-review-cache go test -tags=osusergo ./cmd/lumen -run '^TestCanonicalAccountHomeIgnoresHomeEnvironment$' -count=1` passed, illustrating that the test cannot distinguish the fallback. The same two commands without the temporary `GOCACHE` failed before test execution because the sandbox could not open the default Go build cache. No launchctl state was changed.

## Account-home hardening re-review — 2026-09-27T23:50:24Z

**Verdict:** The HOME-redirection subcase is now closed for the supported Darwin build profiles. `account_home.go:13-22` obtains the system account home and rejects empty or relative values before returning a clean absolute path. `main.go:1489-1517` passes that path through the existing symlink, owner, and mode checks. The `osusergo` implementation at `account_home_osusergo.go:9-11` returns an error before the LaunchAgents directory is created, so that build fails closed. The rewritten `main_test.go:201-237` starts a fresh test process with a redirected `HOME`, avoiding the cached-lookup defect in former WR-03; that historical finding is resolved and excluded from the frontmatter's open counts. No new HOME-redirection blocker was found. This conclusion does not qualify Linux launchd overrides, and CR-02 remains BLOCKER for GUI-login scope, enabled-override-only boot evidence, and absent live logout/login or reboot proof. CR-01, CR-03, WR-01, and WR-02 remain unchanged; CR-04 was unresolved at the time of this account-home re-review and is resolved below.

### WR-04 (resolved 2026-09-27T23:53:01Z): A failing home-lookup test printed a personal path

**Classification:** WARNING
**File:** `cmd/lumen/main_test.go:213-236`
**Original issue:** On mismatch, the child test printed both the observed and expected account home; the parent included the child's combined output in its failure. An absolute home commonly contains the owner's username, contrary to the repository rule against personal identifiers in test logs.
**Fix:** Report a generic account-home mismatch without printing either path; keep the exact paths only in process-local comparisons.

**Resolution:** `cmd/lumen/main_test.go:201-237` now uses generic failure messages for lookup failure and account-home mismatch. The parent process may still print the child test output on failure, but those targeted failure paths no longer include either home value. WR-04 is resolved and excluded from the frontmatter's open counts. This does not change the unresolved CR-02 login/reboot limitation.

**Checks run:** Both `rtk proxy env GOCACHE=/private/tmp/lumen-account-home-review.OE1j8l/cache TMPDIR=/private/tmp/lumen-account-home-review.OE1j8l go test ./cmd/lumen -run '^TestCanonicalAccountHomeIgnoresHomeEnvironment$' -count=1` and the same command with `-tags=osusergo` passed. With `CGO_ENABLED=0`, `go build ./cmd/lumen` passed for Darwin and Linux, each at amd64 and arm64, both with default tags and with `-tags=osusergo` (eight builds); `file` identified the outputs as the expected Mach-O or ELF architecture. Go emitted sandbox stat-cache write warnings while the builds still exited 0. No live launchctl operation was run.

## Launchd status re-review — 2026-09-27T23:55:15Z

**Verdict:** CR-04 is resolved for the reported failure mode. The launchd-specific parser reads the exact `state` field and maps `waiting` to stopped; a separate path containing `/running/` cannot override it. The targeted status test passed with a mocked `launchctl print` response containing `state = waiting` and `path = /Users/alice/running/lumen-host`, and also checked `state = running`. CR-01, CR-02, CR-03, WR-01, and WR-02 remain open. This unit evidence does not replace the live macOS logout/login or reboot evidence required for CR-02.

**Checks run:** `rtk proxy env GOCACHE=/private/tmp/lumen-launchd-status-review.Ocl5aG/cache TMPDIR=/private/tmp/lumen-launchd-status-review.Ocl5aG go test ./internal/setup -run 'TestLaunchdStatusUsesOnlyTheStateField|TestLaunchdLifecycleUsesCurrentUserDomain|TestSupervisorStatusParsesRunningAndBoundsContext' -count=1` passed. The first attempt did not execute tests because its chosen `TMPDIR` did not exist; the passing run used a newly created temporary directory. No live launchctl operation was run.

## Activation-gate follow-up — 2026-09-28

**Verdict:** Independent security review passed the narrow gate/recovery change. Ungated macOS service activation is denied unless the invocation is explicitly manual or uses the journal-bound staging path. The staging path is allowed only during `ServicesInstalled`, `ServicesStarted`, or finalization/recovery at `Validated`; if terminal evidence exists, the full exact binding must validate. Non-Darwin startup without the gate environment remains unaffected. Generated staged plist uses `serve --setup-staging`; published plist uses `serve` with `LUMEN_REQUIRE_VALIDATED_SETUP=1` and no staging argument. Legacy on-disk plist migration, Launchd retry policy/live behavior, and post-publication crash/reboot evidence remain open under CR-03/CR-02.

**Checks run:** `rtk go test ./cmd/lumen-host -run '^TestLaunchActivationGate|^TestCLIProcessLifecycleAndBoundary$' -count=1` (4 tests) and `rtk go test ./cmd/lumen -run '^TestWriteLaunchAgentIsPrivateEscapedAndIdempotent$' -count=1` (1 test) passed; `rtk git diff --check` passed; Graphify refreshed. No live Launchd state was changed.
