# Phase 01 context: baseline and planning authority

## Decisions

- Use the separate `docs/gsd-phase01` branch from `origin/main`; preserve draft PR #20 and its dirty worktree unchanged.
- Phase 01 is documentation, evidence, and planning only. Do not implement product features, merge PR #20, or commit planning files before owner review.
- `docs/PRD.md` owns behavior, `docs/ARCHITECTURE.md` owns boundaries, `docs/DECISIONS.md` owns accepted choices, and `docs/PLAN.md` owns the full delivery sequence. GSD owns phase execution state and plans.
- Classify the seven approved source documents through GSD ingest. Preserve the existing `.planning/codebase/` map.
- Retain all `FR-*` identifiers and the exact 17-phase sequence in the approved master plan. Phase 02 remains blocked on owner review.
- Treat merged `main`, draft PR #20, recorded live evidence, and unverified claims as distinct evidence classes.

## Scope

- Establish GSD project files, codebase map, evidence inventory, requirement-to-phase traceability, risk and experiment references, and executable Phase 01 plans.
- Repair active references to the removed Superpowers workflow and stale current-implementation statements.
- Verify automated baseline and document live gates without converting unverified claims to passes.

## Out of scope

- Feature code, database migration, Hermes runtime changes, mobile/node implementation, and Phase 02 execution.
- Rewriting or cleaning PR #20's worktree.
- Committing or opening a pull request before the owner reviews the artifact set.

## Sources of truth

- `docs/PLAN.md` Phase 01 and owner-review gate.
- `.planning/research/BASELINE-EVIDENCE.md` for observed state and test commands.
- `.planning/INGEST-CONFLICTS.md` and `.planning/intel/SYNTHESIS.md` for seven-document ingest findings.
