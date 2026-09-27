# Constraints

## Personal-alpha release sequence
- source: docs/PLAN.md
- type: protocol
- content: Personal alpha includes one Space, one owner, an always-on Host, Hermes, web administration, phone conversation and approval surface, Mac execution node, inspectable memory, Android desk voice, recovery, and measured latency. Messaging, Apple abilities, controlled browsing and coding follow in expanded personal beta; paid self-hosted and managed releases follow later.

## Host authority and node execution
- source: docs/ARCHITECTURE.md
- type: protocol
- content: The Space Host owns canonical shared state, policy, routing, approvals, task truth, and audit history. A target node revalidates the exact capability invocation and local permission before execution.

## Host and Hermes separation
- source: docs/ARCHITECTURE.md
- type: protocol
- content: Hermes is the intelligence runtime behind a versioned adapter. Runtime output is untrusted evidence; the Host owns grants, approval, canonical context and final task outcomes.

## Conversation records
- source: docs/PLAN.md
- type: schema
- content: A canonical conversation consists of messages and product events independent of Hermes session IDs. Required records are Conversation, Message, Surface, Task, RuntimeBinding, CommandReceipt, ConversationEvent, ContextRecord, MemoryProposal, ApprovalRequest, InteractionRequest, Node, Capability, Grant, Invocation, ExecutionReceipt, and ArtifactReference.

## Message acceptance and retry
- source: docs/PLAN.md
- type: protocol
- content: Every accepted message stores its receipt and execution intent before runtime I/O. Retry with the same command ID returns the same receipt; reuse with different content is rejected. One conversation has one active turn initially, with new messages queued by default.

## Memory acceptance and projection
- source: docs/PLAN.md
- type: protocol
- content: Memory begins with explicit user preferences and clearly proposed facts. Accepted records carry provenance, scope, sensitivity, retention or expiry, revision, permitted destinations, and confirmation state. Authorization and destination policy apply before context serialization.

## Persistence selection gate
- source: docs/PLAN.md
- type: nfr
- content: The existing encrypted snapshot store remains the foundation baseline. A transactional local database is experiment-gated against atomic message, receipt, task-intent and event-cursor commits; crash recovery; no plaintext leakage; portable builds and licensing; and measured alpha-volume performance.

## Personal-alpha connectivity
- source: docs/PLAN.md
- type: protocol
- content: Use Tailscale-assisted private reachability for personal alpha while retaining a transport-independent Lumen protocol. Use HTTPS request/response, SSE for browser events, and a persistent authenticated node connection for bidirectional dispatch. Introduce binary RPC only if measured requirements justify it.

## Enrollment and revocation
- source: docs/PLAN.md
- type: protocol
- content: Pairing uses a short-lived single-use invitation, a device-generated key, owner confirmation, membership and scoped credentials. A short human code is not the sole cryptographic secret. Host revocation blocks future access after commitment; cached offline grants remain bounded by expiry.

## Approval and clarification
- source: docs/PLAN.md
- type: protocol
- content: Approval binds one exact action, arguments, destination, task, expiry and grant revision. Clarification supplies input without new authority. Both use durable records and one accepted resolution across surfaces.

## Cross-node effects
- source: docs/PLAN.md
- type: protocol
- content: Persist invocation before dispatch. The node checks identity, epoch, expiry, resource scope and local permission, then durably records receipt before an effect. Delivery is at least once with idempotent commands; uncertain external effects require reconciliation before retry.

## Alpha latency targets
- source: docs/PLAN.md
- type: nfr
- content: Alpha targets include local UI acknowledgement p95 under 100 ms, Host acceptance excluding WAN p95 under 100 ms, Lumen overhead before runtime dispatch p95 under 150 ms, connected approval appearance p95 under 1 second, local playback stop p95 under 150 ms, first response audio p50 under 2.5 s and p95 under 5 s, and small missed-history replay p95 under 5 s. These are targets, not measured claims.

## Phase exit evidence
- source: docs/PLAN.md
- type: nfr
- content: Every phase completes only when requirement mappings, automated checks, negative cases, recovery evidence and relevant owner demonstration pass. Missing device or credential means blocked or unverified.

## Headless Host service
- source: docs/PHASE-2-HOST-HERMES.md
- type: protocol
- content: The native Go Host has create-once initialization, foreground serve, readiness, graceful shutdown, owner-restricted authenticated Unix-domain operator socket, bounded requests and redacted diagnostics. A supervisor manages restart outside the process.

## Encrypted Host state
- source: docs/PHASE-2-HOST-HERMES.md
- type: schema
- content: Version 1 uses an encrypted envelope with format version, key identifier, cipher suite, fresh nonce, authenticated metadata and ciphertext. AES-256-GCM requires a new 96-bit nonce per write. Unknown versions, corrupt ciphertext, missing keys, permissive access, lock contention and failed writes fail closed without overwriting committed state.

## Hermes Runs adapter
- source: docs/PHASE-2-HOST-HERMES.md
- type: api-contract
- content: The versioned adapter uses capabilities and health endpoints, run submission, run status and bounded events, and approval, steer and stop only after exact Host authorization. An indeterminate run creation is not retried after restart without a recovered mapping and resolves to unknown_outcome when evidence is insufficient.

## Runtime isolation
- source: docs/PHASE-2-HOST-HERMES.md
- type: protocol
- content: Hardened Linux, macOS and VPS deployments isolate Host and Hermes with separate principals or containers and pinned authenticated TLS endpoint identity. Same-UID Termux Hermes is compatibility-only evidence; hardened Termux proof uses isolated Hermes over pinned mutual TLS.

## Security invariants
- source: docs/THREAT_MODEL.md
- type: protocol
- content: Policy defaults to deny before dispatch and execution. Local execution retains local capability policy; cross-node work requires Host authorization. Adapters cannot grant authority. Exact one-time approval is not a durable grant. Unknown security-critical versions and fields fail closed.

## Release-blocking security evidence
- source: docs/THREAT_MODEL.md
- type: nfr
- content: Protocol, policy, recovery, context, lifecycle, capability-abuse, Host/Hermes, managed-isolation, presence, component, file-access and runtime-registry fixtures gate their respective releases. An owner reviews unresolved blocking threats at each phase exit.
