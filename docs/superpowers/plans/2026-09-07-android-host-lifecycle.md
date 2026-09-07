# Android Host lifecycle implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Android companion show the real foreground Host lifecycle and let its owner start or stop it with truthful recovery and error states.

**Architecture:** `HostRuntime` is an Android-process-only observable health model, never a Space authority record. `LumenHostService` transitions it around encrypted `SpaceHost.open`; Compose observes it to render setup, opening, ready, unavailable, and stopped states.

**Tech Stack:** Kotlin, Jetpack Compose Material 3, Android foreground services, Kotlin `StateFlow`, Android Keystore-backed existing state store, JUnit 4.

**Spec:** `docs/superpowers/specs/2026-09-07-android-block-2-design.md`

## Global constraints

- Canonical Space data remains only in the encrypted `SpaceStateStore`; runtime health is volatile evidence.
- Add no microphone, camera, location, overlay, accessibility, local-network, or battery-exemption permissions in this lifecycle slice.
- Start and stop the `dataSync` service only from the companion; add no boot receiver or uptime claim.
- Use Material 3 semantic tokens, native controls, explicit text state, and 48dp minimum touch targets.
- Preserve the existing `adb install -r` update workflow and validate the Xiaomi phone after each completed build.

---

### Task 1: Define the volatile Host runtime state

**Files:**
- Create: `apps/android-host/src/main/kotlin/dev/lumen/android/host/HostRuntime.kt`
- Create: `apps/android-host/src/test/kotlin/dev/lumen/android/host/HostRuntimeTest.kt`

**Interfaces:**
- Produces `enum class HostRuntimeStatus { STOPPED, OPENING, READY, DEGRADED }`.
- Produces `data class HostRuntimeState(val status: HostRuntimeStatus, val detail: String)`.
- Produces `object HostRuntime` with `val state: StateFlow<HostRuntimeState>`, `opening()`, `ready()`, `degraded(detail)`, and `stopped()`.

- [ ] **Step 1: Write the failing test**

```kotlin
@Test fun `runtime exposes the latest explicit health state`() {
    HostRuntime.resetForTest()
    HostRuntime.opening()
    assertEquals(HostRuntimeStatus.OPENING, HostRuntime.state.value.status)
    HostRuntime.degraded("Encrypted state unavailable")
    assertEquals("Encrypted state unavailable", HostRuntime.state.value.detail)
}
```

- [ ] **Step 2: Verify the test fails**

Run: `rtk gradle :apps:android-host:testDebugUnitTest --no-daemon`

Expected: `HostRuntime` is unresolved.

- [ ] **Step 3: Implement the minimal state holder**

```kotlin
object HostRuntime {
    private val mutableState = MutableStateFlow(HostRuntimeState(STOPPED, "Host is stopped"))
    val state: StateFlow<HostRuntimeState> = mutableState.asStateFlow()
    fun opening() { mutableState.value = HostRuntimeState(OPENING, "Opening encrypted Space state") }
    fun ready() { mutableState.value = HostRuntimeState(READY, "Encrypted Space state recovered") }
    fun degraded(detail: String) { mutableState.value = HostRuntimeState(DEGRADED, detail) }
    fun stopped() { mutableState.value = HostRuntimeState(STOPPED, "Host is stopped") }
}
```

- [ ] **Step 4: Run the unit suite**

Run: `rtk gradle :apps:android-host:testDebugUnitTest --no-daemon`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/android-host/src/main/kotlin/dev/lumen/android/host/HostRuntime.kt apps/android-host/src/test/kotlin/dev/lumen/android/host/HostRuntimeTest.kt
git commit -m "feat(android): model Host runtime health"
```

### Task 2: Publish service lifecycle state

**Files:**
- Modify: `apps/android-host/src/main/kotlin/dev/lumen/android/host/LumenHostService.kt`
- Test: `apps/android-host/src/test/kotlin/dev/lumen/android/host/HostRuntimeTest.kt`

**Interfaces:**
- Consumes Task 1 state methods.
- Produces `start(context)`, `stop(context)`, and `ACTION_STOP` handling in `LumenHostService`.

- [ ] **Step 1: Add the failing stopped-state test**

```kotlin
@Test fun `stopped state never reports active coordination`() {
    HostRuntime.resetForTest()
    HostRuntime.stopped()
    assertEquals(HostRuntimeStatus.STOPPED, HostRuntime.state.value.status)
    assertEquals("Host is stopped", HostRuntime.state.value.detail)
}
```

- [ ] **Step 2: Verify failure, then update the service one path at a time**

Run: `rtk gradle :apps:android-host:testDebugUnitTest --no-daemon`

```kotlin
override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
    if (intent?.action == ACTION_STOP) { stopSelf(); return START_NOT_STICKY }
    HostRuntime.opening()
    startForeground(NOTIFICATION_ID, notification())
    when (SpaceHost.open(AndroidEncryptedSpaceStateStore(this), "android-startup")) {
        is SpaceHostOpenResult.Ready -> { HostRuntime.ready(); notifyActive() }
        is SpaceHostOpenResult.Unavailable -> { HostRuntime.degraded("Encrypted Space state is unavailable"); stopSelf(startId) }
    }
    return START_NOT_STICKY
}
```

Add `onDestroy()` to publish stopped unless the current state is degraded. Implement the API-35 timeout callback to publish timeout degradation then call `stopSelf()`.

- [ ] **Step 3: Run Android tests and assemble the APK**

Run: `rtk gradle :apps:android-host:testDebugUnitTest :apps:android-host:assembleDebug --no-daemon`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/android-host/src/main/kotlin/dev/lumen/android/host/LumenHostService.kt apps/android-host/src/test/kotlin/dev/lumen/android/host/HostRuntimeTest.kt
git commit -m "feat(android): report foreground Host lifecycle"
```

