# O-001 stack comparison spike

> **Historical spike.** This comparison informed `D-020`; it does not define the current Host implementation, product roadmap, or required client platforms.

## Purpose

This spike recorded the comparison that selected the original portable-core approach:

1. Kotlin Multiplatform code shared by Android and Apple targets.
2. Separate native implementations sharing JSON Schema contracts and golden fixtures.

The spike is disposable evidence, not production protocol. `D-020` in [DECISIONS.md](./DECISIONS.md) records the accepted historical choice and its current scope. The later native Go Host supersedes Kotlin as the production Space authority; portable protocol fixtures and native client boundaries remain useful conclusions.

## Common contract

Both prototypes consume the files in [`spikes/o-001/fixtures/`](../spikes/o-001/fixtures/) and must produce the same result for each case:

| Fixture | Expected result | Reason |
| --- | --- | --- |
| `pairing-request.valid.json` | Accept | Version, shape, and lifetime are valid at the fixed evaluation time. |
| `pairing-request.expired.json` | Reject | The message expiry is at or before the fixed evaluation time. |
| `pairing-request.unsupported-version.json` | Reject | The schema version is not supported. |
| `pairing-request.unknown-field.json` | Reject | Security-sensitive envelopes fail closed on unknown fields. |

The fixed evaluation time is `2026-09-04T12:00:00Z`. Implementations must not depend on the wall clock for this comparison.

Implementations must parse timestamps strictly, require `issuedAt < expiresAt`, and reject `expiresAt <= evaluation time`; they may not rely on a JSON Schema library treating `format` as an assertion.

Both prototypes also parse `task-events.sse` in that fixture directory and must preserve event IDs, event types, and JSON data without interpreting data as authority.

This fixture deliberately omits signatures, key proof, and durable nonce storage. Cryptographic message authentication and replay rejection remain required by the V1 threat model, but are separate comparison tracks because they need platform keystores and persistent state. Passing this fixture must not be reported as authentication or replay evidence. SSE reconnection, duplicate IDs, and malformed streams likewise remain pending beyond the parser comparison.

## Measurements

Record evidence in `EVIDENCE.md` using the same machine and warm/cold conditions:

- clean setup commands and elapsed time;
- compile and test commands and elapsed time;
- binary or framework size where applicable;
- amount of shared versus platform-specific source;
- JSON strictness and error clarity;
- Swift interop ergonomics;
- debugger and failure-trace quality;
- database migration and encrypted-store integration effort;
- Android background-service integration effort;
- SSE reconnect and event parsing effort;
- packaging limitations and required IDE/toolchains.

Headless checks established only the recorded prototype behavior. Pending Android or Apple rows gate only a future product surface that relies on those prototypes; they do not block the current journey-first roadmap or the combined Linux/VPS reference deployment.

## Environment

Repository tools are pinned in [`mise.toml`](../mise.toml). Run tools through `mise exec -- ...` so global installations do not affect the results. Apple CLI validation uses the system Swift toolchain because Xcode/Swift is platform-managed rather than installed by mise.
