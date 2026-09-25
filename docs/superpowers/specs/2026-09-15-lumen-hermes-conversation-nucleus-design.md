# Lumen–Hermes conversation nucleus design

## Status

Proposed for owner review on 2026-09-15. This specification refines Milestone 2 in `docs/PLAN.md`. It does not close the remaining Milestone 1 distribution and real-runtime evidence gates.

## Purpose

Turn the existing Host-to-Hermes Runs integration into a real Lumen conversation: the Lumen Host owns identity, persona, canonical messages, personalization, context policy, runtime profile, approvals and durable outcomes, while an unmodified Hermes supplies reasoning and its mature feature implementations behind Lumen-owned contracts.

The primary deployment is one owner-controlled machine or VPS containing separately isolated Lumen Host and Hermes processes. Existing independently managed Hermes is supported first. A combined installer may provision the same components later without changing their contracts. A cross-machine deployment uses the same protocol but requires an explicitly reachable authenticated endpoint.

## Non-goals

- Forking or patching Hermes core.
- Sharing Lumen state, encryption keys, owner socket or raw database files with Hermes.
- Treating a static skill, prompt, Hermes session, `MEMORY.md`, `USER.md` or external memory provider as canonical Lumen state.
- Enabling web, files, shell, browser, device control, MCP, delegation, cron or messaging merely because Hermes discovers them.
- Shipping the responsive web UI, voice, messaging continuity or node fabric in this slice.
- Completing a one-command combined installer in this slice. Combined packaging remains a supported topology and later delivery task.

## Requirements

### Functional

1. An authenticated owner can create a conversation, send a message, inspect canonical history and continue after Host or Hermes restart.
2. Every Hermes request carries a Host-owned immutable Lumen persona, a Host-generated runtime profile digest, a Host-generated session mapping and bounded canonical history.
3. A user preference affects a response only after it exists as an accepted, typed Lumen context record selected by policy.
4. A conversation started from a future Hermes surface can enter through the same Host conversation contract; Hermes-native session history remains a cache.
5. Hermes may call narrow Lumen plugin tools. The Host validates every intent and remains the only authority for grants, approvals, target selection and canonical mutation.
6. Existing-Hermes installation supports a pinned plugin plus bundled skill. Combined deployment installs those same artifacts, rather than a fork or a different integration.

### Security and privacy

1. Ordinary conversation uses `conversation.chat/respond`, not the broad `agent.run/execute` capability.
2. Ordinary conversation is zero-tool and zero-Hermes-memory by default. It fails before `POST /v1/runs` unless the active Hermes deployment can prove the exact profile.
3. Caller input cannot set or override instructions, session ID, conversation history, model, provider, toolsets, memory mode or runtime profile digest.
4. The Host projects no secrets and applies byte, message, token, turn and deadline limits before serialization.
5. The plugin never receives an owner credential or direct storage path. It receives only a plugin identity and short-lived, audience-bound capability material.
6. Plugin, skill, model, Hermes memory, tool output and external content are untrusted.
7. Installing or upgrading executable plugin code requires explicit owner approval and an immutable source commit. A natural-language prompt may guide installation but cannot silently authorize it.

### Operational

1. Uninstalling, disabling or upgrading the plugin cannot delete or corrupt the Space.
2. Losing Hermes session state cannot erase history; the Host can bind a new session and re-project bounded canonical context.
3. Profile, plugin, endpoint or certificate substitution fails closed and produces redacted actionable evidence.
4. Co-located processes use an owner-restricted Unix socket for plugin-to-Host calls. Remote processes use pinned mutual TLS. No raw owner socket is exposed over TCP.
5. Doctor reports Host conversation readiness, Hermes Runs readiness, plugin binding and profile certification independently.

## Architecture

```text
surface
   │ authenticated conversation command
   ▼
Lumen Host / Space authority
   ├── encrypted canonical conversations, messages and accepted context
   ├── persona and runtime-profile compiler
   ├── capability broker and approvals
   ├── HermesRuntimeAdapter v1 ── authenticated Runs/SSE/status ──► Hermes
   └── PluginBroker v1 ◄──────── Unix socket or pinned mTLS ───── Lumen plugin
                                                                    ├── Lumen tools
                                                                    ├── Lumen skill
                                                                    └── bounded hooks
```