### Task 3: Render live lifecycle state in the companion

**Files:**
- Modify: `apps/android-host/src/main/kotlin/dev/lumen/android/host/CompanionScreenModel.kt`
- Modify: `apps/android-host/src/main/kotlin/dev/lumen/android/host/LumenHostActivity.kt`
- Modify: `apps/android-host/src/test/kotlin/dev/lumen/android/host/CompanionScreenModelTest.kt`
- Modify: `apps/android-host/build.gradle.kts`

**Interfaces:**
- Consumes `HostRuntimeState` and `LumenHostService.start`/`stop`.
- Produces `CompanionScreenModel.from(storage: SpaceStateStoreRead, runtime: HostRuntimeState)`.

- [ ] **Step 1: Write failing model tests**

```kotlin
@Test fun `a ready Host offers an explicit stop action`() {
    val screen = CompanionScreenModel.from(presentState, HostRuntimeState(READY, "Encrypted Space state recovered"))
    assertEquals("Stop Host", screen.primaryAction)
}
```

- [ ] **Step 2: Verify failure and implement observation**

Run: `rtk gradle :apps:android-host:testDebugUnitTest --no-daemon`

```kotlin
val runtime by HostRuntime.state.collectAsState()
val screen = CompanionScreenModel.from(store.read(), runtime)
CompanionScreen(screen) {
    if (runtime.status == READY) LumenHostService.stop(this) else startOrCreateHost()
}
```

Add `kotlinx-coroutines-android` explicitly. Preserve one primary action and the existing 48dp height.

- [ ] **Step 3: Run unit tests and assemble**

Run: `rtk gradle :apps:android-host:testDebugUnitTest :apps:android-host:assembleDebug --no-daemon`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/android-host/build.gradle.kts apps/android-host/src/main/kotlin/dev/lumen/android/host/CompanionScreenModel.kt apps/android-host/src/main/kotlin/dev/lumen/android/host/LumenHostActivity.kt apps/android-host/src/test/kotlin/dev/lumen/android/host/CompanionScreenModelTest.kt
git commit -m "feat(android): show live Host health in companion"
```

### Task 4: Physically validate and document the slice

**Files:**
- Modify: `docs/PHASE-2-ANDROID-HOST.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/PLAN.md`
- Modify: `docs/CHANGELOG.md`

- [ ] **Step 1: Build and install without clearing data**

```bash
rtk gradle :apps:android-host:assembleDebug --no-daemon
rtk proxy /Users/ashwanthreddyboddireddy/Library/Android/sdk/platform-tools/adb install -r apps/android-host/build/outputs/apk/debug/android-host-debug.apk
```

Expected: `Success`, with the existing Space still present.

- [ ] **Step 2: Exercise the owner flow**

1. Open Lumen and confirm stored Space.
2. Start Host; confirm opening then ready text and notification.
3. Background/reopen; confirm honest state.
4. Stop Host; confirm stopped text and notification disappears.
5. Start again; confirm ready without recreating Space.

- [ ] **Step 3: Record observed, not assumed, evidence**

Add phone model/API, command results, and pass/fail observations. Do not claim pairing, transport, camera, microphone, boot recovery, or five-run completion.

- [ ] **Step 4: Run final checks and commit**

```bash
rtk gradle :apps:android-host:testDebugUnitTest :apps:android-host:assembleDebug --no-daemon
rtk mise run phase1-check
rtk git diff --check
git add docs/PHASE-2-ANDROID-HOST.md docs/ARCHITECTURE.md docs/PLAN.md docs/CHANGELOG.md
git commit -m "docs(android): record lifecycle verification"
```

## Follow-on plans

- Pairing and transport: portable envelope schema, Keystore identity, explicit pairing session, mDNS, authenticated local channel, replay/idempotency tests, and one fake-node route.
- Companion hardware: local TTS status, foreground-only microphone/camera preview, exact permission explanations, and denial-path checks.

## Self-review

- Lifecycle and health map to Tasks 1–4; pairing/transport and hardware have separate trust boundaries and are deliberately split.
- Each task names concrete files, interfaces, tests, commands, and failure behavior.
- `HostRuntimeState`, `HostRuntimeStatus`, `HostRuntime`, and `CompanionScreenModel.from` use consistent names across tasks.
