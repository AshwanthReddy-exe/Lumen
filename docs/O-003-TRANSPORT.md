# O-003 local transport decision

## Status

Accepted as `D-022`. This is the smallest local-network transport baseline.
Remote relay, push wake-up, and internet traversal are explicitly deferred to
the later remote-access phase.

## Decision

- Discover the Host with mDNS on the local network. The advertisement contains
  only a protocol/service name, Space identifier, Host instance identifier,
  protocol version, and connection endpoint. It never contains credentials,
  grants, context, or task payloads.
- Treat discovery as a hint, not authentication. A node connects only to a
  previously paired Host identity and verifies the advertised Space and
  endpoint against its configured record.
- Use one mutually authenticated, encrypted bidirectional channel for node to
  Host traffic. The paired device identities authenticate both ends; transport
  encryption protects messages in transit. The exact platform API and crypto
  library remain implementation details and must not alter this contract.
- Send versioned envelopes containing Space, active Host epoch, sender,
  recipient, message ID, task ID when applicable, issue time, expiry, nonce,
  and payload. The active epoch is authenticated with the envelope and must
  equal the Host's durable epoch. The Host is the authority for cross-node
  commands and persists a command before dispatching it.

## Message behavior

The receiver rejects an envelope when its Space, active Host epoch, sender,
recipient, version, signature/authentication, nonce, issue time, or expiry is
invalid. It records a message ID (or command idempotency key) before applying a
state-changing operation. A duplicate therefore returns the recorded outcome
and does not run the action again. Reordered events are accepted only when the
task state transition or event version permits them; arrival order never
overwrites canonical state silently.

The sender retries only while the envelope is unexpired and the operation's
delivery policy allows retry. Backoff is bounded. Cancellation stops retries;
it cannot claim that a remote action was cancelled unless the Host has a
durable cancellation result. If the connection drops after dispatch, the
outcome is `unknown_outcome` until reconciliation, never success by inference.

## Time behavior

V1 uses the platform wall clock; it does not claim an external trusted-time
service. Issue and expiry times are checked with no grace period. Each Host
persists the greatest wall-clock time it has accepted. If the current time is
earlier than that high-water mark, the Host enters `degraded` and refuses new
cross-node commands and scheduled runs until the wall clock has been corrected
past the recorded value. This favors a visible availability failure over
silently extending a grant or schedule. Nodes apply the same expiry checks to
received work and report a local clock problem rather than retrying indefinitely.

## Discovery and lifecycle

Discovery is opportunistic and short-lived: a node may rediscover after network
changes, but pairing is required before use. A Host restart keeps its identity
and durable task records, re-advertises only after storage and policy are
ready, and reconciles in-flight tasks after reconnect. A node that is revoked
is refused during handshake and loses future command and synchronization
authority immediately after the Host records revocation.

When the Host cannot be discovered or authenticated, the node reports
`offline` or `degraded` with a user-actionable reason. Same-device local work
may continue under its cached, unexpired local grant. Cross-node work is not
silently executed locally and is queued only when the user-selected delivery
policy explicitly permits it; otherwise it remains failed/unsubmitted.

## Non-goals for the local-network baseline

- Internet/NAT traversal, remote relay, push wake-up, or cloud presence.
- Discovery as a trust mechanism or unauthenticated fallback channel.
- Host migration, multi-Host consensus, or split-brain resolution.
- Streaming private context outside the selected synchronization policy.
- Best-effort execution without a durable command record.

## Acceptance checks

1. A paired node discovers the Host over mDNS, rejects a wrong Space or
   unpaired identity, and establishes an authenticated encrypted channel.
2. Discovery packets and connection logs contain no credentials, grants,
   context, prompts, or task payloads.
3. Wrong recipient, unsupported version, invalid authentication, expired
   envelope, reused nonce, and revoked node all fail closed.
4. Duplicate command delivery produces one durable execution and the same
   recorded outcome; reordered events cannot regress task state.
5. Disconnect before dispatch, after dispatch, and during reconciliation each
   produce the documented honest state (`failed/unsubmitted`,
   `unknown_outcome`, or the reconciled terminal state).
6. Host restart, Wi-Fi change, and temporary mDNS loss result in bounded
   reconnect attempts and an explicit offline/degraded status.
7. With the Host unavailable, a local task follows its cached grant while a
   cross-node task follows its explicit delivery policy and never bypasses
   Host authorization.
8. A backwards clock jump puts the Host in `degraded`; it accepts neither a
   cross-node command nor a schedule until time exceeds its durable high-water
   mark.
9. A stale or future Host epoch is rejected before dispatch, including after a
   completed migration and stale-Host reconnect.

## Reversal trigger

Revisit this decision only if physical-device tests show that mDNS or the
single authenticated channel cannot meet reconnection, battery, latency, or
security acceptance checks. Any replacement must preserve the same envelope,
authority, idempotency, expiry, and offline semantics.
