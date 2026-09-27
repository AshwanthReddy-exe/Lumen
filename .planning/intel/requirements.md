# Requirements

## REQ-fr-01
- source: docs/PRD.md
- description: Create one private Space with one active Host and a recoverable owner identity.
- acceptance: absent
- scope: absent

## REQ-fr-02
- source: docs/PRD.md
- description: Pair and revoke nodes using device-generated keys and explicit confirmation.
- acceptance: absent
- scope: absent

## REQ-fr-03
- source: docs/PRD.md
- description: Let any eligible deployment environment run the Host contract and migrate explicitly; co-located Host and node identities remain separate.
- acceptance: absent
- scope: absent

## REQ-fr-04
- source: docs/PRD.md
- description: Keep canonical shared context, policies, task history, and node registry on the active Host.
- acceptance: absent
- scope: absent

## REQ-fr-05
- source: docs/PRD.md
- description: Authenticate signed, expiring messages and reject repeated nonces or revoked senders.
- acceptance: absent
- scope: absent

## REQ-fr-06
- source: docs/PRD.md
- description: Run the Host as a headless foreground process supervised externally under one portable executable contract.
- acceptance: absent
- scope: absent

## REQ-fr-07
- source: docs/PRD.md
- description: Keep companion applications outside Host authority even when they share a physical device.
- acceptance: absent
- scope: absent

## REQ-fr-08
- source: docs/PRD.md
- description: Provide resumable `lumen setup` that installs or adopts compatible Lumen and Hermes artifacts, generates configuration, and initializes Host state exactly once.
- acceptance: absent
- scope: absent

## REQ-fr-09
- source: docs/PRD.md
- description: Supervise Host and Hermes at boot where supported and report exactly `ready`, `degraded`, or `action_required` without silent downgrade.
- acceptance: absent
- scope: absent

## REQ-fr-10
- source: docs/PRD.md
- description: Let a node advertise versioned capabilities, health, constraints, and local execution support.
- acceptance: absent
- scope: absent

## REQ-fr-11
- source: docs/PRD.md
- description: Configure each capability as `deny`, `ask`, or `allow`, with action, resource, time, and data scopes.
- acceptance: absent
- scope: absent

## REQ-fr-12
- source: docs/PRD.md
- description: Validate capability, scope, user authority, expiry, node health, and approval before execution.
- acceptance: absent
- scope: absent

## REQ-fr-13
- source: docs/PRD.md
- description: Report unsupported or offline work as queued, failed, or unavailable, never completed.
- acceptance: absent
- scope: absent

## REQ-fr-14
- source: docs/PRD.md
- description: Support many adapters without giving one access to unrelated capabilities or context.
- acceptance: absent
- scope: absent

## REQ-fr-15
- source: docs/PRD.md
- description: Pair nodes through QR or short code with explicit confirmation and device-management controls.
- acceptance: absent
- scope: absent

## REQ-fr-20
- source: docs/PRD.md
- description: Execute same-node work locally when policy permits without routing model or tool traffic through the Host.
- acceptance: absent
- scope: absent

## REQ-fr-21
- source: docs/PRD.md
- description: Synchronize only the user-permitted task outcome and context delta with the Host.
- acceptance: absent
- scope: absent

## REQ-fr-22
- source: docs/PRD.md
- description: Route cross-node work through the Host using idempotent commands and durable task state.
- acceptance: absent
- scope: absent

## REQ-fr-23
- source: docs/PRD.md
- description: Allow local work during Host unavailability only within unexpired cached grants and show unsynchronized state.
- acceptance: absent
- scope: absent

## REQ-fr-24
- source: docs/PRD.md
- description: Reconcile duplicates, conflicts, and delayed events without silently overwriting newer context.
- acceptance: absent
- scope: absent

## REQ-fr-30
- source: docs/PRD.md
- description: Persist an honest task lifecycle across restart, disconnect, cancellation, and unknown outcomes.
- acceptance: absent
- scope: absent

