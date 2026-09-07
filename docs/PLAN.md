# Lumen V1 delivery plan

## How delivery works

Lumen is built in blocks, not as disconnected features. Each block starts with one useful personal journey, a frozen contract, automated checks, and a live owner test. It ends only when the journey works repeatedly on the intended devices and its failure path is honest.

Do not begin a later block because its UI can be mocked. Fix or simplify the current block when recovery, approval, privacy, or daily-use evidence fails. Record build, devices, preconditions, expected and actual result, pass/fail, and a redacted evidence reference for every owner check.

## Current execution lock

**The first task in the next session is to finish Block 2: the headless Host service and its real Hermes execution loop. Begin with the one-time Android authority quarantine required by that block; otherwise do not resume pairing, node transport, Android companion features, hardware capabilities, Mac, or iPhone implementation until the Block 2 exit gate passes.** See [PHASE-2-HOST-HERMES.md](./PHASE-2-HOST-HERMES.md).

## Block 1 — Space foundation

**Status:** complete. The portable Space core proves creation, pairing, capability policy, exact approvals, idempotency, revocation, redacted audit, durable transition acknowledgment, and conservative restart recovery. See [PHASE-1-CONTRACT.md](./PHASE-1-CONTRACT.md).

**Live proof:** create a three-node fake Space, grant and revoke a capability, route an authorized task, and restart while work is queued.

## Block 2 — Headless Host and Hermes execution loop

**Status:** next and blocking. The portable Space core is ready, but no production headless Host, durable JVM store, operator CLI, supervisor integration, or Hermes adapter exists. The Android foreground-Host experiment is superseded evidence, not a base for more features.

**Goal:** run the canonical Host as a terminal service and complete one bounded Hermes task with honest authority, persistence, events, approval, cancellation, failure, and restart behavior.

**Execution sequence:** superseded Android authority quarantine → Host-local execution contract → Host process and CLI → encrypted durable store → authenticated Hermes adapter → run/events → approval and cancellation → recovery and reconciliation → supervisor parity → Android Termux proof.

**Build:** one Kotlin/JVM 21 `lumen-host` distribution; foreground `serve` command; owner-restricted authenticated operator socket; encrypted, locked, atomic Host state; redacted health; a versioned Hermes Runs API adapter; and systemd, launchd, Docker, and Termux/runit examples.

**Live proof:** initialize and supervise the Host, submit a task from its CLI, observe a real Hermes event stream, resolve one exact approval, cancel one run, interrupt another, restart, and confirm the Host records a proven terminal state or `unknown_outcome`. Repeat the service lifecycle and one task in Android Termux.

**Exit:** [the Phase 2 automated and owner gates](./PHASE-2-HOST-HERMES.md#exit) pass. Host/Hermes completion is required before any node work.

## Block 3 — Paired-node protocol and local transport

**Goal:** let the proven Host authenticate and coordinate one external node without weakening its local authority contract.

**Build:** Host and node signing identities; QR/SAS pairing; durable membership; versioned signed envelopes; mTLS; replay and duplicate protection; mDNS as discovery only; revocation; and reconnect reconciliation.

**Checks:** wrong identity or Space, stale epoch, expiry, replay, duplicate collision, reordered delivery, revocation, Wi-Fi loss, reconnect, and lost acknowledgements.

**Live proof:** pair one test node, disconnect and reconnect it, route one harmless command, replay the envelope, revoke the node, and prove future work is rejected.

**Exit:** the authenticated-node scenario and independent protocol/security review pass without relying on discovery as trust.

## Block 4 — Android companion and device capabilities

**Goal:** make the old Android phone a useful desk companion and capability node for the already-running Host.

**Build:** finish the Android companion module/package rename after the earlier authority quarantine; add node identity, pairing, Host connection, task and approval views, and local text-to-speech. Microphone and camera remain separate foreground capabilities with explicit Android permission and local policy checks.

**Live proof:** install with `adb install -r`, pair with the Host, follow the Host/Hermes task from the desk display, deny and grant each hardware capability separately, force-stop the app, and prove the Host continues running.

**Exit:** companion removal or failure cannot change Host authority; permission-denial and reconnect paths are honest on the Xiaomi reference phone.

## Block 5 — Private daily loop, context, and browser actions

**Goal:** make the Host/Hermes system useful daily without creating a surveillance archive or an unsupervised account operator.

**Build:** typed Host-owned context with provenance, retention, expiry, inspect/edit/delete/export controls; read-only `browser.run` research on allowlisted public sites; then separately gated `draft` and `submit` with exact preview, one-time approval, receipt, cancellation, and uncertain outcomes.

**Checks:** a runtime cannot persist memory or expand grants; deletion and expiry are durable; blocked domains, changed targets, credential entry, upload, download, payment, and 2FA fail closed or ask again; retries do not duplicate a submission.

**Live proof:** complete repeated research tasks, inspect and delete retained context, block one domain, reject one stale approval, submit one harmless approved action, and interrupt another.

**Exit:** ten useful tasks complete; denied synchronization stays absent; every attempted side effect has approval, receipt, and an honest final or unknown outcome.

## Block 6 — Mac coding node and companion

**Goal:** safely use Hermes for bounded coding work on the Mac.

**Build:** Mac pairing and health; isolated Git worktree execution; `coding.run` scope and patch-digest validation; review, apply, cleanup, and stale-base handling.

**Live proof:** give Lumen a small task in this repository, review the patch, alter the base before approval, and verify stale application is refused.

**Exit:** every canonical-project change is reviewed under capability policy and no direct `main` write occurs.

## Block 7 — iPhone interaction and reminders

**Goal:** make the iPhone a useful approval, notification, and reminder node.

**Build:** pairing, Keychain-backed identity, Host connection, task and approval views, notifications, App Intents, and `reminder.manage`.

**Exit:** OS permission denial and Host unavailability have clear, testable behavior.

## Block 8 — schedules, remote use, and private alpha

**Goal:** make Lumen resilient enough for continued personal use before inviting others.

**Build:** schedules; offline reconciliation; encrypted export, restore, and explicit Host migration; remote encrypted transport; observability and privacy review.

**Live proof:** use the Android, Mac, and iPhone Space for 20 real tasks over 14 days. Test revocation, restart, offline recovery, migration, export, and deletion.

**Exit:** the V1 gate in [PRD.md](./PRD.md) passes.

## Coordination rule

For independent work, use a coordinator plus at most three lanes: portable core, platform adapter, and verification. Freeze shared schemas first. Only the coordinator changes shared schemas during integration; verification independently reviews protocol, authorization, persistence, migration, and sandbox changes.
