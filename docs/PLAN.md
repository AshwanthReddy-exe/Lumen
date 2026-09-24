# Lumen delivery plan

## Product delivery rule

Lumen ships complete capability journeys, not collections of platform features. Each milestone must give the user a useful end-to-end outcome across real trust boundaries and pass acceptance, negative, recovery, privacy, and operational checks before the next milestone becomes active.

The product north star is a Jarvis-like personal intelligence Space: the user can speak naturally from any supported surface, continue one relationship and inspectable memory, and safely act across permitted devices and services. Hermes supplies the intelligence and existing agent subsystems; Lumen owns the Space, authority, continuity, product experience, and commercial service.

## One-agent delivery sequence

Future OS at commit `98f7f3a` is a reference for inspectable work, approachable setup, desktop/mobile continuity, remote control, and release presentation. It is not Lumen's state authority. [D-053](./DECISIONS.md) keeps Hermes as the preferred execution runtime while preserving the option to qualify another runtime behind the same Host contract. Direct code reuse requires per-component license, attribution, dependency, and security review; copying an unrestricted tool policy or process-local approval route is prohibited.

Apply its concrete patterns where they fit: `agent/src/session/run_journal.rs` records start and terminal markers for honest restart reporting; Lumen should express those markers through its encrypted Host task ledger. `agent/src/session/tools.rs` projects tool status from durable records; Lumen should expose a bounded, redacted Host-owned activity timeline after capability checks. Its staged model setup and doctor flow inform the first-run journey. The reference is mixed-license (root MIT; `orchestration/loop` Apache-2.0 with NOTICE), so review each copied component separately.

The shortest useful commercial path is one visible continuity journey, then broader capabilities:

1. **Foundation:** finish the distributable manifest, live replacement/rollback, cross-machine external Runs, and pinned-runtime negative probes. Milestone 1 stays open until those real checks pass.
2. **One conversation:** resolve every Important finding in the existing conversation nucleus review, wire a real certified zero-tool Hermes profile, and prove one Host-owned history from two reconnectable clients after Host and Hermes restart. No assistant message follows an uncertain run.
3. **One action across devices:** pair one node, grant one selected file root, ask from another surface, revalidate at the node, and show an exact receipt or an honest uncertain outcome. Revoke access and prove the next attempt fails.
4. **One natural relationship:** add a messaging surface and then voice to the same conversation; make listening, progress, interruption, and approval visible. Do not create channel-specific memory.
5. **Commercial readiness:** package self-hosted and managed deployment with equivalent Space semantics, encrypted export/import, billing, support boundaries, backups, rollback, and user-tested onboarding. A compelling demo and concise product promise come from these working journeys, not from listing future features.

For each step, acceptance requires a clean-install journey, negative and restart tests, real intended-platform evidence, and an independent security review for authority, persistence, transport, or migration changes. Track time to the first useful continuity-and-action moment, setup completion, recovery success, latency, and repeat use before expanding the feature catalog.

The existing conversation-nucleus branch has six Important review findings. This branch has test-backed changes for certificate binding (I1), preference projection (I3), deadline rejection (I5), durable session reservation before certification I/O (I4), and forbidden-event invalidation (I2). The I2 change passed independent review and encrypted-store reopen testing. Restart reconciliation (I6) now has a bounded Host path that recovers a mapped run only with fresh matching certification and validated events; unmapped, expired, mismatched, and oversized outcomes remain uncertain without redispatch. The conversation gate remains **open** until this recovery change passes independent review, a real certifier is wired, and the two-client, Host-restart, Hermes-restart, and zero-tool journeys pass on intended deployments. No product-surface milestone is complete merely because package tests pass.

## Current execution lock

**Milestone 1 foundation closure and the bounded Milestone 2 conversation nucleus are the active implementation lanes.** The durable Go Space authority, encrypted store, authenticated local control boundary, Host-to-Hermes Runs execution, dual-topology public setup, and durable doctor/service paths exist. The combined gate passed twice clean on Azure Ubuntu 22.04 amd64 and survived a real reboot. External setup against an independently supervised, pinned Hermes runtime passed authenticated mutual-TLS adoption, Host-only lifecycle control, doctor, canonical-state preservation, and a real reboot on that same VPS; the automated external journey also passed twice clean with an isolated compatibility endpoint. The release manifest is still not distributable, live deployment artifact replacement/rollback is absent, and real cross-machine external Runs behavior remains unproven. M1 therefore remains open and cannot be inferred complete.

