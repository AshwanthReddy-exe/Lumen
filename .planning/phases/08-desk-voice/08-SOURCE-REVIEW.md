# Source-grounded plan review — Phase 08

Status: **OPEN; not cross-AI converged.** Independent CLI review was unavailable after a bounded attempt; this is an inline source review.

## 08-01

- **HIGH:** `08-01-PLAN.md:19` requires a “licensed, device-resident wake engine” but names no selected engine, supported ABI, model artifact, download/provenance, or license decision. The source currently has only the Android host shell (`apps/android-host/src/main/AndroidManifest.xml`, `apps/android-host/src/main/kotlin/dev/lumen/android/host/LumenHostActivity.kt`). Gate implementation on an explicit engine selection and license record, with tap-to-talk as the executable fallback.
- **MEDIUM:** The plan claims pre-activation zero egress (`08-01-PLAN.md:19,23`) but the test ownership lists no network observation hook. Add a physical or process-level egress check that fails on audio bytes before wake; a state test alone cannot prove this.

## 08-02

- **HIGH:** The plan asks for E7 physical thresholds (`08-02-PLAN.md:20`) without listing numeric acceptance values in the task. Copy the approved values into the evidence contract or cite their exact governing lines; a basic executor cannot decide PASS reproducibly.
- **MEDIUM:** A remote speech opt-in is described (`08-02-PLAN.md:19`) but no opt-in UI/state or failure path is file-owned. Either explicitly defer remote speech and prohibit it in this phase, or assign its contract and UI files.

CYCLE_SUMMARY: unresolved HIGH 2; actionable MEDIUM 2; LOW 0. Revise plans, then run independent `gsd-review` again.
