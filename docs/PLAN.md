# Lumen V1 delivery plan

## How delivery works

Lumen is built in blocks, not as disconnected features. Each block starts with one useful personal journey, a frozen contract, automated checks, and a live owner test. It ends only when the journey works repeatedly on the intended devices and its failure path is honest.

Do not begin a later block because its UI can be mocked. Fix or simplify the current block when recovery, approval, privacy, or daily-use evidence fails. Record build, devices, preconditions, expected and actual result, pass/fail, and a redacted evidence reference for every owner check.

## Current execution lock

**Block 2 remains the only active delivery block. Setup Tasks 1–4 are implemented, and Task 5 has a committed supervision checkpoint, but the setup runner, whole-deployment diagnostics, configured-runtime proof, clean-machine evidence, physical Termux proof, and isolated-Hermes mutual-TLS proof remain. Task 5 also retains review debt around concrete create-once integration, lifecycle coverage, and mandatory Termux artifact digests. Block 2 may reserve and test the `lumen connect` command handoff, but authenticated pairing, node transport, companions, and device capabilities remain blocked until its exit gate passes.** See [PHASE-2-HOST-HERMES.md](./PHASE-2-HOST-HERMES.md).

## Evidence format

Every owner check records the date, build or commit, intended device and runtime versions, preconditions, exact bounded action, expected and actual result, pass/fail, and a redacted evidence reference. Evidence must use synthetic identifiers and content; it must never contain credentials, private prompts, raw audio, or personal device data. A passing desktop development harness is labeled as development evidence and cannot satisfy a hardened or physical-device release gate.

## Status snapshot

| Area | Status | Next proof |
| --- | --- | --- |
| Space authority and encrypted persistence | Complete | Preserve contracts while setup calls create-once initialization. |
| Host-to-Hermes Runs execution | Desktop development passing | Isolated mutual-TLS run with real approval behavior. |
| Public `lumen` command contract | Complete | Preserve strict parsing and redacted outcomes as orchestration lands. |
| Resumable setup planner and journal | Complete | Bind evidence to the setup runner without weakening resume checks. |
| Artifact verification and Hermes adoption | Complete | Bind the verified manifest to orchestration and clean-machine evidence. |
| Generated Host/Hermes configuration | Complete | Validate the pinned Hermes schema through the configured-runtime proof. |
| Host and Hermes supervision | Checkpoint committed; review debt remains | Bind real create-once Host initialization, require Termux digests, close lifecycle tests, then prove reboot. |
| `lumen-host doctor` | Partial local metadata | Public whole-deployment `lumen doctor` with truthful outcomes. |
| macOS | Development harness passing | One-command setup and reboot evidence. |
| Linux/VPS and old PC | Templates/build passing | Hardened clean-machine setup and reboot evidence. |
| Android Termux | Artifact and installer contracts passing | Physical setup/boot proof; hardened proof uses isolated Hermes. |
| QR/manual node pairing | Not started | Block 3 single-use offer and authenticated transport. |
| Telegram/WhatsApp and broader Hermes features | Designed, not exposed | Block 5 typed capabilities and integration lifecycle. |

## Block 1 — Space foundation

**Status:** complete. The portable Space core proves creation, pairing, capability policy, exact approvals, idempotency, revocation, redacted audit, durable transition acknowledgment, and conservative restart recovery. See [PHASE-1-CONTRACT.md](./PHASE-1-CONTRACT.md).

**Live proof:** create a three-node fake Space, grant and revoke a capability, route an authorized task, and restart while work is queued.

## Block 2 — Lumen-owned Host and Hermes setup

**Status:** active checkpoint. The native Go Host, encrypted store, operator boundary, Hermes Runs adapter, public `lumen` contract, deterministic planner/journal, verified artifact lifecycle, strict generated configuration, and initial cross-platform Host/Hermes supervision are committed. The Task 5 review breaker left explicit debt: concrete create-once integration, fuller lifecycle/boot tests, and mandatory manifest-bound Termux digests. Tasks 6–8—the complete orchestrator, whole-deployment `lumen doctor`, configured-runtime proof, automated release gate, and owner evidence—have not started.

**Goal:** one `lumen setup` command turns a supported environment into a configured, boot-started, verified Host plus Hermes deployment with no manual environment-file plumbing, then completes one bounded Hermes task under Lumen authority.

