# Lumen V1 threat model

## Scope and security objective

This threat model covers the product direction described in [PRD.md](./PRD.md): one active Host, paired nodes, local and cross-node execution, restricted device-file access, capability adapters, approvals, conversations and shared context, Jarvis-style voice and presence, scheduling, remote transport, dedicated managed deployments, paid self-hosting, backup, restore, and explicit Host migration. Later-milestone surfaces remain threats to address before their own exits; listing them here does not select an implementation or imply that it ships.

Lumen must ensure that no node, adapter, runtime, transport, model, or external content receives authority or context beyond an explicit grant. A failure must be reported honestly; uncertainty must never be converted into success.

The Space owner is the only Space administrative principal. Managed operators and the shared control plane are infrastructure actors, never Space principals, and receive no implicit content or action authority. Multi-user Spaces, automatic Host failover, and arbitrary screen control are outside this model.

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
| Host-state key, operator credential, and Hermes bearer credential | Independent confidentiality, rotation, least privilege, non-interchangeability |
| Conversation, voice, and presence state | Confidentiality, intentional activation, correct speaker/session binding, minimal disclosure |
| Managed tenant identity and isolation boundary | Confidentiality, integrity, deletion, non-confusion between customer data planes |
| Public protocol and SDK compatibility state | Authenticity, downgrade resistance, fail-closed version negotiation |
| Commercial entitlement and billing state | Integrity, privacy, separation from Space authority and content |
| Upstream component registry and artifacts | Provenance, integrity, license compliance, rollback, revocation |

Availability matters, but it does not override authorization or confidentiality. Lumen may become unavailable rather than fail open.

## Trust boundaries

1. **Owner to surface:** text, voice, approval, pairing, recovery, and policy UI may be spoofed or misunderstood.
2. **Node to local runtime:** models, tools, Hermes, OS services, and capability adapters are untrusted inputs to policy enforcement.
3. **Node to Host:** every message crosses an authenticated, encrypted, replay-resistant boundary even on a trusted LAN.
4. **Host storage:** process memory, durable records, backups, migrations, and restored state have different exposure and freshness risks.
5. **Space to relay or push provider:** remote infrastructure transports opaque envelopes and owns no Space authority.
6. **Coding workspace to canonical repository:** generated changes remain untrusted until scope, digest, base, and approval checks pass.
7. **Context namespace to capability:** access to one capability or record type must not imply access to another.
8. **Operator CLI to Host:** a local process crossing the owner-restricted Unix-domain control socket is untrusted until its operator credential, endpoint identity, request bounds, actor mapping, freshness, and action scope are validated.
9. **Host to Hermes:** the configurable runtime endpoint, mutual endpoint authentication, bearer authentication, HTTP behavior, SSE stream, runtime approvals, tool requests, and results are untrusted adapter inputs even when both processes are local.
10. **Host to capability node:** discovery, routing, invocation, result, and file metadata cross a mutually authenticated boundary; Host authorization and node-local policy are independent required checks.
11. **Voice and ambient surface to Space:** wake detection, speaker identity hints, transcripts, presence state, and synthesized speech may be observed, replayed, or impersonated.
12. **Managed control plane to dedicated data plane:** provisioning, upgrades, health, billing, support, backup, deletion, and migration must not create ambient access to canonical customer content or credentials.
13. **Tenant to tenant:** compute, storage, network identity, caches, logs, backups, queues, metrics, and operator workflows must not mix customer data or authority.
14. **Public client or integration to protocol endpoint:** version negotiation, manifests, schemas, and SDK behavior are attacker-controlled until authenticated and validated.
15. **Build and dependency source to release:** Hermes and other upstream source, binaries, containers, plugins, models, skills, and licenses may change or be compromised between qualification and deployment.

Physical compromise of an unlocked node, a compromised operating system, malicious firmware, and denial of service by the network provider cannot be fully prevented by Lumen V1. The product must limit resulting authority, support revocation and recovery, and state these residual risks during setup.

