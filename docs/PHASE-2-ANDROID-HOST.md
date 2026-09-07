# Phase 2 Android Host slice

## Status

**In progress.** The first physical Android slice is implemented and ready for owner verification on the reference Xiaomi Android 13 phone. It is not the Block 2 exit: node pairing, authenticated local-network transport, boot/locked-store recovery, health signals, and the five-run lifecycle gate remain unimplemented.

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

## Owner verification

1. Build with `rtk gradle :apps:android-host:assembleDebug --no-daemon`.
2. Copy `apps/android-host/build/outputs/apk/debug/android-host-debug.apk` to the phone with `adb push`, then install it from **Files → Downloads**. MIUI on the reference phone blocks USB installation without its own account/SIM configuration, so manual sideloading is the supported validation path.
3. Open **Lumen**, tap **Create your Space**, then confirm the screen changes to **Starting your Host** and Android shows a persistent **Lumen Host** notification.
4. Force-stop or reboot the phone, reopen Lumen, tap **Start Host**, and confirm the notification returns. Record the outcome; this is a smoke check only, not the five-run Block 2 exit.

## Validation evidence

- `:apps:android-host:testDebugUnitTest` covers the owner-visible empty and ready setup states.
- `:apps:android-host:assembleDebug` produces the sideloadable APK.
- `mise run phase1-check` remains the portable policy and recovery regression gate.
- 2026-09-07 Xiaomi M2101K7BI (Android 13 / API 33): `adb install -r` updated the app without clearing its existing Space. After the owner tapped **Start Host**, accessibility output showed **Host is ready** and **Stop Host**; Android service state confirmed `LumenHostService` is foreground with notification ID `1001`.

## Remaining Block 2 work

1. Add device-generated pairing identities and the versioned, mutually authenticated envelope contract.
2. Implement mDNS discovery as a non-trust hint plus encrypted local transport and reconnect behavior.
3. Add durable Host health, storage/key/clock/network degradation, Android-supported restart paths, and corresponding negative tests.
4. Bind the companion to Host health, node, task, and approval state.
5. Complete and record five physical lifecycle runs: restart, Wi-Fi change, power loss/recovery, service stop, and reconnect after a paired-node task.
