---
phase: 02-close-the-existing-host-hermes-foundation
plan: "09"
subsystem: host-hermes-runtime
tags: [hermes, nvidia-nim, chat-only, security]
requires:
  - phase: 02-close-the-existing-host-hermes-foundation
    provides: Real NIM Host result and pinned Hermes profile
provides:
  - Bound local NIM runtime/profile identity and reviewed preserved runtime candidates
  - Chat-only no-tool surface and explicit no-effect action-canary evidence
affects: [phase-02-foundation, phase-07-mac-action]
actuals:
  tasks: 2
  commits: 0
tech-stack:
  added: []
  patterns: [Host owns authority; Hermes tool configuration cannot grant capabilities]
key-files:
  created: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-09-EVIDENCE.md]
  modified: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-09-PLAN.md, .planning/phases/02-close-the-existing-host-hermes-foundation/02-09-EVIDENCE.md, deploy/docker/compose.yaml]
key-decisions:
  - "Keep the owner-selected NVIDIA NIM model as the default and expose no tools in ordinary chat."
  - "Do not claim an attempted tool-call denial event when the configured chat runtime has no tool surface; assign event-level denial to Phase 07 when a Host broker exists."
requirements-completed: []
coverage:
  - id: P02-09-RUNTIME-BOUND
    description: The pinned runtime, provider/model, canonical profile digest, and zero-tool inventory are recorded.
    requirement: FR-38
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-09-EVIDENCE.md; exact image/config identity and authenticated capability/toolset preflight
        status: pass
    human_judgment: true
    rationale: Profile identity is bound to the named runtime and no secret is retained.
  - id: P02-09-CHAT-ONLY
    description: The profile exposes no tool surface and the explicit action canary has no effect.
    requirement: FR-41
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-09-EVIDENCE.md; zero enabled toolsets and absent canary
        status: pass
    human_judgment: true
    rationale: This establishes no transitive authority for chat; it does not claim a model-originated attempted-call denial event.
  - id: P02-09-TOOL-DENIAL-EVENT
    description: Model-originated attempted tool-call denial event.
    requirement: FR-41
    verification:
      - kind: other
        ref: Phase 07 Host capability broker plan; no tool exists in Phase 02 chat profile to produce this event
        status: unsupported
    human_judgment: true
    rationale: Owned by Phase 07 when Lumen introduces a real tool broker.
status: complete
completed: 2026-10-02
---

# Phase 02 Plan 09: Bind the chat-only Hermes profile

The pinned NIM profile identity and empty Hermes API toolset were verified. An explicit terminal-action canary produced no side effect. The preserved conversation/certifier candidates were reviewed and left as later-phase adaptation candidates; no unreviewed code was copied. The log initializer uses no-follow descriptor-relative checks, rejects non-regular and hard-linked logs, and is ordered before Hermes starts.

The chat-only acceptance is limited to zero tool surfaces and no observed effect. Hermes has no configured tool call to deny or corresponding denial event to emit in this profile, so event-level attempted-call handling is assigned to Phase 07. This plan does not claim refusal wording or model-call denial.

Validation: pinned profile/canary evidence is in `02-09-EVIDENCE.md`; `rtk go test ./internal/host ./internal/hermes -count=1` passed; the installed GSD plan-structure validator passed. No credentials or answer content were retained.
