# Lumen delivery plan

## Product delivery rule

Lumen ships complete capability journeys, not collections of platform features. Each milestone must give the user a useful end-to-end outcome across real trust boundaries and pass acceptance, negative, recovery, privacy, and operational checks before the next milestone becomes active.

The product north star is a Jarvis-like personal intelligence Space: the user can speak naturally from any supported surface, continue one relationship and inspectable memory, and safely act across permitted devices and services. Hermes supplies the intelligence and existing agent subsystems; Lumen owns the Space, authority, continuity, product experience, and commercial service.

## Current execution lock

**Milestone 1 is the only active implementation milestone.** The durable Go Space authority, encrypted store, authenticated local control boundary, Host-to-Hermes Runs execution, and most setup primitives exist. The public `lumen setup`, `lumen doctor`, and `lumen service` paths remain stubs; the release manifest is not distributable; clean-install, reboot, real configured-runtime, and rollback evidence remain incomplete. Node transport, web UI, messaging continuity, voice, canonical conversations, and device capabilities are not implemented.

The current setup branch must be integrated and verified before product-surface implementation begins. Physical Android, Mac, or iPhone completion is not a prerequisite unless it proves the active milestone's contract.

## Evidence format

Every release check records the date, commit, build and dependency versions, topology and assurance profile, preconditions, exact bounded journey, expected and actual result, pass/fail, and a redacted evidence reference. Evidence uses synthetic identities and content and never contains credentials, private prompts, raw audio, or personal device data.

## Status snapshot

| Area | Current truth | Next proof |
| --- | --- | --- |
| Space authority and encrypted persistence | Implemented and covered by Go tests | Preserve while adding conversations, memory, and node protocol |
| Host-to-Hermes run execution | Development adapter and recovery path implemented | Real pinned Hermes compatibility and approval evidence |
| Setup primitives | Planner, journal, artifacts, configuration, adoption, and supervisors implemented | Wire the public orchestration journey |
| Public `lumen` lifecycle CLI | Contract exists; setup, doctor, and service are stubs | Clean combined Linux/VPS deployment |
| Distribution | Cross-builds and templates exist; manifest is a fixture | Reproducible integrity-pinned release artifacts |
| Conversations and memory | Not implemented | Canonical conversation and inspectable-memory nucleus |
| Node fabric and remote capabilities | Domain fixtures only | Authenticated transport plus one restricted capability |
| Product surfaces | Disabled Android shell only | Responsive web conversation surface |
| Jarvis voice and presence | Not implemented | Local activation, streaming speech, barge-in, and handoff |
| Messaging | Designed only | One Hermes gateway mapped into Space conversations |
| Managed service | Not implemented | Dedicated per-Space data plane and portable migration |
| Earned autonomy | Not implemented | Hermes automation behind graduated grants |

## Milestone 0 — Space foundation

**Status:** complete at its intended contract boundary.

The portable Go domain and encrypted store prove Space creation, identity, capability policy, exact approvals, idempotency, revocation, redacted audit, durable transition acknowledgement, and conservative recovery. Historical cross-language and Android experiments remain evidence rather than active product architecture.

**Preserve:** `internal/space`, `internal/store`, public schema discipline, recovery invariants, and negative fixtures.

## Milestone 1 — Reliable combined foundation

**Status:** active.

**Journey:** install a combined Lumen and Hermes deployment on a clean Linux/VPS environment, reboot it, diagnose it, and complete, approve, cancel, and recover real tasks without manual environment plumbing.

**Deliver:**

1. Connect the public `lumen` CLI to the existing planner, journal, artifact, configuration, Host initialization, supervisor, and validation components.
2. Implement truthful `lumen setup`, `lumen doctor`, and `lumen service start|stop|restart|status` behavior.
3. Replace the fixture release manifest with reproducible integrity-pinned Lumen and compatible Hermes artifacts.
4. Certify configured Hermes capabilities and documented Runs behavior rather than treating an HTTP health response as compatibility.
5. Add continuous verification for formatting, vet, unit, race, contract, reproducible build, secret scanning, deployment configuration, and clean setup.
6. Prove interrupted setup, safe rerun, boot restart, failed update, rollback, approval, cancellation, lost event stream, and Host restart recovery.

macOS and Termux remain supported deployment targets, but physical platform evidence does not block the conversation nucleus. Hardened Termux continues to require an isolated Hermes endpoint.

