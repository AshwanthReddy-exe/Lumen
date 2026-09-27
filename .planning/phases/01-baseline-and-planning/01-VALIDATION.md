---
phase: "01"
slug: "baseline-and-planning"
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-26"
---

# Phase 01 — Validation strategy

Phase 01 changes documentation and planning artifacts, not application behavior. Existing Go and cross-platform suites establish the baseline only; they do not prove traceability or owner approval.

## Test infrastructure

| Property | Value |
|---|---|
| Framework | Existing Go test and `mise` tasks; no new documentation test dependency |
| Quick check | `rtk git diff --check` |
| Full baseline | `rtk go test ./...` and `rtk mise run phase0-check` |
| Manual audit | FR ownership/phase comparison, active Markdown target resolution, evidence-class review |

## Per-plan verification

| Plan | Check | Completion evidence |
|---|---|---|
| 01-01 | Inspect map and baseline ledger against the named revision, recorded checks and PR status | Seven `.planning/codebase/` files and dated `BASELINE-EVIDENCE.md` |
| 01-02 | Search and resolve active doc links; inspect source-of-truth ownership and status claims | Canonical docs plus ingest conflict report |
| 01-03 | Count FR rows, check unique owner/phase, check phase order and GSD plan structure | `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`, three PLAN files |

## Manual-only gates

| Behavior | Why manual | Instruction |
|---|---|---|
| Owner accepts Phase 01 artifact set | Product/authority choice cannot be inferred from tests | Review staged diff and evidence; explicitly approve before Phase 02 |
| Live Host/Hermes and device claims | No new live run occurs in this planning phase | Keep them unverified or historical in the ledger; Phase 02/03 run the probes |

## Sign-off

- [x] All three plans have executable checks with a stated failure signal.
- [x] Every active FR appears once with one owner and destination phase.
- [x] Active links resolve and removed Superpowers files are not navigation targets.
- [x] Owner reviewed and approved the uncommitted artifact set on 2026-09-27.
