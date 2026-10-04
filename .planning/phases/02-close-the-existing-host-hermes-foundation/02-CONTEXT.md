# Phase 02: Close the existing Host/Hermes foundation - Context

**Gathered:** 2026-09-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Close the six Host/Hermes requirements assigned to Phase 02 (FR-01, FR-06, FR-08, FR-09, FR-38, FR-41) with reproducible setup, runtime interaction, recovery, update, external topology, and containment evidence. Keep personal-alpha conversations, memory, node execution, and voice in their assigned later phases.

</domain>

<decisions>
## Implementation Decisions

### Authority and scope
- **D-01:** The approved master plan and GSD roadmap define the current Phase 02 exit. `docs/PHASE-2-HOST-HERMES.md` supplies dated implementation history and profile-specific evidence, not extra global release gates.
- **D-02:** One Host owns Space state, policy and task truth. Hermes runs behind the existing versioned adapter; externally managed Hermes remains independently supervised. Neither a profile digest nor a successful synthetic run proves process containment.
- **D-03:** Run E1 against the pinned runtime, including approval, clarification, stop, lost stream and restart. If a requested interaction lacks a supported API path, record it as unsupported and keep that path disabled until Phase 07 supplies a certified extension.
- **D-04:** Run E2 as negative attempts against file, network, tool and memory boundaries. A successful escape blocks the claimed restricted profile. Preserve exact runtime/configuration and redacted outcomes.

### Delivery and cleanup
- **D-05:** Reuse current setup, update/rollback and recovery mechanisms where they meet the contract. Restrict alpha container updates to the proven path if safe generation replacement cannot be verified in this phase.
- **D-06:** Preserve useful committed and uncommitted PR #20 conversation work on a recoverable branch, verify that preservation, then close draft PR #20. Do not merge it wholesale into this foundation phase. Remove only code or documentation with evidence of obsolescence, and ignore generated Python probe bytecode.
- **D-07:** A live check requires its named environment: a real update/rollback host, pinned Hermes, and two machines for cross-machine Runs. Record missing prerequisites as unverified; a fake endpoint never becomes live Hermes evidence.

### the agent's Discretion
Choose the smallest code changes, probe orchestration and evidence format consistent with the existing repository. Keep all security and recovery checks at real trust boundaries.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/ROADMAP.md` — Phase 02 goal, six requirements and two success criteria.
- `.planning/REQUIREMENTS.md` — exact FR-01, FR-06, FR-08, FR-09, FR-38 and FR-41 text.
- `docs/PLAN.md` — Phase 02 tasks, E1/E2 pass/fail rules and failure behavior.
- `docs/PRD.md` — product requirements and personal-alpha boundary.
- `docs/ARCHITECTURE.md` — Host authority and runtime trust boundary.
- `docs/DECISIONS.md` — accepted and superseded choices.
- `docs/PHASE-2-HOST-HERMES.md` — dated implementation detail and live evidence with profile-specific limits.
- `docs/THREAT_MODEL.md` — explicit security threats and mitigations.
- `.planning/research/BASELINE-EVIDENCE.md` — merged, draft, live and synthetic evidence classification at Phase 01 baseline.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/space/`, `internal/store/`, `internal/host/` and `internal/hermes/` already provide authority, durable state, task lifecycle and the Runs adapter.
- `internal/setup/`, `cmd/lumen/` and `scripts/lumen-*-check` already implement setup, doctor, release management and several local/deployment probes.

### Established Patterns
- Durable create intent precedes runtime I/O; ambiguous outcomes are bounded and never blindly redispatched.
- Tests with fake runtime endpoints establish control flow but must be labeled separately from pinned-runtime and physical-machine evidence.

### Integration Points
- Extend existing scripts and adapters only where a Phase 02 gate proves a concrete gap. Preserve operator and runtime credential separation.

</code_context>

<specifics>
## Specific Ideas

The owner requested a Ponytail review: trace the real flow before editing, prefer deletion and reuse over new abstraction, and audit PR #20 and `.gitignore` for actual bloat without discarding work.

</specifics>

<deferred>
## Deferred Ideas

- Preserved PR #20 conversation, memory and node-status features belong to Phases 04–08 after independent review.
- Android Termux and other platform-profile gates qualify those profiles when claimed; they do not silently expand this Phase 02 exit.

</deferred>

---

*Phase: 02-Close the existing Host/Hermes foundation*
*Context gathered: 2026-09-27*
