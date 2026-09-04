# Lumen V1 Delivery Plan

## End state

V1 delivers one usable personal Space with:

- one active, recoverable Host;
- an old Android phone running the first Host and optional desk companion;
- a Mac node with the pet surface, local execution, and `coding.run` through Hermes;
- an iPhone node with interaction, approvals, notifications, and `reminder.manage`;
- deny/ask/allow capability controls;
- local execution without a Host round trip;
- authenticated cross-node execution through the Host;
- configurable shared-context synchronization;
- immediate and scheduled tasks;
- encrypted local-network access and explicit Host migration;
- recovery, audit, export, and deletion behavior required by `PRD.md`.

## Delivery rules

Every phase follows the same order:

1. Freeze the phase contracts and observable acceptance criteria.
2. Build the smallest end-to-end increment with fakes where needed.
3. Run automated unit, contract, integration, security, and recovery checks.
4. Produce an installable build and a short manual test script.
5. Stop for owner sign-off. A failed manual check reopens the phase.

Do not build later-phase features behind unfinished foundations. Feature count is not progress; a phase exits only with evidence.

## Multi-agent workflow

Use one coordinating agent and no more than three parallel workers:

- **Core lane:** protocol, Space state, policy, task lifecycle, context, and migrations.
- **Platform lane:** Android, macOS, iOS, OS permissions, UI, and lifecycle behavior.
- **Verification lane:** test fixtures, threat cases, compatibility checks, and independent review.

At the start of a phase, the coordinator assigns non-overlapping files and publishes interface fixtures. Workers may implement against those fixtures in parallel. Only the coordinator changes shared schemas after the freeze. At integration, merge the core lane first, rebase adapters onto it, run the entire suite, and have the verification lane review failures and security-sensitive diffs. Never parallelize two implementations of the same state machine or schema.

## Evidence format

Each manual check records: build/version, devices and OS versions, preconditions, steps, expected result, actual result, pass/fail, and a screenshot or redacted log reference. Store no secrets, raw private context, or personal identifiers in evidence.

## Phase 0 — Product and technical foundation (decision baseline complete)

**Outcome:** freeze the smallest production-sensible baseline before executable core work.

### Completed baseline

1. Selected the KMP/native stack and retained native platform boundaries (`D-020`).
2. Defined context defaults, local transport, Android Host limits, manual recovery, and four capability contracts (`D-021`–`D-025`).
3. Added a threat model and paired Kotlin/Swift strict-decoding baseline.
4. Added `mise run phase0-check` as the reproducible Phase 0 validation command.

### Validation evidence

- The Kotlin and Swift candidates both pass valid, expiry, version, unknown-field, time-order, and SSE fixture checks.
- The accepted decision contracts define the negative, recovery, and physical-device checks that gate their implementation phases.

**Decision exit gate met:** the Phase 0 decisions are accepted, the threat model is present, and one command runs the cross-platform contract baseline. Implementation, security, and physical-device evidence remain exit criteria for the phases that implement those components; this is not a claim that those later checks have passed.

## Phase 1 — Executable Space core

**Goal:** prove Space semantics without depending on production UI or platform integrations.

**Status:** in progress. The first slice is defined in [PHASE-1-CONTRACT.md](./PHASE-1-CONTRACT.md).

The in-memory Space core and fake-node scenario now prove creation, pairing, advertisements, default-deny grants, exact approvals, idempotency, revocation, and honest terminal states. Encrypted persistence, key management, and restart reconciliation remain required Phase 1 work.

### Build steps

1. Implement Space creation, owner identity, one-active-Host lease, and encrypted storage.
2. Implement node pairing, key rotation, revocation, capability advertisement, and health.
3. Implement deny/ask/allow policy evaluation with explainable results.
4. Implement the durable task state machine, idempotent commands, permission records, audit events, and restart reconciliation.
5. Create fake Android, Mac, and iPhone nodes and a command-line scenario runner.

### Parallel work

- Core lane owns state, policy, and persistence.
- Platform lane owns fake nodes and scenario tooling.
- Verification lane independently implements protocol-negative and state-transition tests.

### Automated tests

- Pair, revoke, rotate, expire, and reconnect nodes.
- Reject wrong Space, sender, target, capability, scope, signature, nonce, and schema.
- Simulate duplicate delivery, out-of-order events, process death, disk-full failure, and an unknown target outcome.
- Verify an adapter cannot grant itself authority or access unrelated context.

### Owner manual checks

- Run the scenario tool to create a Space, pair three fake nodes, grant one capability, and route a task.
- Change the capability from allow to ask to deny and verify the explanation shown each time.
- Stop and restart the Host mid-task and confirm the final state is honest.

**Exit gate:** the simulated three-node Space completes one authorized task and blocks every negative security fixture with a comprehensible reason.

## Phase 2 — Android Host and desk companion

**Goal:** make the old phone a dependable, understandable Host for local-network use.

