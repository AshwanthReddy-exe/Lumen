# Lumen V1 threat model

## Scope and security objective

This threat model covers the planned V1 end state described in [PRD.md](./PRD.md): one active Host, paired nodes, local and cross-node execution, capability adapters, approvals, shared-context synchronization, scheduling, remote transport, backup, restore, and explicit Host migration. Later-phase surfaces remain threats to address before their phase exits; listing them here does not select an implementation.

Lumen must ensure that no node, adapter, runtime, transport, model, or external content receives authority or context beyond an explicit grant. A failure must be reported honestly; uncertainty must never be converted into success.

The Space owner is the only administrative principal in V1. Multi-user Spaces, automatic Host failover, and arbitrary screen control are outside this model.

## Protected assets

| Asset | Security property |
| --- | --- |
| Owner and node private keys | Confidentiality, integrity, non-exportability where hardware permits |
| Space identity and active-Host epoch | Integrity, authenticity, single-authority enforcement |
| Node registry and revocation state | Integrity, freshness, durable recovery |
| Capability manifests and grants | Integrity, least privilege, explainability |
| Commands, task state, and approvals | Authenticity, integrity, freshness, idempotency |
| Shared and local context | Confidentiality, integrity, origin, retention, deletion |
| Artifacts and patch digests | Integrity, provenance, stale-base detection |
| Schedules | Integrity, bounded authority, predictable execution |
| Audit and recovery records | Integrity, availability, redaction |

Availability matters, but it does not override authorization or confidentiality. Lumen may become unavailable rather than fail open.

## Trust boundaries

1. **Owner to surface:** text, voice, approval, pairing, recovery, and policy UI may be spoofed or misunderstood.
2. **Node to local runtime:** models, tools, Hermes, OS services, and capability adapters are untrusted inputs to policy enforcement.
3. **Node to Host:** every message crosses an authenticated, encrypted, replay-resistant boundary even on a trusted LAN.
4. **Host storage:** process memory, durable records, backups, migrations, and restored state have different exposure and freshness risks.
5. **Space to relay or push provider:** remote infrastructure transports opaque envelopes and owns no Space authority.
6. **Coding workspace to canonical repository:** generated changes remain untrusted until scope, digest, base, and approval checks pass.
7. **Context namespace to capability:** access to one capability or record type must not imply access to another.

Physical compromise of an unlocked node, a compromised operating system, malicious firmware, and denial of service by the network provider cannot be fully prevented by Lumen V1. The product must limit resulting authority, support revocation and recovery, and state these residual risks during setup.

## Threat actors and inputs

- A remote attacker who can observe, replay, delay, reorder, inject, or drop traffic.
- A revoked, stolen, or compromised node with previously valid credentials or cached grants.
- A malicious or compromised model, MCP server, tool, Hermes runtime, adapter, relay, or notification provider.
- Malicious content in repositories, prompts, files, reminders, notifications, or synchronized context.
- An accidental owner action caused by ambiguous targeting, misleading approval details, or stale UI.
- A local attacker who obtains a device, backup, diagnostic bundle, or unencrypted storage.
- Faults such as crashes, clock skew, disk exhaustion, partial writes, duplicate delivery, and interrupted migration.

All external content is data, never policy or authority.

## Threat matrix

