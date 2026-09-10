# Changelog

## Unreleased

- Reframed Lumen as a proprietary paid personal intelligence Space with public interoperability protocols: one Jarvis-like conversation, inspectable memory, and capability-scoped action across surfaces and nodes.
- Adopted Hermes-first reuse as a product principle: use Hermes and qualified upstream subsystems before custom development while keeping canonical Space authority, storage, policy, and task truth in Lumen.
- Replaced device-by-device delivery with a capability-first roadmap covering the reliable combined foundation, conversation and memory, secure node fabric, cross-node action, natural voice and presence, messaging continuity, managed portability, earned autonomy, and ecosystem launch.
- Defined combined, separated, managed, and hybrid deployment topologies independently from development, personal, and hardened assurance levels; managed deployments use a dedicated customer data plane.

- Checkpointed the first four Lumen-owned setup tasks: the public command/model contract, deterministic planner and resumable journal, verified artifact lifecycle and Hermes adoption, and strict private Host/Hermes configuration generation. Added an initial cross-platform supervision implementation, while explicitly retaining review debt for create-once Host integration, lifecycle/boot coverage, and mandatory Termux artifact digests. The setup runner, whole-deployment doctor, configured-runtime proof, and release evidence remain pending.

- Added deterministic setup planning and a private resumable setup journal with strict evidence validation, idempotent stage recording, atomic persistence, and honest durability-uncertain outcomes.

- Froze the strict public `lumen setup`, `lumen doctor`, `lumen service start|stop|restart|status`, and reserved `lumen connect` command contract with stable redacted outcomes; orchestration remains intentionally deferred to the next setup block.

- Approved the Lumen-owned setup design and recast Block 2 around `lumen setup`: automatic Lumen/Hermes acquisition and configuration, resumable initialization, boot supervision, whole-deployment diagnostics, and clean-machine evidence. Reserved secure QR/manual node pairing for Block 3 and scheduled Telegram, WhatsApp, and other Hermes-backed integrations behind Lumen capability contracts.
- Elevated Lumen's product goal to a one-command install and instant pairing experience across eligible always-on devices, with QR/manual-code connectivity, device management, boot-started Host/Hermes services, and honest deployment-profile reporting.
- Added `lumen-host doctor` as a local JSON deployment-profile report for install and onboarding flows.

- Corrected the Termux Host distribution target from Linux ARM64/glibc to Android ARM64, so the artifact uses Android's `/system/bin/linker64` instead of the unavailable `/lib/ld-linux-aarch64.so.1` loader.

- Consolidated the Mac development workflow into `lumen-mac-host` for foreground lifecycle management and `lumen-mac-test` for a bounded, automatically policy-approved local Host-to-Hermes AI round trip with durable output.
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