### Build steps

1. Implement the native Host service, encrypted store, foreground-service lifecycle, boot recovery, local discovery, authenticated node channel, and health endpoint.
2. Build the companion shell: Space status, node list, current task, permission prompt, text entry, and clear offline/degraded states.
3. Add storage, battery, thermal, network, and update diagnostics.
4. Connect a fake Mac node over the real transport.

### Parallel work

- Core lane integrates persistence and transport with Phase 1 semantics.
- Platform lane builds Android service and companion UI.
- Verification lane owns physical-device lifecycle and resource tests.

### Automated tests

- Service restart, database migration, reconnect, duplicate messages, queue limits, and corrupted-record recovery.
- Authentication and encryption tests across the real node channel.
- UI state tests for healthy, offline, locked, low-storage, and permission-waiting states.

### Owner manual checks

- Create a Space on the old phone without developer tools.
- Reboot the phone and verify the Host returns automatically or gives one clear recovery action.
- Lock the phone, enable battery optimization, change Wi-Fi, unplug/replug power, and leave it idle overnight.
- Submit a task to the fake Mac node and approve or deny it from the companion.

**Exit gate:** the old phone maintains or truthfully reports Host availability through the manual lifecycle matrix and routes a task to the fake Mac after restart and reconnect.

## Phase 3 — Mac node, pet, and coding capability

**Goal:** prove local execution and real cross-node execution on the Mac.

### Build steps

1. Build Mac pairing, capability advertisement, node health, secure Host connection, and local task storage.
2. Build the optional pet surface with text entry, task status, permission prompts, and a full-detail handoff view.
3. Implement the local fast path: direct model/runtime use followed by context-delta synchronization.
4. Implement the Hermes Runs API adapter with version pinning, capability discovery, SSE reconnect, status reconciliation, cancellation, and normalized errors.
5. Implement `coding.run` with isolated Git worktrees, scope validation, patch digesting, review, apply, cleanup, and stale-base handling.

### Parallel work

- Core lane implements context delta and coding safety contracts.
- Platform lane builds the Mac node and pet.
- Adapter lane implements Hermes and workspace isolation.
- Verification begins after the contracts freeze and reviews the integrated result.

### Automated tests

- Hermes fake-server contract tests plus pinned-version compatibility tests.
- Reject path traversal, symlink escape, out-of-scope changes, patch tampering, stale base, repeated approval, and unavailable Hermes.
- Recover from event-stream loss, Hermes crash, cancellation race, Host loss, and partial apply.
- Prove local model/tool traffic is not proxied through the Host.

### Owner manual checks

- Close or disconnect the Host, ask the Mac pet to perform a local coding task, and inspect the local result.
- Reconnect the Host and verify only the configured context level synchronizes.
- From the Android companion, ask the Mac to change a sample project; review, approve, and reject separate patches.
- Tamper with the project before approval and confirm Lumen refuses a stale patch.
- Quit the pet and confirm the node behavior follows the chosen background policy.

**Exit gate:** both local and Android-to-Mac coding journeys work, canonical files change only under the configured policy, and every isolation/recovery test passes.

## Phase 4 — iPhone node and personal capabilities

**Goal:** make the third node useful for interaction, approval, notification, and reminders.

### Build steps

1. Implement pairing, Keychain-backed identity, Host connection, task list, status, approval, and revocation UI.
2. Add App Intents for asking Lumen and checking task status; add voice only through supported system surfaces initially.
3. Implement `notification.deliver` with privacy-safe previews and deep links.
4. Implement `reminder.manage` through a narrow Apple adapter with create, list, complete, and delete actions.
5. Add target-selection UI when multiple nodes expose the same capability.

### Parallel work

- Core lane owns permission and target-selection contracts.
- Platform lane builds iOS UI, App Intents, Keychain, and notifications.
- Adapter lane builds and tests the Apple reminder adapter.
- Verification lane reviews OS permission denial and private-notification behavior.

### Automated tests

- Wrong task, node, actor, arguments, digest, expiry, and reused approval are rejected.
- Reminder actions cannot access unrelated calendars, files, shell, or accounts.
- Notification payloads contain no private content when previews are disabled.
- App Intent requests survive app launch, cancellation, and Host unavailability.

### Owner manual checks

- Pair the iPhone using only the product UI, then revoke and pair it again.
- Ask from iPhone for a Mac coding task and approve the exact result.
- Ask the Android companion to create an iPhone/Apple reminder under allow, ask, and deny policies.
- Deny Contacts/Reminders/notification OS permissions and confirm Lumen explains the limitation.

**Exit gate:** all three reference devices can originate requests, understand status, and enforce capability-specific permission boundaries.

## Phase 5 — Shared context and offline reconciliation

**Goal:** make continuity real without turning context into uncontrolled data collection.

### Build steps