Processes in one Termux installation share an Android UID. Co-locating Host and Hermes there is therefore a compatibility profile for synthetic, non-sensitive state, not an isolation boundary. A security-valid Termux deployment uses an isolated Hermes endpoint on another machine or container with pinned mutual TLS; compromise of the Termux UID remains compromise of the Host.

## Threat actors and inputs

- A remote attacker who can observe, replay, delay, reorder, inject, or drop traffic.
- A revoked, stolen, or compromised node with previously valid credentials or cached grants.
- A malicious or compromised model, MCP server, tool, Hermes runtime, adapter, relay, or notification provider.
- Malicious content in repositories, prompts, files, reminders, notifications, or synchronized context.
- An accidental owner action caused by ambiguous targeting, misleading approval details, or stale UI.
- A local attacker who obtains a device, backup, diagnostic bundle, or unencrypted storage.
- A malicious local process that can scan loopback ports, read another process’s environment or command line, alter configuration, follow redirects, or race the operator and Hermes endpoints.
- A compromised managed control plane, operator account, support workflow, deployment worker, tenant workload, or backup system attempting cross-tenant access.
- A nearby person, recording, synthesized voice, compromised microphone/speaker path, or malicious surface attempting to impersonate the owner or infer presence.
- A malicious or obsolete public-protocol client attempting downgrade, schema confusion, capability spoofing, or incompatible replay.
- A compromised upstream maintainer, registry, package, release artifact, plugin, model, or build pipeline, and a dependency whose license no longer permits distribution or hosted use.
- A user or attacker attempting entitlement bypass, billing-identity confusion, or using commercial state to infer private Space activity.
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
| T-012 | Backup, export, or restore exposes data or restores stale authority. | Owner-authenticated encrypted export, versioned migrations, recovery-key handling, and revocation/Host-authority reconciliation defined by `D-024`. | Corrupt, truncate, roll back, and restore backups with valid, invalid, and lost keys using the accepted recovery design. | Blocking |
| T-013 | Relay or push provider reads content or gains authority. | End-to-end encrypted envelopes; relay holds no keys or grants; minimal wake-up metadata; expiry and abuse limits. | Compromised-relay tests for read, forge, replay, reorder, and traffic retention. | Blocking before remote access |
| T-014 | Ambiguous routing sends work or data to the wrong node. | Deterministic eligibility from capability, policy, health, and constraints; require owner selection when multiple valid targets remain. | Equal-candidate, stale-health, unsupported-version, and offline-target fixtures. | Blocking |
| T-015 | Clock manipulation bypasses expiry or schedules unintended work. | Revalidate expiry at dispatch and execution; `D-022` requires a durable time high-water mark and fail-closed degraded state after a backwards jump. | Test forward/backward jumps, timezone changes, DST, suspend, and long offline periods against the accepted design. | Blocking before scheduling |
| T-016 | Pairing is intercepted or confirms the wrong device. | Short-lived authenticated transcript, out-of-band comparison or explicit owner confirmation, key proof, rate limits, cancellation. | MITM, transcript substitution, timeout, brute-force, and concurrent-pairing tests. | Blocking |
| T-017 | A local process impersonates the operator or replays an administrative request. | Owner-restricted Unix-domain socket; pinned Host endpoint identity; independent generated operator credential; bounded authenticated requests; explicit owner-principal mapping; idempotency, freshness, rotation, and redacted diagnostics. | Probe unauthorized socket access, unauthenticated, stolen, stale, replayed, oversized, and rotated credentials; scan process arguments and logs for canary secrets. | Blocking for foundation release |
| T-018 | Hermes or a redirected runtime endpoint causes SSRF, bypasses approval, loses events, or forges completion. | Hardened deployments isolate Hermes under another OS principal or container and use pinned mutual TLS through a protected endpoint or proxy. The adapter rejects redirects and userinfo, uses a separate scoped bearer credential, pins a behavior-tested Hermes compatibility record, treats SSE as lossy single-consumer evidence, and derives authority from Host state plus bounded status reconciliation. Plain loopback is a synthetic development profile only. | Test endpoint confusion, redirects, authentication failure, certificate and identity mismatch, advertised-but-broken features, approval bypass, concurrent or subsequent SSE consumers, disconnect, forged terminal events, and credential rotation. | Blocking for the affected deployment profile |
| T-019 | A delegated run, skill, tool, MCP server, or remote executor escapes its parent task or changes policy. | Treat metadata and outputs as untrusted; launch an immutable verified worker profile containing only the parent task's approved tools, credentials, models, remote backend, context, budget, deadline, and cancellation lineage. Because Hermes children inherit the parent runtime surface, disable delegation unless that entire transitive surface is granted. Children cannot approve themselves or create grants. | Attempt profile mutation, scope expansion, ambient credential access, untracked child work, target substitution, recursive delegation, and completion after parent cancellation. | Blocking before each Hermes surface ships |
| T-020 | Model routing sends sensitive context to an unapproved provider, region, or local runtime. | Host or node policy constrains provider, model, location, data classification, cost, and retention before context projection and seals them into the run profile; unknown or changed routing fails closed. | Substitute providers/models/regions, mutate the run profile, mislabel local execution, exceed budget, and inspect outbound payloads at every classification. | Blocking before dynamic model routing |
| T-021 | Voice activation records or executes without an intentional invocation. | Pre-activation wake-word/VAD runs only on the node without Hermes client-capture or model calls; visible listening state; short-lived audio; immediate stop; no raw-audio retention by default; capability and OS permission checks before capture or transmission. | Use a network capture to prove zero outbound audio, activation metadata, model calls, and tool calls before activation; also test false wake, locked device, revoked mic grant, background capture, barge-in, stop latency, network loss, and retention deletion. | Blocking before voice ships |
| T-022 | A managed control-plane or operator compromise exposes a customer's canonical state, credentials, or action authority. | Dedicated per-customer data plane; no content keys or Space credentials in the shared control plane; least-privilege short-lived support access with owner-visible audit and expiry; content-free health telemetry; separate backup and deletion roles. | Compromise control-plane and support fixtures; attempt content reads, credential minting, task dispatch, backup access, and access after expiry. | Blocking before managed beta |
| T-023 | Tenant confusion or isolation failure mixes customer storage, compute, caches, logs, backups, network identity, or migration targets. | Stable tenant binding on every provision, route, credential, store, backup, restore, deletion, and migration operation; isolated data-plane resources; deny ambiguous or mismatched tenancy; no shared runtime cache containing customer content. | Cross-tenant identifier substitution, cache/log canaries, restore-to-wrong-tenant, concurrent deletion/migration, and compromised-tenant escape tests. | Blocking before managed beta |
| T-024 | A recording, synthesized voice, or compromised surface impersonates the owner and authorizes an action. | Wake word and speaker recognition are presence hints, never sole authority for consequential action; bind the active surface and session; require action-appropriate confirmation or a pre-existing narrow grant; make spoken approvals exact, expiring, interruptible, and visibly attributable. | Replay and synthetic-voice corpus, nearby-speaker attack, session swap, locked-device, stale transcript, and cross-surface approval-substitution tests. | Blocking before voice actions ship |
| T-025 | Listening, speaking, node availability, or handoff metadata reveals the owner's presence, location, routine, or sensitive conversation. | Minimize and classify presence metadata; keep pre-activation processing local; disclose state only to authorized surfaces; use privacy-safe notifications and logs; bound retention; make remote presence sharing explicit and revocable. | Observe network, relay, logs, notifications, analytics, and revoked surfaces across idle, wake, listening, acting, speaking, and handoff states. | Blocking before ambient presence ships |
| T-026 | Protocol downgrade or schema confusion weakens authentication, grant scope, replay defense, or capability semantics. | Authenticate negotiation; bind selected protocol/schema/capability versions into pairing, grants, invocations, and receipts; maintain supported-version floors; reject unknown security fields and incompatible semantics; revoke vulnerable versions through the component registry. | Strip or alter version offers, replay old manifests and grants, substitute schemas, and connect below the configured security floor. | Blocking before public protocol beta |
| T-027 | Commercial entitlement bypass grants service, while billing or entitlement failure disables local authority or leaks private activity. | Keep entitlement and billing outside Space authority; use signed, minimal, time-bounded entitlement records with an explicit offline grace policy; never send conversation or capability contents for billing; failure may limit commercial service but cannot corrupt, erase, or silently unlock the Space. | Forge, replay, expire, revoke, and confuse account/tenant entitlement; disconnect billing; inspect billing payloads; verify export and local canonical-state integrity. | Blocking before paid release |
| T-028 | A compromised, incompatible, abandoned, or improperly licensed upstream component introduces code execution, disclosure, behavioral drift, or unlawful distribution. | Component registry records exact provenance, hashes/signatures, license and hosted-use rights, data access, behavioral compatibility, vulnerabilities, fallback, staged rollout, kill switch, and rollback. Pin qualified artifacts; minimize plugins and transitive access; requalify material changes. | Tampered artifact, dependency confusion, revoked signature, license-policy failure, incompatible Hermes behavior, malicious plugin, vulnerable-version rollback, and kill-switch exercises. | Blocking per component cohort |
| T-029 | Restricted file access escapes an owner-selected root, follows a malicious link, leaks excessive content, or turns read authority into write/execution authority. | Separate typed search, metadata, read, and write actions; bind grants to canonical owner-selected roots and data limits; resolve and revalidate paths and symlinks on the node; reject traversal, special files, mounts, stale handles, and oversized results; writes require separate authority and atomic/evidence-aware behavior. | Traversal, symlink race, mount escape, case/Unicode confusion, stale path, hard link, special file, oversized/binary content, write-through-read, revocation, and disconnect tests on each supported node OS. | Blocking before file capability ships |

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
- Host-state encryption, operator control, and Hermes authentication use independently generated credentials; compromise or rotation of one never authorizes use of another.
- The shared managed control plane never owns Space authority, customer content keys, or reusable node credentials.
- Wake word, speaker recognition, voice resemblance, and physical proximity never independently authorize a consequential action.
- Search or read permission never implies write, execute, broader-root, or cross-node disclosure authority.
- Commercial entitlement may enable a product service; it cannot mint Space grants or become the only path to owner export and recovery.
- Protocol compatibility is authenticated and explicit; security-critical behavior never silently downgrades.