**Completed foundation:** Android authority quarantine → Go Space authority → encrypted durable Host → authenticated operator boundary → constrained Hermes Runs adapter → durable events/approval/cancellation/recovery → supervisor templates → reproducible Android Termux artifact.

**Completed first slice:** the Android APK no longer depends on `core:space`, creates or opens canonical Space state, declares a foreground Host service, or requests foreground-service permissions. It remains installable only as a disabled companion shell until Block 4 pairing work begins. Existing encrypted prototype state is ignored and must be explicitly archived or cleared by its owner; it is never imported into the production Host.

**Execution sequence:**

1. **2.1 Public setup contract — complete:** owner-facing `lumen setup`, `lumen doctor`, `lumen service`, and reserved `lumen connect` shapes are frozen; `lumen-host init|serve` remains internal.
2. **2.2 Common planner — complete:** deterministic platform/profile planning and a private resumable journal are implemented without mutation or silent downgrade.
3. **2.3 Artifacts and configuration — complete:** pinned manifest parsing, staged verification/replacement, strict Hermes adoption, private credentials, and Host/Hermes configuration generation are implemented.
4. **2.4 Platform supervision — checkpoint:** systemd, launchd, Docker, and Termux/runit definitions and adapters are committed; the residual review debt above must be closed through the real setup runner before this slice is accepted.
5. **2.5 Whole-deployment doctor:** report Host state, Hermes health/capabilities, versions, supervisor and boot state, credential readiness, isolation level, and exact next action as `ready`, `degraded`, or `action_required`.
6. **2.6 Release proof:** on clean environments, run setup, rerun it, interrupt/resume it, reboot, run doctor, and complete/approve/cancel/recover a real task. Record macOS development, Linux/VPS hardened, Android Termux compatibility, and Termux-to-isolated-Hermes evidence separately.

**Detailed task plan:** [Lumen-owned setup implementation plan](./superpowers/plans/2026-09-10-lumen-owned-setup.md).

**Live proof:** on a clean supported environment run `lumen setup` with no manual Hermes edits, verify `lumen doctor`, reboot, verify both services, submit one bounded task, resolve one exact approval, cancel one run, interrupt another, restart, and confirm a proven terminal state or `unknown_outcome`.

