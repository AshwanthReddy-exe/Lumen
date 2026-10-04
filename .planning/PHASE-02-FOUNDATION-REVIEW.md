# Phase 02 foundation implementation review

Reviewed: 2026-10-03. Scope: the current uncommitted Phase 02 implementation delta against `origin/feat/host-hermes-foundation-closure`, plus the relevant setup, supervision, and recovery contracts. This is an independent source review, not a production qualification.

## BLOCKER — A repeat macOS setup interrupts the running Host

**Files:** `internal/setup/runner.go:97-116`, `cmd/lumen/main.go:1061-1064`, `cmd/lumen/main.go:1545-1595`, `internal/setup/supervisor.go:458-480`.

Once the journal is at `Validated`, every subsequent `lumen setup` invokes `RunStage(Validated)`. On launchd that always invokes `publishLaunchAgent`, which always calls `CommandSupervisor.Install`. `launchdBootstrap` then unconditionally performs `launchctl bootout` of an already loaded matching job before bootstrap. A no-change setup rerun therefore kills and restarts a healthy Host, interrupting live tasks, even though the setup journal says the same deployment is validated. The existing test at `internal/setup/supervisor_test.go:312` asserts the bootout path, but does not test the user-facing idempotent rerun contract.

**Fix:** If the published definition and loaded path already match, verify the running job and return without bootout. Only replace the staging job during the first committed publication or when an explicit release operation changes the service. Add a rerun test that proves no `bootout`, no second bootstrap, and an unchanged Host process/task.

## WARNING — macOS auto-start readiness is not qualified by a real login or reboot

**Files:** `internal/setup/supervisor.go:390-403`, `cmd/lumen/main.go:1128-1171`, `.planning/phases/02-close-the-existing-host-hermes-foundation/02-10-EVIDENCE.md:34-52`.

The new `BootStatus` path parses `launchctl print-disabled` and the setup readiness path checks a currently running job. These checks cannot establish that launchd reloads the job after logout/login or reboot, or that an interrupted stage-to-published plist transition recovers on real macOS. The evidence explicitly covers Docker-based synthetic lifecycle and distinguishes it from the still open live deployment gates. Keep the macOS auto-start profile gated until the Phase 10 physical login/reboot and interrupted-publication journey is recorded.

**Fix:** Add real macOS login/reboot evidence with pre/post journal, job path, process identity, Host status, and unchanged Space identity, plus interrupted publication recovery. Do not mark this gate passed from unit tests or Docker Compose.

## WARNING — Compose init uses a root process with default container privileges

**File:** `deploy/docker/compose.yaml:9-81`.

The new init process parses Hermes configuration and performs filesystem mutations as UID 0 with the default Docker capability set. It is a one-shot, but it shares a persistent volume with the Hermes runtime. The task needs ownership changes only; default root privileges are broader than required and a compromised pinned dependency or malformed preexisting volume content would execute in that root context.

**Fix:** Drop all capabilities except those needed for `chown`/`fchown`, set `no-new-privileges`, and make the root filesystem read-only with only the named state volume writable. Exercise initial and existing-volume paths under that profile. If the image requires additional capabilities, document and test the exact set.

## WARNING — Plan contains duplicate dead assignment

**File:** `internal/setup/planner.go:92-93`.

`_ = adopted` appears twice. Remove one; this is a small signal that the new planner change was not diff reviewed.

## Evidence and PR disposition

The parent coordinator independently ran `mise run phase0-check` and an unsandboxed `go test ./... -count=1`, both passing on the dirty tree. The sandboxed Go run failed to bind local test sockets and write its home temp path; those failures were environmental. The Phase 02 evidence records a pinned synthetic Hermes lifecycle journey and one separate Host-mediated NIM result; it does not qualify NIM lifecycle, VPS, physical Termux, macOS login/reboot, two-machine Runs, or release rollback. `docs/PLAN.md:410,468-472` assigns the latter deployment gates to Phase 10. The current PR #22 is draft and its committed head does not yet include this dirty delta.

**PR grouping:** Update PR #22 only with the coherent Phase 02 foundation implementation, its tests, Compose/scripts, Phase 02 evidence and affected canonical documentation after the blocker is fixed and independent security review is acknowledged. Exclude Phases 03–17 plans, new GSD handoff documents, generated runtime state, and unrelated branch changes. Keep the future planning package in a separate stacked PR. A passing Go suite is necessary but insufficient to promote #22 out of draft or claim a release-ready one-command installer.
