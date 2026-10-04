# D-056 client branch contract

Phase 05-01 must record the physical shared-client result before 05-04 or 06-03 starts. The decision is one of **shared client passes**, **Kotlin Android plus responsive phone web**, or **blocked**. Record device models, OS/builds, app revisions, pairing, secure storage, replay, approval and audio observations in `05-EVIDENCE.md`; a simulator or build alone cannot pass D-056. Rewrite the chosen plan's `files_modified`, task `<files>`, and automated command to the selected branch before execution. Keep the other branch unimplemented.

| Contract | Shared client passes | Kotlin Android plus phone web fallback |
|---|---|---|
| Identity and key | `apps/phone/src/pairing.tsx`; secure-storage native module under `apps/phone/` | `apps/android-host/src/main/kotlin/dev/lumen/android/host/Pairing.kt`; Android Keystore-backed local key; responsive `apps/web/src/pairing.tsx` |
| Durable request outbox | `apps/phone/src/outbox.ts` and `apps/phone/src/sync.test.ts` | `apps/android-host/src/main/kotlin/dev/lumen/android/host/Outbox.kt` and `apps/android-host/src/test/kotlin/dev/lumen/android/host/OutboxTest.kt`; phone web uses the Host receipt API |
| Conversation projection | `apps/phone/src/chat.tsx` | `apps/web/src/chat.tsx` responsive view; Android shell opens the authenticated phone web after native pairing |
| Attention | `apps/phone/src/attention.tsx` and `attention.test.tsx` | `apps/web/src/attention.tsx` and `attention.test.tsx` responsive cards; no duplicate Android approval implementation |
| Check | Selected `apps/phone/package.json` check/test scripts plus physical Android and iPhone evidence | Existing Gradle `:apps:android-host:testDebugUnitTest :apps:android-host:assembleDebug`, web `check`/`build` scripts plus physical Android and phone-browser evidence |

## Mechanical plan binding after 05-01

`05-04-PLAN.md`, `06-03-PLAN.md` and the phone-client portion of `10-04-PLAN.md` are branch templates, not executable client handoffs until `05-01-EVIDENCE.md` records a physically measured `PASS_SHARED` or `FAIL_SHARED_SELECT_FALLBACK` and owner sign-off. `BLOCKED` leaves all dependent client work blocked. Phase 05-01 Task 2 edits all three plans before their execution wave; the downstream executor must reject a plan whose declared files or checks still name the other branch.

| Plan/task | `PASS_SHARED` exact file manifest | `FAIL_SHARED_SELECT_FALLBACK` exact file manifest |
|---|---|---|
| 05-04 Task 1 | `apps/phone/src/pairing.tsx`, `apps/phone/src/outbox.ts`, `apps/phone/src/sync.test.ts` | `apps/android-host/src/main/kotlin/dev/lumen/android/host/Pairing.kt`, `apps/android-host/src/main/kotlin/dev/lumen/android/host/Outbox.kt`, `apps/android-host/src/test/kotlin/dev/lumen/android/host/OutboxTest.kt`, `apps/web/src/pairing.tsx` |
| 05-04 Task 2 | `apps/phone/src/chat.tsx`, `apps/phone/src/sync.test.ts` | `apps/web/src/chat.tsx`, `apps/web/src/chat.test.tsx` |
| 06-03 Task 1 | `apps/web/src/attention.tsx`, `apps/web/src/attention.test.tsx` | same web files |
| 06-03 Task 2 | `apps/phone/src/attention.tsx`, `apps/phone/src/attention.test.tsx` | reuse `apps/web/src/attention.tsx`, `apps/web/src/attention.test.tsx`; no Android approval UI |
| 10-04 Host epoch Task 2 client portion | `apps/phone/src/pairing.tsx`, `apps/phone/src/sync.test.ts` | `apps/android-host/src/main/kotlin/dev/lumen/android/host/Pairing.kt`, `apps/android-host/src/test/kotlin/dev/lumen/android/host/PairingTest.kt` |

The shared branch's `apps/phone/package.json` must declare `check`, `test`, and `build` scripts in its scaffold before these tasks start. Its automated task check runs `rtk npm --prefix apps/phone run check && rtk npm --prefix apps/phone run test && rtk npm --prefix apps/phone run build` plus the named Host contract test. The fallback branch runs `rtk ./gradlew :apps:android-host:testDebugUnitTest :apps:android-host:assembleDebug && rtk npm --prefix apps/web run check && rtk npm --prefix apps/web run build` for 05-04 Task 1 and web `check`, `test`, `build` for its chat and attention tasks; `apps/web/package.json` must declare the `test` script before use. For the 10-04 fallback epoch test use the Android Gradle test/build command; for the shared branch use the phone `check/test/build` scripts. The binder updates each downstream plan's `files_modified`, `must_haves.artifacts`, `key_links`, task `<files>`, `<action>` and `<verify>` to exactly the selected manifest and checks, then runs GSD frontmatter/structure validation. Client build/test PASS does not replace the physical gate in `05-01-EVIDENCE.md`.

Both branches use the same versioned Host wire contracts and Host IDs. The client owns only its device key and pending request IDs; Host owns message, run, decision and receipt truth. On uncertain send, query the receipt with the original request ID before retry. A phone web client cannot assert device key possession unless the native shell proves it via the accepted protocol. Do not mark iPhone native support from the Android fallback.