The owner authorized the conversation nucleus on 2026-09-15 after a real-model manual journey showed that the current test script was only a generic Hermes API caller: Hermes selected web, files, terminal and memory tools and claimed Hermes-owned persistence without Lumen-owned identity, context or policy. The approved slice may proceed without weakening M1 gates because it preserves the existing Host and setup boundaries and adds the missing fail-closed Lumen authority layer. Node transport, web UI, messaging continuity, voice and device capabilities remain out of scope.

The current setup branch must be integrated and verified before product-surface implementation begins. Physical Android, Mac, or iPhone completion is not a prerequisite unless it proves the active milestone's contract.

## Evidence format

Every release check records the date, commit, build and dependency versions, topology and assurance profile, preconditions, exact bounded journey, expected and actual result, pass/fail, and a redacted evidence reference. Evidence uses synthetic identities and content and never contains credentials, private prompts, raw audio, or personal device data.

## Status snapshot

| Area | Current truth | Next proof |
| --- | --- | --- |
| Space authority and encrypted persistence | Implemented and covered by Go tests | Preserve while adding conversations, memory, and node protocol |
| Host-to-Hermes run execution | Development adapter and recovery path implemented; the pinned combined runtime completed the configured synthetic task | Real pinned-runtime exact deny/cancel/lost-stream probes and cross-machine external Runs evidence |
| Setup and public lifecycle | Combined and external Docker journeys pass; real Azure reboot, doctor, state preservation, and Host-only external lifecycle are recorded | Publish distributable artifacts and prove live replacement/rollback |
| Distribution | Official Hermes `v2026.9.7` source archive and container recipe are pinned and locally build-verified; release manifest remains a fixture | Reproducible Lumen artifact plus distributable combined manifest |
| Conversations and memory | A real-model path reaches Hermes, but identity, history, personalization, tool policy and memory are Hermes-owned rather than canonical Lumen state | Implement the approved canonical conversation nucleus with a zero-tool certified profile and separate Hermes plugin |
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

## Milestone 1 — Qualified Host and Hermes foundation

**Status:** active.

**Journey:** choose an explicit `combined` Docker deployment or an `external` independently managed Hermes endpoint, set up the Lumen Host, reboot and diagnose the deployment, and complete, approve, cancel, and recover real Host-mediated tasks without manual environment plumbing. The governing [dual-topology design](./superpowers/specs/2026-09-11-milestone-1-dual-topology-design.md) and [implementation plan](./superpowers/plans/2026-09-11-milestone-1-dual-topology-foundation.md) define the contract.

**Deliver:**

1. Connect one public setup runner to the existing planner, journal, artifact, configuration, Host initialization, supervisor, and validation components, persisting topology independently from assurance profile.
2. In `combined`, own and supervise the Lumen and Hermes artifacts/services in the Docker reference deployment; in `external`, adopt an existing Hermes endpoint and control only the Host.
3. Certify `HermesRuntimeAdapter v1`, one active runtime binding, capability negotiation, bounded events, exact approvals, cancellation, and honest reconciliation without exposing general Space state.
4. Make `lumen setup`, `lumen doctor`, and `lumen service start|stop|restart|status` truthful for both topologies, returning only `ready`, `degraded`, or `action_required`.
5. Replace the fixture release manifest with reproducible integrity-pinned combined inputs and preserve external-adoption identity and credential references without remote mutation.
6. Add continuous verification for formatting, vet, unit, race, contract, reproducible build, secret scanning, deployment configuration, and clean setup.
7. Prove interrupted setup, safe rerun, boot restart, failed update, rollback, approval, cancellation, lost event stream, Host/Hermes restart, external outage/recovery, and Host-only external service control.
8. Record owner evidence for combined and external Linux/VPS topologies, including artifact checksums, real configured-runtime behavior, independent service ownership, reboot, rollback, and canonical state preservation.