## REQ-fr-31
- source: docs/PRD.md
- description: Bind one-time approval to the exact task, action, arguments or artifact digest, actor, node, and expiry.
- acceptance: absent
- scope: absent

## REQ-fr-32
- source: docs/PRD.md
- description: Support immediate and scheduled tasks with explicit target, authority, delivery, and expiry.
- acceptance: absent
- scope: absent

## REQ-fr-33
- source: docs/PRD.md
- description: Let users select `none`, `metadata`, `summary`, or `content` synchronization per capability where supported.
- acceptance: absent
- scope: absent

## REQ-fr-34
- source: docs/PRD.md
- description: Encrypt sensitive Host state and cross-node content with export, backup, restore, and deletion controls.
- acceptance: absent
- scope: absent

## REQ-fr-35
- source: docs/PRD.md
- description: Record redacted audit events for routing, policy, approval, execution, synchronization, and outcome.
- acceptance: absent
- scope: absent

## REQ-fr-36
- source: docs/PRD.md
- description: Let a runtime propose typed memory while only the Host validates and persists canonical context.
- acceptance: absent
- scope: absent

## REQ-fr-37
- source: docs/PRD.md
- description: Keep browser profiles and credentials outside shared context and gate browser actions by capability policy.
- acceptance: absent
- scope: absent

## REQ-fr-38
- source: docs/PRD.md
- description: Integrate Hermes through a versioned adapter with discovery, runs, events, approvals, cancellation, and untrusted-result handling.
- acceptance: absent
- scope: absent

## REQ-fr-39
- source: docs/PRD.md
- description: Reuse approved Hermes model routing, tools, skills, MCP, browser, voice, delegation, and remote execution through immutable scoped profiles.
- acceptance: absent
- scope: absent

## REQ-fr-40
- source: docs/PRD.md
- description: Perform no background model or tool work by default; keep pre-activation wake-word and VAD local with zero outbound audio, while schedules and triggers require explicit inspectable grants.
- acceptance: absent
- scope: absent

## REQ-fr-41
- source: docs/PRD.md
- description: Expose Host-local Hermes reasoning first as `agent.run/execute` without granting transitive authority.
- acceptance: absent
- scope: absent

## REQ-fr-42
- source: docs/PRD.md
- description: Expose Hermes-backed integrations only through Lumen lifecycle, policy, credential, audit, cancellation, receipt, and uncertainty contracts.
- acceptance: absent
- scope: absent

## REQ-fr-60
- source: docs/PRD.md
- description: Represent canonical conversations independently from Hermes sessions and map web, messaging, voice, native, and ambient surfaces into them.
- acceptance: absent
- scope: absent

## REQ-fr-61
- source: docs/PRD.md
- description: Give accepted memory provenance, classification, retention, scope, inspection, correction, deletion, and export; sensitive or consequential memory requires confirmation.
- acceptance: absent
- scope: absent

## REQ-fr-62
- source: docs/PRD.md
- description: Preserve identity, conversations, memory, and task continuity across Host restart, Hermes failure or replacement, surface disconnect, restore, and migration.
- acceptance: absent
- scope: absent

## REQ-fr-63
- source: docs/PRD.md
- description: Project only task-relevant, policy-permitted context to a runtime, model, integration, or node.
- acceptance: absent
- scope: absent

## REQ-fr-64
- source: docs/PRD.md
- description: Make the Host select, dispatch, observe, cancel, and reconcile node work while the target node revalidates the exact invocation and current local permission.
- acceptance: absent
- scope: absent

## REQ-fr-65
- source: docs/PRD.md
- description: Begin file access with separate `files.search` and `files.read` grants over user-selected roots and reject traversal, symlink escape, stale paths, oversized results, and implicit writes.
- acceptance: absent
- scope: absent

## REQ-fr-66
- source: docs/PRD.md
- description: Provide a Devices and abilities view for pairing, health, grants, activity, synchronization, local stop, and revocation.
- acceptance: absent
- scope: absent

