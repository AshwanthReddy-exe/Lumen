# Changelog

## Unreleased

- Added a pinned-Hermes chat configuration candidate and a preflight for resolved ordinary API toolsets and constructed agent memory/tool state. Live Runs, room policy overrides, session memory, and artifact/process binding remain qualification gates.
- Verified pinned Hermes `v2026.9.7` plugin hooks fail open on callback exceptions and corrected the zero-tool conversation plan: production certification now requires a dedicated qualified runtime profile and negative tool/memory probes; chat stays blocked meanwhile.
- Added conversation-specific Host restart reconciliation: durable run and certificate binding, fresh endpoint certification, bounded evidence checks, one canonical assistant append, truthful replay, and unknown outcomes for missing or unsafe evidence.
- Prevented generic Host-run reconciliation, task completion, cancellation, and runtime approval from mutating conversation tasks; verified that Host restart leaves an interrupted chat run uncertain without appending an assistant message.
- Scanned Hermes conversation events even after a terminal run status, rejected unknown or duplicate event payload fields and conflicting terminal evidence, and kept disconnected streams without terminal evidence uncertain. Added a redacted Host certification-invalidation record for forbidden events and verified session reservation through encrypted-store reopen.

- Merged the existing Host-owned conversation nucleus into an isolated continuation branch and strengthened certificate binding, accepted-preference projection, reconciliation deadlines, durable pre-certification session reservation, unexpected-event rejection, and replayed task receipts with regression tests. The conversation path is not release-ready: real runtime certification, violation invalidation, and restart-aware reconciliation remain open review gates. Clarified the one-agent-across-devices direction and Future OS reference boundary in the canonical plan and decisions.
- Verified both Docker topology lifecycle paths on Azure Ubuntu 22.04 amd64. The combined pinned-Hermes gate passed twice clean and survived a real reboot with stable canonical state. External public setup adopted independently supervised Hermes `0.21.1` through pinned mutual TLS, controlled exactly the Host, survived another real reboot with unchanged service identities and state, and returned a ready doctor result. Added a repeatable external journey gate that passed twice clean. Milestone 1 remains open for distributable immutable release assets, public live artifact replacement/rollback, residual pinned-runtime probes, and cross-machine external Runs evidence.
- Completed the package-level dual-topology setup foundation: durable combined/external bindings, authenticated live doctor evidence, topology-scoped service control, safe legacy binding migration, configured Hermes Runs recovery certification, and an official Hermes `v2026.9.7` source/archive pin with a real frozen Docker build. Milestone 1 remains open for the distributable manifest and real combined/external Linux, process-death, reboot, and rollback evidence.
- Reconciled the canonical product, architecture, decisions, threat model, and delivery plan with the approved Lumen-on-Hermes boundary: one Host authority, one active Hermes runtime, a Host-owned feature registry, immutable profiles and task capabilities, separate `HermesRuntimeAdapter v1` and narrow Lumen Hermes plugin seams, canonical conversation/session/context/`MemoryProposal` rules, and Host-selected nodes with node-local revalidation. Milestone 1 remains active because the distributable manifest and real-machine, reboot, and rollback evidence are still absent; later Hermes surfaces and M2–M9 journeys remain target behavior, not shipped claims.
- Defined the Milestone 1 dual-topology contract: combined Docker owns both Lumen and Hermes lifecycle, while external adopts an independently managed Hermes endpoint and controls only the Host. Both paths share Host task semantics and truthful `ready`, `degraded`, and `action_required` outcomes.
- Reframed Lumen as a proprietary paid personal intelligence Space with public interoperability protocols: one Jarvis-like conversation, inspectable memory, and capability-scoped action across surfaces and nodes.
- Adopted Hermes-first reuse as a product principle: use Hermes and qualified upstream subsystems before custom development while keeping canonical Space authority, storage, policy, and task truth in Lumen.
- Replaced device-by-device delivery with a capability-first roadmap covering the reliable combined foundation, conversation and memory, secure node fabric, cross-node action, natural voice and presence, messaging continuity, managed portability, earned autonomy, and ecosystem launch.
- Defined combined, separated, managed, and hybrid deployment topologies independently from development, personal, and hardened assurance levels; managed deployments use a dedicated customer data plane.

- Earlier checkpoint: added the first four Lumen-owned setup tasks—the public command/model contract, deterministic planner and resumable journal, verified artifact lifecycle and Hermes adoption, strict private configuration, and initial cross-platform supervision—while the setup runner, whole-deployment doctor, configured-runtime proof, and release evidence were still pending.

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
- Historical: added the retired `mise run phase1-check` command for the Kotlin core and scenario validation; current Go authority validation is recorded in `docs/PHASE-1-CONTRACT.md`.
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
