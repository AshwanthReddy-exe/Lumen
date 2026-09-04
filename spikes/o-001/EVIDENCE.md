# O-001 comparison evidence

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

All fixtures contain synthetic identifiers and content. This document records prototype evidence only and does not resolve `O-001`.

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

## Security evidence still pending

- Device-key generation, signature creation and verification, and pairing key proof using platform-backed keys.
- Durable nonce persistence and replay rejection before and after restart.
- Invalid signature, unknown key, altered identity, and malformed timestamp fixtures.
- SSE malformed events, duplicate IDs, disconnect/reconnect, and last-event replay.

## Manual evidence still required

The headless comparison is insufficient to accept `O-001`. Before the decision, run and record:

1. Android foreground-service and encrypted-store prototypes on the intended old-phone class of device.
2. Native Apple encrypted-store and migration prototypes with full Xcode on macOS and iOS.
3. KMP-to-Swift framework integration, startup, debugging, and packaging through a minimal native UI.
4. SSE disconnect/reconnect and last-event replay in both strategies.
5. Installation, startup time, UI responsiveness, logs, and debugging observations using the evidence format in `PLAN.md`.

## Decision status

`O-001` remains open. Record the selected strategy, evidence, consequences, and reversal trigger in `DECISIONS.md` only after the pending platform checks and owner review are complete.
