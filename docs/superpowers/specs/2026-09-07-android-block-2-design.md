# Android Block 2 design

## Purpose and scope

Block 2 makes the reference Android phone an honest, user-operated Lumen Host and a portable companion surface. The completed block creates and recovers a private Space, exposes current Host health, pairs one local node, carries one authenticated command across a local network, and presents a useful desk companion without silently activating hardware.

The Android implementation must remain only one deployment of the Host contract. Any paired node may later provide a companion surface or become Host through the explicit migration contract.

## Product journey

1. The owner creates or reopens the encrypted Space on the phone.
2. They explicitly start Host coordination and see whether it is ready, degraded, stopped, or unavailable.
3. They open pairing, inspect a short authentication string on both devices, and explicitly confirm it.
4. A paired node discovers the Host on the local network, authenticates it, sends one idempotent command, and receives its durable outcome.
5. The owner backgrounds, restarts, changes Wi-Fi, or stops the service; the companion reports the resulting state truthfully.
6. The companion can speak a local status summary. Camera and microphone remain off until the owner activates the corresponding foreground capability and accepts Android's runtime permission.

## Non-goals

- No unattended server guarantee, automatic failover, internet relay, push wake-up, or background camera/microphone capture.
- No raw media leaves the phone, is persisted, or is sent to a runtime in this block.
- No Host starts from `BOOT_COMPLETED`: Android 15+ prohibits boot receivers from starting a `dataSync` foreground service, and `dataSync` services have a six-hour per-24-hour allowance. The user restarts coordination from the companion after reboot or timeout. [Android foreground-service behavior](https://developer.android.com/about/versions/15/behavior-changes-15)
- No generic transport or crypto abstraction. The implementation has one versioned local channel with narrowly defined pairing and envelope rules.

## Host lifecycle and health

`LumenHostService` owns the volatile runtime state. The encrypted Space state remains the sole authority for canonical Space data.

```text
stopped → opening → ready
                  ↘ degraded
any state → stopping → stopped
```

- `opening`: foreground notification appears; the service opens the encrypted store and applies portable restart recovery before it can become ready.
- `ready`: storage and local listener are ready. The service advertises only a non-sensitive mDNS record.
- `degraded`: a failed listener, local-network denial, clock issue, storage/key error, or Android service timeout is shown with a user-actionable reason. New cross-node commands are refused; no prior result is rewritten.
- `stopped`: service is absent; new cross-node commands cannot be accepted.

The companion observes this runtime state and rechecks the encrypted store on launch. It never infers readiness merely because a state file exists. The foreground service handles the platform timeout by recording a redacted health transition, stopping itself, and prompting the owner to restart it from the app.

## Identity, pairing, and transport

The Host creates a non-exportable Android Keystore P-256 signing identity. The public key fingerprint is stored in encrypted Space state; a hardware-backed implementation is used when available but is not assumed on the old phone. Hardware attestation is optional evidence, not a requirement for pairing. [Android key attestation](https://developer.android.com/privacy-and-security/security-key-attestation)

Pairing is an owner-visible session:

1. The Host creates a short-lived pairing session and displays a QR code plus a human-comparable authentication string.
2. The joining node scans or enters the session data, creates its own signing identity, and displays the same authentication string.
3. The owner confirms both strings. Only then does the Host persist that node's public key and membership through the portable Space boundary.
4. A failed, expired, mismatched, or cancelled session leaves no paired node or transport trust record.

The Android Host advertises `_lumen._tcp` through `NsdManager`. Discovery carries service version, a random Host instance ID, the Space ID, and port only. It is never authentication. On API 37, local-network access is requested through Android's service picker where possible; a broad local-network permission is requested only when required for hosting or the intended connection path. [Android NSD](https://developer.android.com/reference/android/net/nsd/NsdManager)

The connection uses one mutually authenticated encrypted channel. A session binds both long-term public keys, the Space ID, active Host epoch, protocol version, ephemeral key agreement, and the pairing transcript. Each frame is encrypted and authenticated, then contains a versioned envelope:

```text
spaceId · hostEpoch · sender · recipient · messageId · taskId?
issuedAt · expiresAt · nonce · payload
```

Before applying a state-changing envelope, the Host verifies the session identity, membership, epoch, recipient, time bounds, nonce, and schema version. It persists the idempotency/message result through `SpaceHost` before it acknowledges the command. A duplicate returns that durable outcome; a malformed, expired, replayed, revoked, or stale-epoch frame fails closed.

## Companion and hardware

The companion has four status areas: Host health, paired nodes, recent tasks, and pending approvals. It uses Material 3 semantic colors and text labels, native controls, a 48dp minimum touch target, dynamic color/dark-mode support, and an explicit state beside every action.

Local speaker output is a foreground, user-invoked `companion.speak_status` action using Android text-to-speech. It needs no microphone, camera, or network permission.

`companion.listen` and `companion.camera_preview` are separately advertised capabilities, default-denied, and exposed only as foreground controls. Activating either explains the data boundary and requests Android's corresponding runtime permission. Audio is used only for the chosen local interaction and discarded afterward; camera preview is not captured or uploaded. A later task capability must separately request any recording, storage, transcription, or sharing.

## Failure behavior

| Failure | User-visible result | Authority behavior |
| --- | --- | --- |
| Keystore/store unavailable | `Storage unavailable` with retry guidance | No Host listener or command acceptance. |
| Local network denied/lost | `Local network unavailable` | Advertisement/listener stops; no fallback to unauthenticated networking. |
| Pairing mismatch/expiry | `Pairing was not completed` | No node key or membership is stored. |
| Channel authentication/replay failure | `Connection rejected` in redacted health/audit view | Frame is dropped before command handling. |
| Android foreground timeout/service stop | `Host stopped — restart required` | Listener ends; cross-node work is unavailable, never marked complete. |
| Microphone/camera denied | Permission-specific explanation and settings link | Capability remains denied; hardware remains off. |

## Validation

- Unit and contract tests: codec migration/recovery, health state transitions, pairing expiry and mismatch, envelope authentication/replay/expiry/version/epoch failures, duplicate command outcome, revocation, and permission-denial models.
- Android tests: Keystore store behavior, service start/stop/degraded notification, runtime permissions, and companion rendering.
- Physical Xiaomi Android 13 evidence: create/reopen Space; start/stop Host; background/reopen; Wi-Fi loss/reconnect; USB-installed update preserving state; one paired-node command; speaker invocation; microphone/camera permission denial and grant paths.
- Block exit: five lifecycle runs recorded without manual state repair, matching the delivery plan.