macOS and Termux remain supported deployment targets, but physical platform evidence does not block later product work. Hardened Termux continues to require an isolated Hermes endpoint. No local process restart, fake server, fixture manifest, environment marker, or environment variable substitutes for the required evidence.

**Exit:** both topology gates pass; the combined Docker reference and independently managed external endpoint pass real configured-runtime, artifact checksum, reboot, rollback, recovery, and canonical-state checks; no unresolved Critical or Important finding remains; and evidence is recorded in the format above. Until artifact checksum, real-machine, reboot, and rollback evidence exists, M1 remains active. The exit unlocks M2's feature registry, conversation, and memory nucleus and does not claim node transport, messaging, voice, managed hosting, or earned autonomy. See [the foundation gate](./PHASE-2-HOST-HERMES.md).

The combined Azure machine proof and same-VPS external lifecycle/reboot proof are recorded in [the foundation gate](./PHASE-2-HOST-HERMES.md). They do not substitute for a distributable manifest, a public live artifact replacement path, or cross-machine external Runs evidence.

## Milestone 2 — Feature registry, conversation, and memory nucleus

**Status:** bounded conversation-nucleus implementation authorized while the remaining M1 distribution evidence stays open. The governing [conversation nucleus design](./superpowers/specs/2026-09-15-lumen-hermes-conversation-nucleus-design.md) makes an existing co-located Hermes installation the first supported path. Lumen and Hermes remain separately installed, supervised and stored; an unmodified pinned Hermes gains Lumen abilities through a separately versioned native plugin and skill. Combined packaging must use the same contracts and isolation, not a private fork or shared state.

**Journey:** create one Space, talk to Lumen in a responsive web application, retain one useful memory transparently, continue after restart, and inspect or delete what was remembered.

**Deliver:**

1. Add canonical Host-owned `Conversation`, `Message`, `Persona`, `Surface`, `ContextRecord`, `RuntimeProfile`, `RuntimeCertification`, and replaceable runtime-session mapping records with a safe state migration.
2. Add `conversation.chat/respond` and public conversation commands that accept no Hermes instructions, session IDs, model/provider overrides or profile digests; persist intent before network I/O and append one assistant message only after verified completion.
3. Ship a separately pinned native Hermes plugin and bundled onboarding skill. The plugin talks only to a dedicated authenticated Host broker, exposes narrow Lumen tools and hooks, never opens Space storage, and never grants, approves or selects a node.
4. Certify and enforce `lumen.chat.default/v1`: immutable Lumen persona, bounded canonical context, one turn, zero Hermes tools, zero Hermes memory and explicit provider/model/token/deadline policy. Missing or contradictory certification fails before conversation content is sent.
5. Prove the same conversation contract first with an existing co-located Hermes installation and then with isolated combined containers. Host restart and Hermes session loss must preserve canonical continuity; plugin removal must preserve the Space.
6. Add normalized text, usage and failure events, accepted preferences and deletion behavior. Memory proposals, broader policy dimensions, the responsive product experience and the full Hermes component registry follow only after the zero-tool nucleus passes its acceptance, negative, privacy and recovery gates.

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

## Milestone 4 — Hermes plugin and first cross-node action

**Journey:** continue one conversation across web and a lightweight desktop node, ask Lumen to find a permitted document, summarize it, and deliver the result to another surface.

**Deliver:**

1. Ship the first desktop capability node using the public node contract.
2. Connect the separate narrow Lumen Hermes plugin/broker for typed capability intents and redacted inventory; it must not expose direct Space state or target selection.
3. Implement `files.search` and `files.read` for owner-selected roots; keep `files.write` a separate later grant.
4. Reject traversal, symlink escape, path replacement, oversized results, stale resources, and access outside declared roots.
5. Route authorized file content to Hermes under the task's context policy and preserve a redacted durable receipt.
6. Add cross-surface handoff, progress, cancellation, approval, and `unknown_outcome` presentation.

**Exit:** the defining continuity-plus-action journey completes within ten minutes of onboarding and fails honestly under node loss, revocation, stale grants, and Hermes interruption.

