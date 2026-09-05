# Changelog

## Unreleased

- Completed Phase 1 with a portable durable Host boundary, conservative restart recovery, contract tests, and a fake three-node scenario.
- Completed the existing Space-core file split by restoring command, result, and audit definitions.

- Started Phase 1 with a portable in-memory Space core, contract tests, and a fake Android/Mac/iPhone scenario runner.
- Added `mise run phase1-check` for the core and scenario validation.
- Completed Phase 0 with a KMP/native stack decision, privacy-first context defaults, local-only transport, Android Host limits, manual recovery, and four frozen capability contracts.
- Added `mise run phase0-check` as the reproducible Kotlin/Swift contract baseline.
- Added production system-design rules for contracts, ownership, recovery, privacy, observability, and change control.

### Added

- Added reproducible `O-001` Kotlin Multiplatform and native Swift/JSON Schema spikes with shared strict-decoding and SSE fixtures.
- Added a mise-pinned Java and Gradle environment plus repository-wide build-artifact exclusions.
- Added the V1 threat model with protected assets, trust boundaries, security invariants, release-blocking threats, and required verification evidence.
- Defined Space, Host, node, capability, and optional companion as the core product model.
- Added local and cross-node execution paths, configurable context synchronization, capability manifests, and deny/ask/allow grants.
- Added the old Android Host, Mac pet/execution node, and iPhone node as the first reference topology.
- Added gated plans for scheduling, remote access, and explicit Host migration.
- Added one complete V1 delivery plan with phase goals, agent lanes, automated tests, owner checks, exit gates, and requirement coverage.
- Added production Git history, branch, commit, pull-request, review, and release rules.
- Added risk-based model selection and bounded multi-agent collaboration rules.

### Changed

- Reframed Lumen from an iPhone-to-Mac coding workflow into a configurable personal multi-device Space.
- Kept same-device model and tool traffic local while synchronizing permitted context with the Host.
- Made the Host core independent of Hermes because Android/Termux support cannot provide the required always-on reliability boundary.
- Retained isolated, permission-controlled patch application as the safety model for the `coding.run` capability.
- Reconciled requirements, decisions, and delivery plan with the corrected product principle.
- Consolidated the former roadmap and task list into `PLAN.md` to keep delivery guidance in one canonical place.

## 0.1.0 — 2026-08-29

- Added the initial Lumen product definition.
