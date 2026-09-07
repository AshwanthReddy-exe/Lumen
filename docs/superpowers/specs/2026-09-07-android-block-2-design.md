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

Pairing is an owner-visible, rate-limited session. It has a random 128-bit ID, one active session, a two-minute monotonic expiry, and at most five verification attempts. Each side creates a fresh P-256 ECDH key, signs a length-prefixed and domain-separated transcript with its long-term P-256 signing key, and derives a six-digit SAS from HKDF-SHA-256 over the shared secret and transcript hash. The transcript contains roles, both long-term public keys, both ephemeral keys, random nonces, Space ID, session ID, and protocol version; it prevents reflection, substitution, and role confusion.

1. The Host creates a short-lived pairing session and displays a QR code plus a human-comparable authentication string.
2. The joining node scans or enters the session data, creates its own signing identity, and displays the same authentication string.
3. The owner confirms both strings. Only then does the Host atomically persist the node membership, public key, SHA-256 DER-SPKI fingerprint, and pairing session ID through its encrypted authority record.
4. The joining node remains pending until it receives a signed `PairingAccepted` containing Host epoch, membership version, and transcript hash. Lost acknowledgements are reconciled by an idempotent proof query; cancellation after Host commit revokes the node rather than claiming no record exists.
5. A failed, expired, mismatched, or cancelled pre-commit session leaves no paired node or transport trust record.

The Android Host advertises `_lumen._tcp` through `NsdManager`. Discovery carries protocol version, a random per-service-start Host instance ID, and port only; it never reveals a stable Space ID. It is never authentication. Since API 37 requires local-network permission for a Host that advertises/listens, Host mode requests `ACCESS_LOCAL_NETWORK` and stops/degrades on denial or revocation. Android's service picker is reserved for a client node selecting an outbound Host service. [Android NSD](https://developer.android.com/reference/android/net/nsd/NsdManager)

The connection uses TLS 1.3 with ALPN `lumen/1`, ECDHE, Host certificate SPKI pinning from pairing, and required/pinned client certificates after pairing. TLS resumption never grants authorization. A missing, invalid, or changed Host Keystore identity for the durable fingerprint moves the Host to degraded; it does not silently generate a replacement.

Each received frame has bounded length and carries strict UTF-8 JSON unsigned-envelope bytes plus a P-256/SHA-256 DER signature over `SHA256("lumen-envelope-v1" || unsignedEnvelopeBytes)`. Verification uses exact received bytes before parsing; unknown fields, payload schemas, and oversized frames fail closed. The envelope contains:

```text
kind · payloadSchemaVersion · spaceId · hostEpoch · sender · recipient
messageId · taskId? · issuedAt · expiresAt · nonce · payload
```

Before applying a state-changing envelope, the Host verifies the session identity, membership, epoch, recipient, signature, time bounds, nonce, and schema version. It first looks up `(senderId, messageId)`: the same envelope digest returns the recorded durable outcome; a different digest is an idempotency collision. For a new message ID, it atomically reserves the sender nonce and immutable digest with the accepted command before acknowledging it. Nonces remain reserved through expiry. A malformed, expired, replayed, revoked, or stale-epoch frame fails closed. Durable revocation closes every active session for that node and rejects later resumption.

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