## Milestone 5 — Jarvis voice, presence, and companions

**Journey:** say the local wake phrase from a supported voice surface or optional companion, speak naturally, interrupt Lumen while it responds, follow an action across surfaces, and stop listening immediately.

**Deliver:**

1. Standardize `idle`, `wake_detected`, `listening`, `understanding`, `thinking`, `acting`, `approval_needed`, `speaking`, `interrupted`, `offline`, and `degraded` presence states.
2. Keep wake word and voice-activity detection local with zero outbound audio or activation metadata before successful activation.
3. Adopt Hermes voice orchestration and benchmark supported or external STT, TTS, VAD, wake-word, diarization, and interruption components for accuracy, latency, privacy, resources, maintenance, and commercial licensing.
4. Stream post-activation speech under data-aware provider policy, support barge-in, and propagate one cancellation identity through capture, inference, tools, and playback.
5. Synchronize the conversation rather than raw audio by default and make listening state unmistakable on every active surface.
6. Keep the Android desk companion and Mac pet as replaceable clients that expose presence and approval only through Host contracts; neither owns conversation, memory, or runtime state.

**Exit:** network capture proves pre-activation privacy; latency and transcription benchmarks meet the frozen product target; barge-in, locked device, permission revocation, network loss, and cross-surface handoff behave predictably.

## Milestone 6 — Messaging continuity (Hermes gateway cohort)

**Journey:** connect one Hermes-supported messaging account, continue the same Space conversation there, approve a bounded action, and see the result in the web experience.

**Deliver:**

1. Define one conversation-ingress and delivery contract for provider identity, thread mapping, content, attachments, reply targets, credentials, idempotency, receipts, disconnect, and uncertainty.
2. Adopt one Hermes gateway for the controlled beta, then add further gateways through the same contract.
3. Keep provider mechanics in Hermes while Lumen owns identity mapping, canonical conversation, policy, approval, task state, memory, audit, and connection lifecycle.
4. Treat provider messages and gateway metadata as untrusted content; never let a gateway become an authority path.

**Exit:** web, voice/node, and messaging share one canonical conversation; duplicate delivery does not duplicate effects; disconnect revokes future use; uncertain delivery is visible.

## Milestone 7 — Managed and portable beta (Hermes deployment cohort)

**Journey:** start with one-click managed Lumen, export the encrypted Space, migrate it to a paid self-hosted combined deployment, and optionally move Hermes to a separate endpoint without changing product behavior.

**Deliver:**

1. Build a shared commercial control plane that provisions a dedicated Host, encrypted store, Hermes runtime, credentials, backup, and deletion boundary for each customer.
2. Offer combined, separated, managed, and hybrid topologies under one Space contract; keep topology independent of development, personal, and hardened assurance.
3. Add paid entitlement, support-access approval and expiry, minimal content-free telemetry, regional placement, encrypted backup, export/import, and explicit Host migration.
4. Stage Lumen, Hermes, and upstream updates behind compatibility certification, capability kill switches, health gates, and automatic rollback.

**Exit:** all topologies pass the same conformance suite; managed/self-hosted migration preserves identity, conversations, memory, grants, tasks, and audit; a control-plane compromise cannot read canonical content.

## Milestone 8 — Earned autonomy (Hermes automation cohort)

**Journey:** Lumen notices a repeatable pattern, proposes an automation, previews its effects, earns a narrow grant, runs it visibly, and lets the user pause, narrow, or revoke it.

**Deliver:**

1. Reuse Hermes cron, goals, heartbeats, loops, delegation, kanban, batch work, and remote execution rather than rebuilding their engines.
2. Graduate trust through `suggest → preview → approve once → approve workflow → narrow automatic grant`.
3. Seal tools, credentials, context, targets, provider policy, budget, deadline, and cancellation lineage into every parent and child runtime profile.
4. Provide one inspectable activity and intervention surface for scheduled, delegated, and background work.

**Exit:** automation survives restart, never expands its grant, stops through the full child lineage, reports uncertain effects honestly, and can be revoked immediately.

## Milestone 9 — Ecosystem and public launch (Hermes capability cohorts)

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