## Release-blocking evidence

The following evidence is required before the relevant phase can exit:

1. Protocol fixtures prove authentication, expiry, replay defense, version handling, and fail-closed parsing across every selected platform.
2. Policy fixtures prove default deny, adapter confinement, exact approval binding, and deterministic routing.
3. Recovery fixtures prove honest task outcomes after restart, partial persistence, duplicate delivery, and event loss.
4. Context fixtures prove the selected synchronization level at both serialized-message and Host-store boundaries.
5. Host lifecycle and migration tests prove revocation, encrypted recovery, and stale-Host rejection.
6. Capability-specific abuse cases pass before that capability ships.
7. A named owner reviews unresolved blocking threats at every phase exit; open blocking items prevent release.
8. Host/Hermes fixtures prove operator authentication, endpoint confinement, behavioral compatibility, lossy-event reconciliation, exact approval forwarding, credential separation, and secret redaction.
9. Managed-deployment fixtures prove tenant isolation, control-plane non-authority, scoped support access, content-free telemetry, backup/restore binding, deletion, and migration.
10. Jarvis-experience fixtures prove local pre-activation privacy, impersonation resistance appropriate to action risk, presence minimization, interruption, handoff binding, and permission revocation.
11. Public protocol and component-registry fixtures prove authenticated negotiation, downgrade rejection, artifact provenance, behavioral compatibility, license approval, staged rollback, and kill switches.
12. File-capability fixtures prove root confinement, node-local revalidation, bounded disclosure, distinct write authority, revocation, and honest uncertain outcomes.

Evidence follows the format in [PLAN.md](./PLAN.md#evidence-format) and must use synthetic identifiers and content. Test fixtures must contain no production credentials, personal identifiers, or private context.

## Review triggers

Review this model when a trust boundary changes; a new capability, transport, runtime, data class, or Host platform is introduced; an open decision in [DECISIONS.md](./DECISIONS.md) is accepted; or a security or privacy incident reveals a missing assumption. Record durable architectural consequences in `DECISIONS.md` rather than silently changing this model.
