# Changelog

## Unreleased

### Added

- Defined Space, Host, node, capability, and optional companion as the core product model.
- Added local and cross-node execution paths, configurable context synchronization, capability manifests, and deny/ask/allow grants.
- Added the old Android Host, Mac pet/execution node, and iPhone node as the first reference topology.
- Added gated plans for scheduling, remote access, and explicit Host migration.
- Added one complete V1 delivery plan with phase goals, agent lanes, automated tests, owner checks, exit gates, and requirement coverage.
- Added production Git history, branch, commit, pull-request, review, and release rules.

### Changed

- Reframed Lumen from an iPhone-to-Mac coding workflow into a configurable personal multi-device Space.
- Kept same-device model and tool traffic local while synchronizing permitted context with the Host.
- Made the Host core independent of Hermes because Android/Termux support cannot provide the required always-on reliability boundary.
- Retained isolated, permission-controlled patch application as the safety model for the `coding.run` capability.
- Reconciled requirements, decisions, and delivery plan with the corrected product principle.
- Consolidated the former roadmap and task list into `PLAN.md` to keep delivery guidance in one canonical place.

## 0.1.0 — 2026-08-29

- Added the initial Lumen product definition.