| ID | Threat and impact | Required mitigation | Verification | Release status |
| --- | --- | --- | --- | --- |
| T-001 | Forged node or Host message causes unauthorized work or disclosure. | Device-generated keys; signatures bind Space, sender, recipient, task, schema, nonce, issue time, and expiry; authenticated pairing. | Reject altered identities, fields, signatures, and unknown keys on every supported platform. | Blocking |
| T-002 | Replay or duplicate delivery repeats an action. | Expiring messages, durable nonce tracking, idempotency keys, once-only approval consumption, idempotent state transitions. | Replay before and after restart; duplicate and reordered command/event fixtures. | Blocking |
| T-003 | A revoked node continues executing or synchronizing. | Host-authoritative revocation immediately blocks Host-mediated work and terminates connected sessions. A disconnected node may execute locally only until its previously issued cached grant expires; reconnect requires current authorization before synchronization or new cross-node work. | Revoke online and offline nodes; attempt Host-mediated work, local cached-grant execution, and context upload before expiry and after reconnect. | Blocking |
| T-004 | Split brain permits two Hosts to accept work. | Explicit migration, monotonically increasing Host epoch or lease, signed transfer record, stale-Host rejection, no automatic failover. | Interrupt migration at each durable step; reconnect old Host and verify rejection. | Blocking |
| T-005 | Adapter or runtime expands its grant or accesses unrelated context. | Default deny; Host/node policy checks outside adapters; narrow credential handles; typed context inputs; sandbox or OS boundary where available. | Malicious adapter fixtures request extra action, resource, secret, and context namespaces. | Blocking |
| T-006 | Approval is reused or substituted for a different action. | Bind approval to task, action, canonical arguments or artifact digest, actor, target node, expiry, and single consumption. | Mutate each bound field, reuse approval, and race two consumers. | Blocking |
| T-007 | Prompt or content injection changes policy, target, or scope. | Treat model and external content as untrusted; keep authority in deterministic code; structurally separate data from control; show exact action at approval. | Inject policy-like text through every context and adapter input; confirm no grant or routing change. | Blocking |
| T-008 | Context exceeds the selected synchronization level. | Per-capability `none`, `metadata`, `summary`, or `content` filter before serialization; typed namespaces; classification and retention enforcement. | Inspect serialized envelopes and Host records for every level and data class. | Blocking |
| T-009 | Logs, notifications, diagnostics, or audit leak secrets or private content. | Structured redaction, privacy-safe previews, scoped diagnostics, secret handles, no raw prompts or credentials in ordinary logs. | Seed canary secrets and private values; scan all emitted artifacts and crash paths. | Blocking |
| T-010 | A coding result modifies the wrong repository content. | Isolated Git worktree; path and symlink validation; allowlisted scope; patch digest; stale-base check; approval before canonical apply. | Test traversal, symlink escape, tampering, stale base, submodules, and partial apply. | Blocking for `coding.run` |
| T-011 | Crash, disk pressure, or lost events reports success incorrectly. | Durable append-only events; compare-and-set transitions; atomic persistence; reconciliation; explicit `unknown_outcome`; queue limits. | Kill processes and exhaust/inject storage failures at each transition; lose and resume event streams. | Blocking |
| T-012 | Backup, export, or restore exposes data or restores stale authority. | Owner-authenticated encrypted export, versioned migrations, recovery-key handling, and revocation/Host-authority reconciliation. `O-005` must define authenticated metadata, keys, freshness, and rollback handling before implementation. | Corrupt, truncate, roll back, and restore backups with valid, invalid, and lost keys using the accepted `O-005` design. | Blocking |
| T-013 | Relay or push provider reads content or gains authority. | End-to-end encrypted envelopes; relay holds no keys or grants; minimal wake-up metadata; expiry and abuse limits. | Compromised-relay tests for read, forge, replay, reorder, and traffic retention. | Blocking before remote access |
| T-014 | Ambiguous routing sends work or data to the wrong node. | Deterministic eligibility from capability, policy, health, and constraints; require owner selection when multiple valid targets remain. | Equal-candidate, stale-health, unsupported-version, and offline-target fixtures. | Blocking |
| T-015 | Clock manipulation bypasses expiry or schedules unintended work. | Revalidate expiry at dispatch and execution. `O-003` and `O-004` must select the trusted-time, skew, suspend, and offline-expiry behavior before implementation. | Test forward/backward jumps, timezone changes, DST, suspend, and long offline periods against the accepted design. | Blocking before scheduling |
| T-016 | Pairing is intercepted or confirms the wrong device. | Short-lived authenticated transcript, out-of-band comparison or explicit owner confirmation, key proof, rate limits, cancellation. | MITM, transcript substitution, timeout, brute-force, and concurrent-pairing tests. | Blocking |

## Security invariants

- Policy evaluation defaults to deny and runs before dispatch and again before execution.
- Local execution may bypass the Host data path, never the local capability policy.
- Cross-node execution requires current Host authorization.
- An adapter cannot issue grants, consume unrelated context, or turn runtime approval into Lumen authority.
- One-time approval authorizes one exact action and is not a durable grant.
- Revocation blocks future synchronization and work once the Host records it.
- Unknown versions and unknown security-critical fields fail closed.
- Task and context writes preserve origin and ordering evidence; arrival order alone does not resolve conflicts.
- Secrets and raw private content do not enter prompts, ordinary logs, notifications, or test evidence.

## Release-blocking evidence

The following evidence is required before the relevant phase can exit:

1. Protocol fixtures prove authentication, expiry, replay defense, version handling, and fail-closed parsing across every selected platform.
2. Policy fixtures prove default deny, adapter confinement, exact approval binding, and deterministic routing.
3. Recovery fixtures prove honest task outcomes after restart, partial persistence, duplicate delivery, and event loss.
4. Context fixtures prove the selected synchronization level at both serialized-message and Host-store boundaries.
5. Host lifecycle and migration tests prove revocation, encrypted recovery, and stale-Host rejection.
6. Capability-specific abuse cases pass before that capability ships.
7. A named owner reviews unresolved blocking threats at every phase exit; open blocking items prevent release.

Evidence follows the format in [PLAN.md](./PLAN.md#evidence-format) and must use synthetic identifiers and content. Test fixtures must contain no production credentials, personal identifiers, or private context.

## Review triggers

Review this model when a trust boundary changes; a new capability, transport, runtime, data class, or Host platform is introduced; an open decision in [DECISIONS.md](./DECISIONS.md) is accepted; or a security or privacy incident reveals a missing assumption. Record durable architectural consequences in `DECISIONS.md` rather than silently changing this model.
