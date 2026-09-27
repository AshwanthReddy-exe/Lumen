# Document synthesis

- Sources: 7 classified documents — 1 ADR, 3 SPEC, 1 PRD, 2 DOC. Cross-reference graph checked with a depth cap of 50; no cycles found.
- Locked decisions: 51 accepted entries, all from `docs/DECISIONS.md`. Seven superseded entries are historical and were not promoted to active decisions.
- Requirements: 58, preserving the PRD's stable IDs as `REQ-fr-01`–`REQ-fr-15`, `REQ-fr-20`–`REQ-fr-24`, `REQ-fr-30`–`REQ-fr-42`, and `REQ-fr-60`–`REQ-fr-84`. The PRD's requirement table gives no per-row acceptance criteria or scope; those fields are marked absent rather than inferred from launch acceptance.
- Constraints: 19 — 12 protocol, 2 schema, 4 nfr, 1 api-contract.
- Context topics: 5.
- Conflicts: 0 blockers, 0 competing variants, 1 auto-resolved. See [INGEST-CONFLICTS.md](../INGEST-CONFLICTS.md).

The [decisions](decisions.md), [requirements](requirements.md), [constraints](constraints.md), and [context](context.md) files are the per-type intel. `docs/PLAN.md` owns current release sequencing: personal alpha precedes expanded beta and paid self-hosted or managed releases. Its current-implementation ledger distinguishes merged `main` foundations from candidate draft PR #20 code and uncommitted edits in that PR's separate worktree. The historical sequence in `docs/PHASE-2-HOST-HERMES.md` is not an active release roadmap.
