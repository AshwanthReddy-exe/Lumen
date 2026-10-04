---
phase: 02-close-the-existing-host-hermes-foundation
plan: "11"
subsystem: runtime-containment
tags: [hermes, nvidia-nim, containment, security]
requires:
  - phase: 02-close-the-existing-host-hermes-foundation
    provides: Pinned chat-only Hermes runtime profile
provides:
  - Profile-scoped NIM file/secret and endpoint allowlist evidence
  - Separate synthetic E2 denial evidence and explicit later-phase ownership for unsupported classes
affects: [phase-02-foundation, phases-04-and-07]
actuals:
  tasks: 2
  commits: 0
tech-stack:
  added: []
  patterns: [Separate profile claims; tested endpoint positive and controlled-destination negative in one runtime]
key-files:
  created: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-11-EVIDENCE.md]
  modified: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-11-PLAN.md, .planning/phases/02-close-the-existing-host-hermes-foundation/02-11-EVIDENCE.md, scripts/lumen-nim-e2-check, scripts/lumen-nim-egress-proxy.py, deploy/docker/compose.nim-e2.yaml]
key-decisions:
  - "Keep synthetic no-egress E2 separate from the NVIDIA NIM allowlisted profile."
  - "Do not claim a general all-egress guarantee from the tested NIM endpoints and addresses."
  - "Route event-level tool denial to Phase 07 and Lumen-owned memory denial to Phase 04."
requirements-completed: []
coverage:
  - id: P02-11-FILE-SECRET
    description: The NIM-profile Hermes runtime cannot read or write Host-only canary data or configured secret paths.
    requirement: FR-41
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-11-EVIDENCE.md; latest same-runtime file/secret probe
        status: pass
    human_judgment: true
    rationale: Result is scoped to the exact image/config and tested paths.
  - id: P02-11-NETWORK
    description: The NIM profile reaches its approved endpoint and denies the tested controlled destinations and direct routes.
    requirement: FR-38
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-11-EVIDENCE.md; same-runtime NIM positive and controlled-destination negative with direct-route checks
        status: pass
    human_judgment: true
    rationale: This is not an all-egress claim; only the named endpoint and tested destinations are covered.
  - id: P02-11-TOOL-EVENT
    description: Event-level denial of a model-originated tool attempt.
    requirement: FR-41
    verification:
      - kind: other
        ref: Phase 07 tool broker gate; NIM chat profile has no enabled tool surface
        status: unsupported
    human_judgment: true
    rationale: No configured tool exists to generate the runtime start/denial event in this phase.
  - id: P02-11-MEMORY
    description: Lumen-owned memory denial.
    requirement: FR-41
    verification:
      - kind: other
        ref: Phase 04 accepted-memory and context-controls gate; Lumen-owned memory is not implemented
        status: unsupported
    human_judgment: true
    rationale: No memory boundary exists in this phase.
status: complete
completed: 2026-10-02
---

# Phase 02 Plan 11: Verify profile-scoped containment

The NIM profile passed the same-runtime Host-file/secret checks, approved endpoint positive control, controlled-destination negative controls, direct-route checks, and cleanup verification. The synthetic E2 profile separately passed Host-file/secret, network sink, tool/self-grant, and session-isolation checks. No synthetic claim is transferred to NIM, and no all-egress claim is made.

The NIM chat profile had zero enabled API toolsets and the explicit action canary produced no effect; this meets the no-tool-surface acceptance without claiming a model-originated denial event. Event-level tool denial is owned by Phase 07. Lumen memory denial remains unsupported until Phase 04.

Validation: `rtk sh -n scripts/lumen-nim-e2-check`, proxy self-test, the E2 and NIM E2 probes, focused contract tests and cleanup checks passed as recorded in `02-11-EVIDENCE.md`. No credential, prompt, answer, or personal content was retained by the probes.