The two arrows solve different problems:

- **Host to Hermes:** Lumen asks Hermes to reason using a Host-constructed request.
- **Hermes to Host:** Hermes proposes a conversation ingress, context request or action through Lumen tools. A proposal is not authorization.

No adapter points at Space storage. Both adapters operate through versioned application contracts.

## Deployment topologies

### Co-located existing Hermes — primary

The owner already operates Hermes on a machine or VPS. Lumen installs separately and does not adopt Hermes lifecycle.

- Hermes keeps its own home, model credentials, plugins and service manager.
- Lumen keeps its own encrypted data directory, owner credential and service manager.
- Host calls the authenticated loopback Hermes Runs endpoint.
- The Lumen plugin calls an owner-restricted Lumen Unix socket distinct from the owner control socket.
- File permissions and process identities prevent either component from opening the other's state.
- Lumen setup verifies the Hermes endpoint, pinned plugin identity and profile certificate, but `lumen service` controls only Lumen.

### Combined managed installation — first-class, packaging deferred

One installer later provisions Lumen Host, pinned upstream Hermes and the same Lumen plugin/skill. Separate containers, volumes, credentials and process identities preserve the co-located contract. Combined lifecycle commands may control both services, but neither gains the other's state directory.

The conversation implementation and conformance suite must run unchanged in this topology. Packaging automation may not introduce a private integration API.

### Cross-machine external — secondary

Host and Hermes use the same contracts over pinned mutual TLS. Both endpoints must be reachable or connected through an explicitly supported owner-controlled network. NAT traversal and a Host-initiated reverse connector are deferred to the node-fabric milestone; setup must state this prerequisite rather than imply URLs create reachability.

## Distribution and installation

Lumen publishes three independently versioned artifacts:

1. `lumen-host` — Space authority and brokers.
2. `lumen-hermes-plugin` — Hermes-native Python plugin with tools, hooks and bundled skill.
3. `lumen-hermes-install` — signed manifest and onboarding skill/installer metadata binding compatible Host, plugin, Hermes and contract versions.

The supported existing-Hermes command is based on Hermes's native pinned installation:

```text
hermes plugins install <lumen-plugin-repository> --ref <full-commit> --enable
```

A user may ask Hermes to install a Lumen skill from an immutable HTTPS URL and follow it. The skill must display the source, pinned revision, requested network access and required restart before requesting owner confirmation. Pairing uses a short-lived code or signed challenge; API keys, private keys and long-lived credentials never appear in the prompt.

The plugin is disabled by default until installation, explicit enablement, pairing and compatibility checks all succeed. Later updates require another exact commit and support rollback to the prior installed revision.

## Canonical data model

The existing encrypted Space state gains versioned records:

- `Conversation`: ID, owner, surface, persona, status, timestamps, next sequence and size accounting.
- `Message`: ID, conversation, sequence, role, author, surface, content, content digest, timestamp and optional task link.
- `Persona`: ID, schema version, version, immutable instructions, digest and status.
- `Surface`: stable source identity and type; it never becomes the owner.
- `ContextRecord`: namespace, schema version, provenance, classification, retention, digest and typed content.
- `RuntimeSessionMapping`: conversation, runtime identity, opaque Hermes session ID, persona digest, profile digest, Host epoch and last projected sequence.
- `RuntimeProfile`: exact persona, provider/model constraints, allowed feature set, memory policy, context bounds, turn/token/deadline budgets and digest.
- `RuntimeCertification`: runtime and endpoint identity, artifact digest and process identity fields, plugin identity, profile digest, effective toolsets, memory behavior, limits, evidence and expiry.

Message content is canonical only after a Host transition commits it. Hermes output, sessions and memory remain evidence or caches.

## Conversation lifecycle

### Start and send

