---
phase: "02"
slug: "close-the-existing-host-hermes-foundation"
status: draft
nyquist_compliant: false
wave_0_complete: true
created: "2026-09-27"
---

# Phase 02 — Validation strategy

## Test infrastructure

| Property | Value |
|---|---|
| Framework | Go `testing`; existing contract and deployment probes |
| Config | `go.mod`, `mise.toml` |
| Quick run | `rtk go test ./internal/setup ./internal/host ./internal/hermes ./cmd/lumen ./test/contract -count=1` |
| Full automated suite | `rtk mise run phase2-check` on a machine with the configured Android SDK |
| Feedback time | Measure on the execution machine; no runtime claim from this planning file |

## Sampling

- After a code change, run the smallest affected Go package check.
- Before the Phase 02 gate, run the full automated suite on a supported machine and the live probes below.
- Re-run a live probe only when its runtime, deployment profile or relevant code changes.

## Requirement and threat map

| Requirement | Automated anchor | Live or negative gate | Threat |
|---|---|---|---|
| FR-01 | `internal/host/service_test.go` create-once and restart checks | Same Space, owner and Host identities after setup rerun and process restart | T-011 |
| FR-06, FR-09 | `internal/setup/supervisor_test.go`, `cmd/lumen/doctor_service_test.go` | Foreground/supervisor boot and truthful doctor on the claimed profile | T-011 |
| FR-08 | `internal/setup/journal_test.go`, `cmd/lumen/release_test.go` | Fresh/adopt/interrupted setup plus native update and rollback on a real machine | T-017 |
| FR-38, FR-41 | `internal/hermes/client_test.go`, `internal/host/execution_test.go`, `test/contract/setup_hermes_run_test.go` | E1 pinned Runs/approval/stop/lost stream/restart; E2 file/network/tool/memory denials; two-machine external Runs | T-018, T-030 |

## Wave 0

Existing test infrastructure covers the assigned requirements. Add an assertion only when a probe finds an uncovered failure path.

## Live sign-off

| Gate | Evidence needed |
|---|---|
| Native artifact update/rollback | Machine, before/after digest, preserved identity/state, service health, rollback digest |
| Pinned Hermes E1 | Runtime digest/version, exact requests, events, results, reconciling restart and stream loss |
| Cross-machine external Runs | Two machine identities, mutual TLS endpoint, independent Hermes lifecycle, real run and Host-only restart |
| Anonymous image pull | Unauthenticated client, exact image digest and pull result |
| Restricted profile E2 | File/network/tool/memory attempted escapes, effective OS/container policy and denied outcomes |

Missing hardware, credentials or a supported API path is recorded as unverified or unsupported. It is not a pass.

## Validation sign-off

- [ ] Every execution plan has runnable automated checks with stated failure signals.
- [ ] Full suite passes on a supported machine.
- [ ] Named live gates have reproducible redacted evidence.
- [ ] Phase verifier and independent security review pass.

**Approval:** Pending Phase 02 verification.
