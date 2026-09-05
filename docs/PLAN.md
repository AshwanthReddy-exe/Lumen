# Lumen V1 delivery plan

## How delivery works

Lumen is built in blocks, not as disconnected features. Each block starts with one useful personal journey, a frozen contract, automated checks, and a live owner test. It ends only when the journey works repeatedly on the intended devices and its failure path is honest.

Do not begin a later block because its UI can be mocked. Fix or simplify the current block when recovery, approval, privacy, or daily-use evidence fails. Record build, devices, preconditions, expected and actual result, pass/fail, and a redacted evidence reference for every owner check.

## Block 1 — Space foundation

**Status:** complete. The portable Space core proves creation, pairing, capability policy, exact approvals, idempotency, revocation, redacted audit, durable transition acknowledgment, and conservative restart recovery. See [PHASE-1-CONTRACT.md](./PHASE-1-CONTRACT.md).

**Live proof:** create a three-node fake Space, grant and revoke a capability, route an authorized task, and restart while work is queued.

## Block 2 — Android Host

**Goal:** make the old Android phone a dependable local-network Host.

**Build:** native foreground Host service; Android Keystore-backed encrypted store; boot and locked-store recovery; paired-node discovery and authenticated transport; Space health; and a companion showing status, task, node, and approval state.

**Live proof:** create a Space on the phone, pair a fake Mac, reboot, change Wi-Fi, unplug/replug power, and route one task after reconnect.

**Exit:** five lifecycle runs pass without manual state repair.

## Block 3 — Hermes daily loop and read-only browser research

**Goal:** make Lumen useful every day without web side effects.

**Build:** a versioned Hermes Runtime Adapter; a task feed and audit view on the Mac; `browser.run` limited to `research`, `navigate`, and `extract`; an allowlisted public-site policy; and source artifacts.

**Checks:** Hermes unavailable, cancellation, event-stream loss, duplicate events, untrusted page content, blocked domain, and expired grant. Browser output never changes policy or becomes trusted context automatically.

**Live proof:** use Lumen for a daily research brief for five days and inspect sources, status, and audit events.

**Exit:** five successful runs, no unauthorized browser action, and no manual task-state repair.

## Block 4 — User-controlled memory

**Goal:** preserve useful continuity without an uncontrolled agent memory store.

**Build:** typed `ContextRecord` proposals for preferences, project facts, decisions, and task summaries; scope, source, classification, retention, expiry, confidence, and status; approval, inspection, edit, deletion, and export.

**Checks:** a runtime cannot persist a memory directly; deletion and expiry are durable; revoked nodes lose access; browser text, credentials, secrets, and raw transcripts are rejected by default.

**Live proof:** approve a preference and project fact, use them later, inspect provenance, delete one, and confirm it cannot be used.

**Exit:** seven days of use with a weekly memory audit.

## Block 5 — Approved browser actions

**Goal:** allow a small, reviewable web action without handing browser authority to a runtime.

**Build:** `browser.run` actions for `draft` and `submit`; isolated profiles; exact action preview; one-time action-bound approval; receipt capture; cancellation and uncertain outcome behavior.

**Checks:** profiles cannot expand grants; changed targets, upload, download, payment, credential entry, or 2FA block or ask again; retries do not duplicate a submission.

**Live proof:** create a draft in a low-risk test account after reviewing its target and payload.

**Exit:** every attempted action has approval, receipt, and an honest final or unknown outcome.

## Block 6 — Mac coding companion

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