1. Owner submits `conversation send` with request ID, conversation ID or explicit create intent, surface ID and message.
2. Host validates identity, epoch, UTF-8, size, retention and `conversation.chat/respond` policy.
3. Host atomically persists the user message, task, idempotency record, session mapping and runtime-create intent before network I/O.
4. Host compiles the immutable Lumen persona and bounded canonical history. It computes the profile and context digests.
5. Host requires a fresh exact runtime certification. Missing, stale or contradictory evidence records `runtime_profile_unverified`; no Hermes run is created.
6. Host submits the run and persists the runtime mapping.
7. Host consumes the sole event stream and reconciles terminal status. Tool or memory events in a zero-tool profile fail the run and invalidate certification.
8. Only verified completion atomically appends one deterministic assistant message and completes the task.

`RequestID` is the idempotency key. Repeating identical serialized intent returns the recorded result. Reusing the key with different content rejects. The assistant message ID derives from the task ID, preventing duplicate terminal events from appending twice.

### Continue after restart or replacement

The next turn is constructed from canonical messages, regardless of Hermes session availability. If the endpoint, runtime or profile changes, the Host creates a new opaque Hermes session mapping and supplies bounded canonical history. No automatic import from Hermes memory occurs.

If completion cannot be proven within the reconciliation deadline, the task becomes `unknown_outcome` and no assistant success message is added.

## Persona and personalization

The initial persona is immutable, versioned and owned by Lumen. Its minimum behavioral contract is:

- identify as Lumen, the user's private Space intelligence;
- use only the canonical context supplied by the Host;
- distinguish known context from inference;
- never claim memory, permissions, tools or completed actions not represented by Host records;
- ask when ambiguity changes privacy, authority or target selection;
- treat Hermes and its features as implementation details unless operational detail is requested.

Personalization does not mean concatenating arbitrary user files into a system prompt. The Host selects typed accepted `ContextRecord`s under classification and retention policy, labels their provenance, applies the requested projection level and enforces the profile bounds before serialization.

For the first slice, personalization supports a small `user.preferences/v1` record with owner-editable preferred name, communication style and locale. Memory extraction and automatic acceptance remain deferred.

## Runtime profile and certification

The default profile is `lumen.chat.default/v1`:

- effective Hermes toolsets: empty;
- Hermes memory read/write: disabled;
- delegation, MCP, browser, shell, files, web and automation: disabled;
- maximum turns: 1;
- bounded canonical message count and serialized bytes;
- bounded input, output and total tokens;
- bounded deadline;
- one selected model/provider pair permitted by policy.

The profile digest covers every field. The Host generates it; callers and the plugin cannot override it.

Current upstream Hermes capability discovery does not attest the effective per-run toolset or memory behavior. Inspection of pinned Hermes `v2026.9.7` (`2237be355906fbe6065ce1815711eee52b2d646e`) found that `pre_tool_call` callback exceptions are logged and ignored in `hermes_cli/plugins_dispatch.py`, and the tool executor also continues if hook dispatch raises. A plugin hook therefore cannot certify zero-tool execution. The Host must keep conversation content blocked until a dedicated runtime profile proves, through pinned artifact/configuration evidence and a real negative tool/memory probe, that no tools or memory can execute. The plugin may supply telemetry, but its self-reported certificate is insufficient. The Host additionally rejects unexpected tool/memory events and keeps the conversation capability free of side effects.

Certification is bound to endpoint identity, Hermes version, plugin commit, profile digest, relevant configuration digest and expiry. A vanilla or mismatched Hermes endpoint can continue serving generic `agent.run/execute`, but it cannot serve `conversation.chat/respond`.

If an upstream Hermes release cannot reliably enforce the zero-tool profile through supported configuration and fail-closed hooks, that release is incompatible for chat without a separately pinned and reviewed isolation patch. [D-058](../../DECISIONS.md) records that narrow exception for `v2026.9.7`; the Host still requires deployed artifact, process, configuration, and endpoint binding before issuing a certificate. The general-purpose runtime cannot serve chat merely because it implements the Runs API.

## Hermes plugin contract

### Tools