**Exit:** the combined reference deployment reports `ready`, survives reboot, completes the real Host-to-Hermes journey, rolls back a failed update, and preserves canonical state. See [the foundation gate](./PHASE-2-HOST-HERMES.md).

## Milestone 2 — Conversation and memory nucleus

**Journey:** create one Space, talk to Lumen in a responsive web application, retain one useful memory transparently, continue after restart, and inspect or delete what was remembered.

**Deliver:**

1. Add canonical `Conversation`, `Message`, `Participant`, `Surface`, `ContextRecord`, `MemoryProposal`, `RetentionPolicy`, and `ArtifactReference` contracts.
2. Map canonical conversations to replaceable Hermes runtime sessions and stream normalized text, tool, approval, usage, artifact, and terminal events.
3. Project only task-authorized context into Hermes. Accept low-risk memory with provenance and retention; require confirmation for sensitive or consequential memory.
4. Enforce provider, model, region, retention, locality, cost, and data-class constraints before Hermes routing.
5. Build the responsive Lumen experience around Conversation, Memory, Devices and abilities, Activity, and Automations. Hide infrastructure detail until requested.
6. Add the Hermes/upstream component registry with version, provenance, license, data flow, benchmark, health, fallback, upgrade, rollback, and kill-switch evidence.

**Exit:** conversation and memory survive Host and Hermes restarts; runtime replacement does not lose canonical history; deletion affects future context; unauthorized providers receive no classified context.

## Milestone 3 — Secure node fabric

**Journey:** pair a generic node, see its live abilities, disconnect and reconnect it, invoke a harmless capability, then revoke it.

**Deliver:**

1. Publish versioned pairing, identity, capability, invocation, event, cancellation, receipt, and reconciliation schemas plus conformance fixtures.
2. Implement device-generated identity, short-lived QR/manual pairing, explicit owner confirmation, mutually authenticated encrypted sessions, freshness, replay defense, and Host-epoch validation.
3. Let nodes advertise typed health and constraints without granting themselves authority.
4. Make the Host select eligible targets using capability, grant, health, locality, latency, and explicit preference; ambiguity asks the user.
5. Require the target node to revalidate the signed invocation, current local policy, resource scope, expiry, and approval before acting.
6. Support outbound persistent node sessions for NAT-friendly managed and remote use while keeping relays content-blind and non-authoritative.

**Exit:** pairing, reconnect, duplicate delivery, lost acknowledgement, revocation, stale Host, and malicious capability advertisement tests pass across two independent implementations.

## Milestone 4 — Continuity and first cross-node action

**Journey:** continue one conversation across web and a lightweight desktop node, ask Lumen to find a permitted document, summarize it, and deliver the result to another surface.

**Deliver:**

1. Ship the first desktop capability node using the public node contract.
2. Implement `files.search` and `files.read` for owner-selected roots; keep `files.write` a separate later grant.
3. Reject traversal, symlink escape, path replacement, oversized results, stale resources, and access outside declared roots.
4. Route authorized file content to Hermes under the task's context policy and preserve a redacted durable receipt.
5. Add cross-surface handoff, progress, cancellation, approval, and `unknown_outcome` presentation.

**Exit:** the defining continuity-plus-action journey completes within ten minutes of onboarding and fails honestly under node loss, revocation, stale grants, and Hermes interruption.

## Milestone 5 — Jarvis voice and presence

**Journey:** say the local wake phrase, speak naturally, interrupt Lumen while it responds, follow an action across surfaces, and stop listening immediately.

**Deliver:**

1. Standardize `idle`, `wake_detected`, `listening`, `understanding`, `thinking`, `acting`, `approval_needed`, `speaking`, `interrupted`, `offline`, and `degraded` presence states.
2. Keep wake word and voice-activity detection local with zero outbound audio or activation metadata before successful activation.
3. Adopt Hermes voice orchestration and benchmark supported or external STT, TTS, VAD, wake-word, diarization, and interruption components for accuracy, latency, privacy, resources, maintenance, and commercial licensing.
4. Stream post-activation speech under data-aware provider policy, support barge-in, and propagate one cancellation identity through capture, inference, tools, and playback.
5. Synchronize the conversation rather than raw audio by default and make listening state unmistakable on every active surface.

**Exit:** network capture proves pre-activation privacy; latency and transcription benchmarks meet the frozen product target; barge-in, locked device, permission revocation, network loss, and cross-surface handoff behave predictably.

## Milestone 6 — Messaging continuity

**Journey:** connect one Hermes-supported messaging account, continue the same Space conversation there, approve a bounded action, and see the result in the web experience.