**Exit:** [the Phase 2 automated and owner gates](./PHASE-2-HOST-HERMES.md#exit) pass for the setup journey. The `lumen connect` shape may be reserved, but no external node is trusted before Block 3.

**Evidence (2026-09-09):** `ANDROID_HOME=/Users/ashwanthreddyboddireddy/Library/Android/sdk GOCACHE=/private/tmp/lumen-go-cache mise run phase2-check` passed formatting, vet, all Go and race tests, native and Linux ARM64 builds, ELF64 AArch64 PIE validation, supervisor/Compose contracts, Android companion unit tests, and debug APK assembly. A separately supervised loopback Hermes gateway completed quote task `manual-hermes-quote-001`; Lumen retained its final AI output in the durable task record. Cancellation task `manual-hermes-cancel-001` progressed through `cancelling` to durable `cancelled`. Earlier task `manual-agent-dryrun-004` reconciled to `completed` after a forced Host restart. The gateway auto-approved shell probes, so a live runtime approval under deny-by-default Hermes remains pending even though the exact approval path passes adapter and Host contract tests. Hardened isolated-Hermes/mTLS and physical Termux lifecycle proof also remain pending; the phone was not visible to ADB when artifact transfer was attempted.

## Block 3 — Paired-node protocol and local transport

**Status:** not started. Space membership/revocation transitions and schema experiments exist, but there is no pairing server, QR/manual-code exchange, authenticated node channel, reconnect flow, or device-management command.

**Goal:** let the proven Host authenticate and coordinate one external node without weakening its local authority contract.

**Tasks:** pairing offer and QR/manual representation → node-generated identity → explicit owner confirmation → signed encrypted transport → replay/duplicate/lost-ack handling → reconnect and revocation → `lumen device list|status|grant|revoke`.

**Checks:** wrong identity or Space, stale epoch, expiry, replay, duplicate collision, reordered delivery, revocation, Wi-Fi loss, reconnect, and lost acknowledgements.

**Live proof:** install on one additional device, pair one test node by QR and once by manual code, disconnect and reconnect it, route one harmless command, replay the envelope, revoke the node, and prove future work is rejected.

**Exit:** the authenticated-node scenario and independent protocol/security review pass without relying on discovery as trust.

## Block 4 — Android companion and device capabilities

**Goal:** make the old Android phone a useful desk companion and capability node for the already-running Host.

**Build:** finish the Android companion module/package rename after the earlier authority quarantine; add node identity, pairing, Host connection, task and approval views, and reactive local voice. Pre-activation wake-word/VAD runs entirely on the node with Hermes client-capture disabled and zero outbound audio; microphone, camera, and speaker remain separate foreground capabilities with explicit Android permission and local policy checks.

**Live proof:** install with `adb install -r`, pair with the Host, follow the Host/Hermes task from the desk display, deny and grant each hardware capability separately, force-stop the app, and prove the Host continues running.

**Exit:** companion removal or failure cannot change Host authority; permission-denial and reconnect paths are honest on the Xiaomi reference phone.

## Block 5 — Hermes capability frontier and private daily loop

**Goal:** make the Host/Hermes system useful daily without creating a surveillance archive or an unsupervised account operator.

**Build:** typed Host-owned context with provenance, retention, expiry, inspect/edit/delete/export controls; import approved Hermes tools, skills, MCP servers, browser automation, model routing, post-activation voice, delegation, remote execution, Telegram, and WhatsApp into the Lumen capability registry. Expose them through `lumen integration list|connect|disconnect|status`; provider authentication remains explicit and provider QR codes are never confused with node pairing. Begin `browser.run` read-only, then gate draft and submit separately. Messaging connectors receive account, recipient, content, attachment, credential, approval, idempotency, delivery-receipt, unknown-outcome, and disconnect contracts.

**Checks:** discovery cannot self-enable a capability; a runtime cannot persist memory or expand grants; each profile contains only approved tools, credentials, models, integrations, and remote backend; account disconnect prevents later use; messaging retries do not duplicate effects where the provider supports idempotency; uncertain delivery is never reported as success; browser and delegation restrictions remain fail-closed.

**Live proof:** complete repeated research tasks, inspect and delete retained context, block one domain, reject one stale approval, submit one harmless approved action, and interrupt another.

**Exit:** ten useful tasks complete; denied synchronization stays absent; every attempted side effect has approval, receipt, and an honest final or unknown outcome.

## Block 6 — Mac coding node and companion

**Goal:** safely use Hermes for bounded coding work on the Mac.

**Build:** Mac pairing and health; isolated Git worktree execution; `coding.run` scope and patch-digest validation; review, apply, cleanup, and stale-base handling.

**Live proof:** give Lumen a small task in this repository, review the patch, alter the base before approval, and verify stale application is refused.

**Exit:** every canonical-project change is reviewed under capability policy and no direct `main` write occurs.

## Block 7 — iPhone interaction and reminders

**Goal:** make the iPhone a useful approval, notification, and reminder node.

**Build:** pairing, Keychain-backed identity, Host connection, reactive text and voice, task and approval views, notifications, App Intents, and `reminder.manage`.

**Exit:** OS permission denial and Host unavailability have clear, testable behavior.

## Block 8 — schedules, remote use, and private alpha

**Goal:** make Lumen resilient enough for continued personal use before inviting others.

**Build:** schedules; offline reconciliation; encrypted export, restore, and explicit Host migration; remote encrypted transport; observability and privacy review.

**Live proof:** use the Android, Mac, and iPhone Space for 20 real tasks over 14 days. Test revocation, restart, offline recovery, migration, export, and deletion.

**Exit:** the V1 gate in [PRD.md](./PRD.md) passes.

## Coordination rule

For independent work, use a coordinator plus at most three lanes: portable core, platform adapter, and verification. Freeze shared schemas first. Only the coordinator changes shared schemas during integration; verification independently reviews protocol, authorization, persistence, migration, and sandbox changes.

## Hermes adoption rule

Hermes is the preferred implementation for reasoning, model routing, tools, skills, MCP, browser, post-activation voice adapters, delegation, remote execution, and messaging integrations including Telegram and WhatsApp. Lumen owns installation, public commands, credentials, policy, task state, approvals, redacted status, audit, and lifecycle. Discovery never grants authority. Block 2 installs and validates Hermes but exposes only the frozen common run boundary; later blocks enable each feature behind a typed Lumen capability, immutable verified runtime profile, negative tests, live evidence, and rollback.