- `lumen_capabilities_list`: returns redacted, already-authorized feature descriptions; never grants them.
- `lumen_conversation_turn`: submits external-surface text and correlation metadata to a mapped Lumen conversation.
- `lumen_context_request`: requests one declared projection level and namespace; Host may reduce or deny it.
- `lumen_action_request`: proposes a typed capability action and target constraints; Host chooses the target and approval path.
- `lumen_task_status`: returns bounded task state.
- `lumen_task_cancel`: requests cancellation; terminal outcome still comes from Host reconciliation.
- `lumen_approval_submit`: forwards an exact owner decision bound to the immutable request.

No `state.put`, raw SQL/store access, arbitrary node selection, unrestricted broadcast or generic shell tool exists.

### Hooks

- A `pre_tool_call` block may provide defense in depth, but it is not the zero-tool security boundary on pinned Hermes `v2026.9.7` because callback errors fail open.
- Session hooks map Hermes surface correlation to opaque plugin session identity; they do not define canonical conversation identity.
- LLM/tool observer hooks emit bounded redacted evidence for certification and reconciliation.
- Hooks cannot accept memory or approvals and cannot expand the active capability.

### Broker authentication

Co-located plugin calls use a dedicated Unix socket with peer-credential and file-permission checks. Remote calls use mutual TLS and pinned identities. Each request carries protocol version, plugin identity, Space reference, Host epoch, request ID, timestamp, expiry, nonce, task/conversation correlation and an action/context digest. Replay, stale epoch, unknown plugin, altered digest and expired requests reject before payload processing.

## Public Host APIs

The first operator surface adds:

```text
lumen-host conversation create --request_id ... --conversation_id ... --surface_id ...
lumen-host conversation send --request_id ... --conversation_id ... --surface_id ... --input ...
lumen-host conversation show --conversation_id ...
lumen-host preference set --request_id ... --name ... --value ...
```

Public conversation commands never accept Hermes instructions, session IDs, provider/model overrides or profile digests. The existing raw task commands remain an operator/developer surface and are not described as Lumen chat.

The plugin broker is a separate protocol from the owner control socket. Its exact wire schema is versioned before implementation and exposes only the tools above.

## Failure behavior

| Failure | Required result |
| --- | --- |
| Plugin missing or disabled | Conversation readiness reports action required; canonical history remains available. |
| Profile certification missing/mismatch | No Hermes request; durable `runtime_profile_unverified`. |
| Hermes unavailable before create | User message remains canonical; task fails or remains safely retryable under its recorded intent. |
| Create outcome uncertain | Task becomes `unknown_outcome`; no blind redispatch and no assistant message. |
| Duplicate/reordered events | One monotonic task outcome and at most one assistant message. |
| Unexpected tool or memory event | Fail/stop run, invalidate certification and record redacted audit evidence. |
| Context exceeds a bound | Reduce according to deterministic projection rules or reject before serialization; never silently send more. |
| Plugin/Host disconnect | Plugin returns an honest unavailable/unknown result; it cannot manufacture completion. |
| Hermes session deleted | Next turn uses a new mapping plus bounded canonical history. |
| Host restart | Encrypted conversation, messages, mapping, profile and task intent reload before readiness. |
| Migration interruption | Prior readable state remains authoritative; no partial schema becomes active. |

## Migration

Keep the encrypted envelope format independent from the Space schema. Introduce an explicit current schema version and a pure v1-to-v2 migration that adds empty conversation/context maps and the default persona/profile without changing Space, owner, Host, node, grant, task or audit identities.

Migration writes a newly encrypted temporary state with a fresh nonce, fsyncs it and its parent, verifies reopening, atomically replaces the active state and retains a recoverable prior version until the new state is proven. Failure before replacement leaves v1 readable. Migration is idempotent and future schemas fail closed.

## Observability

Structured redacted events include conversation/task IDs, profile/context/certification digests, runtime identity, bounded usage, lifecycle state and actionable failure code. They exclude message content, persona text, preferences, credentials and raw tool output by default.

Doctor reports separate states for canonical store, conversation schema, Host broker, Hermes Runs compatibility, plugin binding and chat-profile certification. `ready` requires every state needed by the selected journey; generic Hermes health alone is insufficient.