**Deliver:**

1. Define one conversation-ingress and delivery contract for provider identity, thread mapping, content, attachments, reply targets, credentials, idempotency, receipts, disconnect, and uncertainty.
2. Adopt one Hermes gateway for the controlled beta, then add further gateways through the same contract.
3. Keep provider mechanics in Hermes while Lumen owns identity mapping, canonical conversation, policy, approval, task state, memory, audit, and connection lifecycle.
4. Treat provider messages and gateway metadata as untrusted content; never let a gateway become an authority path.

**Exit:** web, voice/node, and messaging share one canonical conversation; duplicate delivery does not duplicate effects; disconnect revokes future use; uncertain delivery is visible.

## Milestone 7 — Managed and portable beta

**Journey:** start with one-click managed Lumen, export the encrypted Space, migrate it to a paid self-hosted combined deployment, and optionally move Hermes to a separate endpoint without changing product behavior.

**Deliver:**

1. Build a shared commercial control plane that provisions a dedicated Host, encrypted store, Hermes runtime, credentials, backup, and deletion boundary for each customer.
2. Offer combined, separated, managed, and hybrid topologies under one Space contract; keep topology independent of development, personal, and hardened assurance.
3. Add paid entitlement, support-access approval and expiry, minimal content-free telemetry, regional placement, encrypted backup, export/import, and explicit Host migration.
4. Stage Lumen, Hermes, and upstream updates behind compatibility certification, capability kill switches, health gates, and automatic rollback.

**Exit:** all topologies pass the same conformance suite; managed/self-hosted migration preserves identity, conversations, memory, grants, tasks, and audit; a control-plane compromise cannot read canonical content.

## Milestone 8 — Earned autonomy

**Journey:** Lumen notices a repeatable pattern, proposes an automation, previews its effects, earns a narrow grant, runs it visibly, and lets the user pause, narrow, or revoke it.

**Deliver:**

1. Reuse Hermes cron, goals, heartbeats, loops, delegation, kanban, batch work, and remote execution rather than rebuilding their engines.
2. Graduate trust through `suggest → preview → approve once → approve workflow → narrow automatic grant`.
3. Seal tools, credentials, context, targets, provider policy, budget, deadline, and cancellation lineage into every parent and child runtime profile.
4. Provide one inspectable activity and intervention surface for scheduled, delegated, and background work.

**Exit:** automation survives restart, never expands its grant, stops through the full child lineage, reports uncertain effects honestly, and can be revoked immediately.

## Milestone 9 — Ecosystem and public launch

**Deliver:**

1. Publish node and capability protocols, SDKs, conformance fixtures, compatibility policy, and integration documentation while keeping the Lumen product proprietary.
2. Release certified Hermes-backed capability cohorts: additional gateways, browser/computer use, MCP and skills, vision and media, smart-home/service integrations, and native or ambient surfaces.
3. Qualify every upstream component for provenance, commercial rights, security, maintenance, data flow, performance, fallback, and rollback.
4. Complete billing, entitlement recovery, privacy operations, support, abuse controls, regional operations, and content-free product analytics.

**Launch gate:** real users repeatedly complete the continuity, memory, action, voice, messaging, migration, and earned-autonomy journeys with acceptable onboarding completion, latency, reliability, recovery, privacy, and retention.

## Cross-cutting verification

- **Space conformance:** identity, authority, conversations, memory, grants, tasks, automation, audit, export, and migration.
- **Hermes certification:** capability discovery, Runs behavior, tools, events, approvals, cancellation, delegation, gateways, and supported version upgrades.
- **Node conformance:** pairing, authentication, advertisement, invocation, local revalidation, receipts, reconnect, revocation, and offline recovery.
- **Deployment conformance:** combined, separated, managed, self-hosted, hybrid, and assurance-level behavior.
- **Jarvis experience:** activation privacy, latency, interruption, presence, handoff, degraded behavior, and cross-surface continuity.
- **Upstream qualification:** quality, cost, resources, security, maintenance, licensing, data flow, rollback, and measurable benefit.
- **Privacy canaries:** prompts, logs, telemetry, gateways, plugins, providers, compression, exports, and crash paths reveal no unauthorized content or credentials.

## Coordination rule

Use one coordinator and at most three independent lanes: Space/domain contract, adapter or experience implementation, and verification/security review. Freeze shared schemas before parallel work. Only the coordinator integrates shared protocol, authority, persistence, migration, or threat-model changes.
