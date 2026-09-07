# Superseded Android foreground-Host slice

## Status

**Superseded.** This document records implementation and physical-test evidence from the first Android experiment. It must not guide new work. The experiment incorrectly placed canonical Host state and lifecycle inside the companion APK. [D-029](./DECISIONS.md) replaces that design with a headless terminal service; [PHASE-2-HOST-HERMES.md](./PHASE-2-HOST-HERMES.md) is the active Block 2 plan.

Do not extend `LumenHostService`, `HostRuntime`, `HostServiceLifecycle`, or `AndroidEncryptedSpaceStateStore`. The Android rewire will remove those Host responsibilities and retain only companion/node concerns. Existing encrypted state on the development phone is test data from the superseded topology and is not accepted as production Host authority.

## Implemented contract

- `AndroidEncryptedSpaceStateStore` keeps the portable `SpaceState` in app-private storage, encrypted with a non-exportable Android Keystore AES-GCM key and committed through `AtomicFile` replacement.
- The desk companion reads the store honestly. An empty store offers **Create your Space**; a present store offers **Start Host**; an unreadable store reports that it cannot start without resetting state.
- Creating a Space calls the portable `SpaceHost.create` boundary before starting anything. A generated Space, owner, and Host node identifier is durable inside that state; device-generated signing keys and node pairing are later work.
- `LumenHostService` starts as an Android `dataSync` foreground service, opens and applies portable restart recovery before it presents itself as active, and keeps an ongoing low-importance notification while it is active.
- This slice requests no microphone, camera, location, network, overlay, accessibility, or battery-optimization-exemption permission. It does not claim an authenticated node channel or network availability.

## Failure behavior

| Condition | Observable result | Authority behavior |
| --- | --- | --- |
| No Space state | The companion offers Space creation. | No Host accepts work. |
| Encrypted state cannot be read or written | The companion reports unavailable storage; it does not overwrite or reset the state. | The service stops and does not accept work. |
| Existing state opens | The service runs portable restart recovery before becoming active. | Queued portable tasks become `unknown_outcome` under the Phase 1 recovery contract. |
| Service is stopped | Its ongoing notification disappears. | No Android process remains to coordinate new cross-node work. |

## Historical owner verification

1. Build with `rtk gradle :apps:android-host:assembleDebug --no-daemon`.
2. Install the debug APK with `adb install -r apps/android-host/build/outputs/apk/debug/android-host-debug.apk`. The reference Xiaomi initially required manual sideloading, but direct replacement installation was subsequently enabled and verified.
3. Open **Lumen**, tap **Create your Space**, then confirm the screen changes to **Starting your Host** and Android shows a persistent **Lumen Host** notification.
4. Force-stop or reboot the phone, reopen Lumen, tap **Start Host**, and confirm the notification returns. Record the outcome; this is a smoke check only, not the five-run Block 2 exit.

## Validation evidence

- `:apps:android-host:testDebugUnitTest` covers the owner-visible empty and ready setup states.
- `:apps:android-host:assembleDebug` produces the sideloadable APK.
- `mise run phase1-check` remains the portable policy and recovery regression gate.
- 2026-09-07 Xiaomi M2101K7BI (Android 13 / API 33): `adb install -r` updated the app without clearing its existing Space. After the owner tapped **Start Host**, accessibility output showed **Host is ready** and **Stop Host**; Android service state confirmed `LumenHostService` is foreground with notification ID `1001`.

## Work moved to later blocks

These items begin only after the headless Host/Hermes exit gate passes:

1. **Secure pairing:** create node identities, show QR/SAS owner confirmation, and atomically bind each paired node’s public-key fingerprint to its Space membership.
2. **Local encrypted transport:** handle API 37 local-network permission, advertise with mDNS as a non-trust hint, establish pinned TLS 1.3 mutual authentication, and process signed/versioned envelopes with replay protection and durable duplicate outcomes.
3. **Companion dashboard:** show paired nodes, Host/network health, tasks, and approvals alongside the companion connection state.
4. **Hardware companion:** add local text-to-speech status, then default-denied foreground microphone and camera-preview capabilities with explicit Android permission prompts. Raw media stays local and is not captured, stored, or shared by default.
5. **Physical exit checks:** verify restart, Wi-Fi loss/reconnect, explicit stop/restart, update retention, a paired-node command, and microphone/camera deny and grant paths. Record five lifecycle runs without manual state repair.