## Test strategy

### Acceptance

1. Set an accepted preferred name and style.
2. Send “Introduce yourself and greet me.”
3. Verify the response identifies as Lumen and uses only the accepted preference.
4. Send “Remember COBALT RAVEN” and receive a canonical response.
5. Remove Hermes session state or replace the runtime session.
6. Restart the Host.
7. Ask for the phrase and verify continuity comes from canonical Lumen history.
8. Inspect conversation history and confirm one ordered user/assistant pair per turn.

### Negative and privacy

- Caller-supplied instructions/session/model/profile are rejected before dispatch.
- Vanilla or stale-certification Hermes receives no conversation content.
- Ordinary chat exposes no web, memory, file, terminal, browser, MCP or delegation tool.
- Unexpected tool/memory events fail the run and cannot change canonical state.
- Unaccepted preferences and unrelated conversations are absent from the serialized request.
- Oversized, malformed and unauthorized input rejects without partial persistence.
- Deleted or expired context never reappears from Hermes session or memory retrieval.

### Recovery

- Duplicate send and completion events do not duplicate messages or runs.
- Crash before and after runtime create preserves intent without blind retry.
- Lost SSE reconciles by status; uncertain completion creates no assistant success message.
- Host restart preserves sequence and canonical history.
- Hermes restart/session deletion creates a replaceable mapping and continues from bounded canonical history.
- Failed v1-to-v2 migration preserves the prior readable state.

### Deployment conformance

Run the same conversation journey against:

1. existing co-located Hermes plus separately supervised Lumen Host;
2. combined isolated containers using the same plugin and contracts;
3. reachable cross-machine Hermes/Host with pinned mutual TLS when infrastructure is available.

Synthetic compatibility tests do not replace at least one real-model proof. Real evidence records versions, artifact and profile digests, topology, exact journey, expected/actual outcomes and redacted logs.

## Delivery sequence

1. Canonical conversation/message/persona/profile records and safe state migration.
2. Host conversation commands, deterministic context projection and atomic task/message lifecycle.
3. Separate pinned Lumen Hermes plugin/skill and dedicated broker protocol.
4. Runtime-profile certification and zero-tool chat enforcement.
5. Existing-Hermes co-located setup and real manual journey.
6. Combined-container conformance using the same artifacts.
7. Context preferences, deletion/restart evidence and normalized usage events.
8. Cross-machine conformance when its network preconditions exist.

Schema, persistence, migration and authority changes execute serially. Read-only discovery, fixtures and independent reviews may use Luna agents in parallel.

## Trade-offs and rulings

- **Plugin plus skill instead of fork:** preserves upstream upgrades and supports existing installations. The cost is explicit compatibility certification for each supported Hermes release.
- **Plugin API before generic MCP:** avoids duplicating a single known integration. The broker contract may later be exposed through MCP when a second runtime needs it.
- **Canonical history on every bounded request:** costs tokens but makes Hermes sessions replaceable. Deterministic projection and later summarization control the cost.
- **Fail closed on uncertified chat:** may temporarily make generic Hermes usable while Lumen chat is unavailable. This is preferable to silently granting web, shell, files or memory.
- **Co-located first:** matches likely self-hosting economics. Separate identities, sockets and state preserve the boundary without requiring another VPS.
- **Combined packaging later:** avoids blocking the conversation contract on installer polish while requiring the same conformance suite from the start.

## Exit criteria

This slice is complete only when:

- the manual conversation identifies as Lumen and uses an accepted preference;
- continuity survives Host restart and Hermes session deletion from canonical encrypted history;
- ordinary chat cannot invoke Hermes memory, web, files, terminal, browser, MCP or delegation;
- caller-controlled runtime configuration is rejected;
- plugin uninstall or Hermes replacement preserves the Space;
- co-located existing-Hermes and combined-container journeys use the same versioned plugin and Host contracts;
- migration, acceptance, negative, privacy, recovery and real-runtime evidence pass with no unresolved Critical or Important finding.
