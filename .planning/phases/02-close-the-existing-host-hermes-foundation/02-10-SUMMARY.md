---
phase: 02-close-the-existing-host-hermes-foundation
plan: "10"
subsystem: host-hermes-lifecycle
tags: [hermes, lifecycle, recovery, approval]
requires:
  - phase: 02-close-the-existing-host-hermes-foundation
    provides: Pinned Hermes runtime and Host task boundary
provides:
  - Evidence for durable Run mapping, approval, cancellation, stream loss and restart on pinned Hermes
  - Separate NIM provider-result and truthful-failure evidence
affects: [phase-02-foundation, phases-03-through-09]
actuals:
  tasks: 2
  commits: 0
tech-stack:
  added: []
  patterns: [Provider-independent fault injection on the pinned runtime; provider result kept separately scoped]
key-files:
  created: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-10-EVIDENCE.md]
  modified: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-10-PLAN.md, .planning/phases/02-close-the-existing-host-hermes-foundation/02-10-EVIDENCE.md]
key-decisions:
  - "Exercise lifecycle/failure semantics against pinned Hermes with the deterministic synthetic provider."
  - "Do not claim NIM-specific restart, cancellation, stream-loss, or approval behavior from synthetic evidence."
requirements-completed: []
coverage:
  - id: P02-10-RUN-LIFECYCLE
    description: One durable mapping reconciles after restart and stream loss without redispatch.
    requirement: FR-38
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-10-EVIDENCE.md; current-tree pinned Hermes synthetic-provider lifecycle journey
        status: pass
    human_judgment: true
    rationale: The live journey records the same Run mapping, one create, and a truthful terminal outcome.
  - id: P02-10-APPROVAL-CANCEL
    description: Exact approval/denial and cancellation semantics hold on the pinned runtime.
    requirement: FR-41
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-10-EVIDENCE.md; action-bound approval, denial, in-flight stop, and terminal race checks
        status: pass
    human_judgment: true
    rationale: All live fault cases use the synthetic provider and remain labeled separately from NIM.
  - id: P02-10-NIM-RESULT
    description: NIM produced one durable Host-mediated result and provider failure is persisted honestly.
    requirement: FR-69
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-08-EVIDENCE.md; one completed NIM result plus failed AgentRouter receipt kept historical
        status: pass
    human_judgment: true
    rationale: Response quality was not assessed; NIM-specific lifecycle is not claimed.
status: complete
completed: 2026-10-02
---

# Phase 02 Plan 10: Verify the pinned Hermes lifecycle

The current-tree pinned Hermes synthetic-provider journey and focused Host/Hermes lifecycle contracts passed. Evidence covers durable Run mapping, approval denial, cancellation, held-run stop, gateway restart, Host restart after SSE loss, duplicate/reordered event handling, and first terminal outcome. A separate NIM request completed through the Host with non-empty durable output; its body was discarded and not quality assessed. The earlier AgentRouter failure remains separate historical evidence.

NIM-specific lifecycle behavior is unqualified; it is not a Phase 02 exit claim. The declared exit contract tests lifecycle semantics on the pinned Hermes runtime using deterministic fault injection, with actual provider qualification recorded independently.

Validation: `rtk go test ./internal/host ./internal/hermes -run 'Test.*(Run|Dispatch|Recover|Restart|Stream|Idempot|Approval|Cancel|Terminal|First|Binding)' -count=1` passed 42 tests. `rtk mise run milestone1-macos-check` passed the nine-package suite and native pinned-Hermes journey; cleanup was independently verified. No new provider request was made.