## REQ-fr-67
- source: docs/PRD.md
- description: Graduate autonomy only through suggestion, preview, one-time approval, workflow approval, and a narrow inspectable revocable grant.
- acceptance: absent
- scope: absent

## REQ-fr-68
- source: docs/PRD.md
- description: Bind delegated children to the parent's tools, credentials, context, targets, provider policy, budget, deadline, and cancellation lineage.
- acceptance: absent
- scope: absent

## REQ-fr-69
- source: docs/PRD.md
- description: Prefer Hermes for intelligence subsystems; use another mature component or custom Lumen code only after supported extension fails and reproducible benchmarks justify it.
- acceptance: absent
- scope: absent

## REQ-fr-70
- source: docs/PRD.md
- description: Map messaging identities and threads into canonical Space conversations and deliver through typed capabilities with deduplication and honest receipts.
- acceptance: absent
- scope: absent

## REQ-fr-71
- source: docs/PRD.md
- description: Expose consistent presence states for idle, wake detection, listening, understanding, thinking, acting, approval, speaking, interruption, offline, and degraded operation.
- acceptance: absent
- scope: absent

## REQ-fr-72
- source: docs/PRD.md
- description: Provide post-activation streaming recognition, interruptible speech, barge-in, immediate stop, and unmistakable listening state.
- acceptance: absent
- scope: absent

## REQ-fr-73
- source: docs/PRD.md
- description: Route models and speech engines only within approved provider, region, retention, locality, latency, cost, and data-class constraints.
- acceptance: absent
- scope: absent

## REQ-fr-74
- source: docs/PRD.md
- description: Support combined, separated, managed, and hybrid topologies under one Space contract, independently of development, personal, and hardened assurance.
- acceptance: absent
- scope: absent

## REQ-fr-75
- source: docs/PRD.md
- description: Keep canonical encrypted storage with the Space Authority and limit Hermes and workers to bounded operational state.
- acceptance: absent
- scope: absent

## REQ-fr-76
- source: docs/PRD.md
- description: Provision a dedicated isolated data plane for each managed customer and minimize shared control-plane metadata.
- acceptance: absent
- scope: absent

## REQ-fr-77
- source: docs/PRD.md
- description: Offer managed and self-hosted Lumen as paid proprietary products with interoperable encrypted exports and equivalent semantics.
- acceptance: absent
- scope: absent

## REQ-fr-78
- source: docs/PRD.md
- description: Publish versioned node, capability, runtime, and integration contracts, SDKs, conformance fixtures, and compatibility rules without publishing proprietary product code.
- acceptance: absent
- scope: absent

## REQ-fr-79
- source: docs/PRD.md
- description: Qualify every upstream component for provenance, commercial rights, security, maintenance, data flow, benchmarks, fallback, rollback, and disablement.
- acceptance: absent
- scope: absent

## REQ-fr-80
- source: docs/PRD.md
- description: Use progressive disclosure so ordinary users avoid infrastructure complexity while power users and self-hosters can inspect advanced controls.
- acceptance: absent
- scope: absent

## REQ-fr-81
- source: docs/PRD.md
- description: Persist Hermes clarification questions separately from action approvals; answering a question supplies input but grants no new authority.
- acceptance: absent
- scope: absent

## REQ-fr-82
- source: docs/PRD.md
- description: Show approvals and clarifications in a synchronized attention inbox and their conversation, with one accepted resolution, expiry, and consistent results across surfaces.
- acceptance: absent
- scope: absent

## REQ-fr-83
- source: docs/PRD.md
- description: Keep wake detection, recognition, and speech synthesis local by default; any remote speech processing requires explicit opt-in and must never be an automatic fallback.
- acceptance: absent
- scope: absent

## REQ-fr-84
- source: docs/PRD.md
- description: Qualify shared mobile UI and native audio/security modules on physical Android and iPhone devices before committing to that client strategy; provide the documented native-Android plus phone-web fallback if it fails.
- acceptance: absent
- scope: absent
