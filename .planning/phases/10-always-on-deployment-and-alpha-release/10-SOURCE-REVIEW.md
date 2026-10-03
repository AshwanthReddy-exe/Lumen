# Source-grounded plan review — Phase 10

Status: **OPEN; not cross-AI converged.** Independent CLI review produced no result; this is an inline source review.

## 10-01

- **HIGH:** The installer must verify bytes “before running anything” (`10-01-PLAN.md:19`), yet the proposed one-command shell bootstrap must execute before it can perform verification. Define the trust anchor and exact public command, plus how the bootstrap itself is authenticated. `internal/setup/release.go:16-21` already separates immutable setup binding from release generation; reuse that contract.

## 10-02

- **MEDIUM:** Linux and macOS live reboot checks are correctly gated (`10-02-PLAN.md:19-23`), but the claimed OS/arch matrix needs explicit entries and selected test machines before this phase can close. Keep unsupported entries out of install advertising.

## 10-03

- **MEDIUM:** The second signed generation and offline rollback (`10-03-PLAN.md:19`) must preserve the existing `ReleaseRecord` generation and immutable binding (`internal/setup/release.go:39-48,74-98`). Name this existing type in the plan rather than letting an executor add a parallel release record.

## 10-04

- **HIGH:** A restored Host cannot unilaterally prevent a partitioned old Host from accepting local writes. `10-04-PLAN.md:20` must define the fencing authority and enrollment/epoch protocol at each node, and state the manual quiescence precondition. Prove old Host return and offline node behavior before claiming one active authority.
- **HIGH:** Backup scope includes identity, grants, tasks, cursors (`10-04-PLAN.md:19`) but owns only `cmd/lumen/backup.go` and `restore.go`. Assign the canonical storage snapshot API and its consistent watermark to a concrete package/file; a CLI-only archive risks inconsistent multi-store state.

## 10-05

- **MEDIUM:** Seven-day soak and owner sign-off are explicit (`10-05-PLAN.md:20-21`); preserve BLOCKED status until actual elapsed duration and named physical acceptance exist. Do not let an automated `Test.*Soak` pass stand in for the duration.

CYCLE_SUMMARY: unresolved HIGH 3; actionable MEDIUM 3; LOW 0. Revise plans, then run independent `gsd-review` again.
