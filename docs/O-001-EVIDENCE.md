# O-001 comparison evidence

> **Historical evidence.** These measurements describe disposable 2026-09-04 prototypes. They are retained for auditability and must not be read as current production implementation or roadmap gates.

## Test environment

| Field | Value |
| --- | --- |
| Date | 2026-09-04 |
| Machine | Apple Silicon Mac |
| macOS | `26.6.2` (`25G83`) |
| Java | mise `25.0.2` |
| Gradle | mise `9.5.0` |
| Kotlin | `2.4.10` Gradle plugin |
| Swift | System Swift `6.3.3` |
| Android SDK/device | Not available; pending |
| Full Xcode/iOS simulator | Not available; pending |

All fixtures contain synthetic identifiers and content. This document records prototype evidence only; `O-001` is resolved separately as `D-020`. The production Space authority later moved to the native Go Host.

## Automated contract results

| Check | Kotlin Multiplatform | Native Swift/schema |
| --- | --- | --- |
| Valid pairing request | Pass | Pass |
| Expired request | Pass | Pass |
| Unsupported schema version | Pass | Pass |
| Unknown top-level field | Pass | Pass |
| Unknown nested field | Pass | Pass |
| Non-increasing timestamps | Pass | Pass |
| SSE IDs, types, and data preserved | Pass | Pass |
| Headless check | `mise exec -- gradle -p spikes/o-001/kmp jvmTest --no-daemon` | `swift run --package-path spikes/o-001/native-schema NativeSchemaSpikeChecker spikes/o-001/fixtures` |
| Full native test suite | Not applicable to JVM-hosted check | Blocked: Command Line Tools SDK has no XCTest; full Xcode pending |

## Comparison

| Criterion | Kotlin Multiplatform | Native Swift/schema |
| --- | --- | --- |
| Clean setup time | Pending | Pending |
| Cold compile/test time | Pending controlled measurement | Pending controlled measurement |
| Warm compile/test time | Pending controlled measurement | Pending controlled measurement |
| Produced binary/framework size | JVM spike JAR: 12,864 bytes | Debug checker executable: 314,064 bytes; not comparable to a framework |
| Shared source | 84 lines in `commonMain` | JSON Schema and golden fixtures only |
| Platform-specific source | 54 lines of JVM-hosted tests | 118 library lines, 47 checker lines, 40 XCTest lines |
| Strict decoding and error clarity | Explicit shape/type/semantic checks; failures normalized to a reason | Codable plus dynamic-key and semantic checks; validation errors preserve a message |
| Swift interop | Pending full Xcode check | Native |
| Debugging and failure traces | Pending | Pending |
| Database migration integration | Pending platform prototype |
| Encrypted-store integration | Pending platform prototype |
| Android background lifecycle | Pending Android SDK/device |
| SSE reconnect behavior | Parser only; reconnect pending |
| Packaging | Pending platform toolchains |

The artifact sizes and source counts are descriptive only. They use different artifact types and do not establish a winner.

## Security evidence not established by this spike

- Device-key generation, signature creation and verification, and pairing key proof using platform-backed keys.
- Durable nonce persistence and replay rejection before and after restart.
- Invalid signature, unknown key, altered identity, and malformed timestamp fixtures.
- SSE malformed events, duplicate IDs, disconnect/reconnect, and last-event replay.

## Historical follow-up evidence

The headless comparison was insufficient to qualify either platform prototype. If a future surface adopts one of these paths, run and record the relevant evidence before that surface ships:

1. Android foreground-service and encrypted-store prototypes on the intended old-phone class of device.
2. Native Apple encrypted-store and migration prototypes with full Xcode on macOS and iOS.
3. KMP-to-Swift framework integration, startup, debugging, and packaging through a minimal native UI.
4. SSE disconnect/reconnect and last-event replay in both strategies.
5. Installation, startup time, UI responsiveness, logs, and debugging observations using the evidence format in [PLAN.md](./PLAN.md).

## Decision status

`O-001` was accepted as `D-020`: Kotlin Multiplatform shared the original portable protocol and Space rules while Android and Apple applications remained native. The native Go Host now owns production Space authority. This evidence remains useful for protocol-fixture and client-boundary history; pending platform checks gate only surfaces that actually adopt those implementations.