1. Implement typed context namespaces, classification, retention, origin, digest, and versioning.
2. Implement `none`, `metadata`, `summary`, and `content` synchronization levels.
3. Add offline local event journals, delta upload, conflict detection, and type-specific resolution.
4. Build context inspection, deletion, export, and per-capability controls.

### Automated tests

- Verify each synchronization level at the serialized-message and Host-store boundaries.
- Test duplicate, delayed, reordered, conflicting, revoked-node, oversized, and corrupted deltas.
- Verify deletion propagation and retention without losing required audit integrity.

### Owner manual checks

- Perform the same local task at each sync level and inspect exactly what reaches the Host.
- Edit related context on two disconnected nodes, reconnect, and review the conflict experience.
- Export and delete one project’s context; confirm unrelated Space history remains intact.

**Exit gate:** the Host provides useful continuity while tests prove it never receives content beyond the selected synchronization level.

## Phase 6 — Scheduling and capability control

**Goal:** let Lumen work independently inside visible, revocable limits.

### Build steps

1. Implement Host-owned `schedule.manage`: create, list, run, pause, resume, and cancel.
2. Bind schedules to a target capability, grant snapshot/reference, timezone, expiry, retry, offline delivery, and notification policy.
3. Build a plain-language permission and schedule editor with an activity history.
4. Require fresh permission when scope expands or the underlying grant expires.

### Automated tests

- Timezone and daylight-saving transitions, missed runs, clock changes, duplicate triggers, offline targets, retries, revocation, and Host restart.
- Prove a schedule never expands capability authority or silently selects a different target.

### Owner manual checks

- Create, pause, resume, run-now, edit, and cancel a schedule from two different surfaces.
- Turn the target node off across a trigger and confirm the chosen missed-run policy.
- Revoke the capability and verify future runs stop immediately.

**Exit gate:** schedules are predictable, explainable, and unable to exceed current user authority.

## Phase 7 — Remote access, backup, and Host migration

**Goal:** make the Space plug-and-play beyond one LAN without weakening its trust model.

### Build steps

1. Implement the chosen end-to-end encrypted direct or relay transport, push wake-up, delivery receipts, expiry, backpressure, and abuse limits.
2. Keep the relay opaque: it stores no decryption keys, capability grants, or authority.
3. Implement encrypted Host backup, restore, owner recovery, and explicit Host migration.
4. Add a second Host implementation or package on the selected Mac/Linux/VPS target.
5. Enforce a Host epoch/lease so a stale Host cannot accept new work after migration.

### Automated tests

- Compromised relay, replay, reordering, duplicate delivery, long offline periods, key rotation, push loss, NAT changes, and rate limits.
- Migration interruption before and after authority transfer, stale Host recovery, rollback, corrupted backup, and lost recovery key.

### Owner manual checks

- Use cellular data to request work on the Mac at home.
- Disconnect each side at multiple task stages and confirm recovery and status.
- Migrate the Space from the old phone to the selected second Host, verify the old Host is rejected, then test the documented rollback path.
- Restore an encrypted backup on a clean installation.

**Exit gate:** remote work preserves the local permission model, and Host migration cannot create two active authorities.

## Phase 8 — Hardening and private alpha

**Goal:** turn the complete system into a dependable V1 rather than a collection of demos.

### Build steps

1. Complete accessibility, onboarding, updates, diagnostics, resource limits, redaction, privacy controls, and recovery guidance.
2. Pin dependencies and Hermes support, add compatibility and rollback policy, and run a supply-chain review.
3. Run the full requirement traceability matrix and threat-model retest.
4. Package reproducible Android, macOS, and iOS builds for the private alpha.
5. Run at least 20 real tasks over 14 days and record reliability separately from agent-result quality.

### Automated tests

- Full unit, contract, integration, UI, security, migration, performance, soak, and recovery suites on supported OS versions.
- Verify logs, notifications, exports, crashes, and diagnostics contain no unintended secrets or private context.

### Owner manual checks

- Complete clean-device onboarding without development tools or undocumented commands.
- Use every core journey repeatedly during the 14-day trial.
- Perform restart, disconnect, revocation, stale permission, storage pressure, backup, restore, and migration drills.
- Confirm every failure explains what happened, what did not happen, and what action is safe next.

**V1 release gate:** every `PRD.md` requirement has passing evidence; no release-blocking threat remains; the reference Space meets the 14-day success measures; export and deletion work; and the owner explicitly signs off the release checklist.

## Requirement coverage

| Area | Requirements | Primary phase |
| --- | --- | --- |
| Space and Host | `FR-01`–`FR-05` | 0–2, 7 |
| Nodes and capabilities | `FR-10`–`FR-14` | 1–4 |
| Local and cross-node work | `FR-20`–`FR-24` | 2–5 |
| Tasks, approval, context | `FR-30`–`FR-35` | 1, 3–7 |
| Complete V1 validation | All requirements | 8 |
