# Changelog

## Unreleased

- Selected a native Go Host and Space authority core while retaining Kotlin Android and Swift Apple applications; cross-language schemas and conformance fixtures replace Kotlin Multiplatform implementation sharing.
- Reserved `agent.run/execute` as the first Host-local Hermes orchestration capability with an `ask` default and no transitive authority.
- Quarantined the superseded Android Host prototype: the APK is now a disabled companion shell with no Space-core dependency, canonical-state access, foreground Host service, or foreground-service permissions. Existing encrypted prototype data is left untouched for explicit owner archive or deletion and cannot be imported automatically.
- Completed the native Go Block 2 desktop development slice: added systemd, launchd, Docker, and Termux/runit foreground supervisor definitions; a stable JSON operator CLI; live Hermes schema compatibility; durable bounded AI output; a reproducible AArch64 Termux PIE artifact; and the `phase2-check` gate. On 2026-09-09 a real loopback Hermes task produced AI output through Lumen, cancellation reached durable `cancelled`, restart reconciliation passed, and Android companion unit/APK checks passed. Physical Termux and hardened isolated-Hermes/mTLS release evidence remain pending.

- Corrected the Host boundary: the Host is a headless native Go service supervised by systemd, launchd, Docker, or Termux/runit; Android remains a companion/node application.
- Reordered delivery so the durable Host-to-Hermes execution loop completes before pairing, node transport, or companion work.
- Marked the merged Android foreground-Host implementation as superseded evidence instead of rewriting published history.
- Adopted Lumen authority plus deep Hermes intelligence: Hermes supplies model routing, tools, skills, MCP, browser, voice adapters, delegation, and remote execution behind Lumen-owned capability and task contracts.
- Made invocation reactive by default and removed superseded Android-Host and duplicate design documents from the active tree; their history remains recoverable in Git.

- Implemented the now-superseded Android foreground-Host experiment: Android Keystore AES-GCM state, atomic replacement, owner-driven setup, and a visible foreground service.
- Added and physically verified the experimental Android Host health and stop controls on the Xiaomi reference phone; this remains historical evidence rather than the production Host direction.

- Replaced phase-by-phase feature accumulation with block-based delivery: each block has one live personal journey, failure checks, and an exit gate.
- Defined Hermes as the execution runtime behind Lumen authority, staged `browser.run`, and made memory Host-owned typed context.

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
