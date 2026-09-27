# Phase 01: Baseline and planning authority - Research

**Researched:** 2026-09-26
**Domain:** Repository evidence, requirement traceability, and documentation reconciliation
**Confidence:** HIGH for recorded repository facts; MEDIUM for the proposed manual validation procedure

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Use the separate `docs/gsd-phase01` branch from `origin/main`; preserve draft PR #20 and its dirty worktree unchanged.
- Phase 01 is documentation, evidence, and planning only. Do not implement product features, merge PR #20, or commit planning files before owner review.
- `docs/PRD.md` owns behavior, `docs/ARCHITECTURE.md` owns boundaries, `docs/DECISIONS.md` owns accepted choices, and `docs/PLAN.md` owns the full delivery sequence. GSD owns phase execution state and plans.
- Classify the seven approved source documents through GSD ingest. Preserve the existing `.planning/codebase/` map.
- Retain all `FR-*` identifiers and the exact 17-phase sequence in the approved master plan. Phase 02 remains blocked on owner review.
- Treat merged `main`, draft PR #20, recorded live evidence, and unverified claims as distinct evidence classes.

### the agent's Discretion
None specified in `01-CONTEXT.md`.

### Deferred Ideas (OUT OF SCOPE)
- Feature code, database migration, Hermes runtime changes, mobile/node implementation, and Phase 02 execution.
- Rewriting or cleaning PR #20's worktree.
- Committing or opening a pull request before the owner reviews the artifact set.
</user_constraints>

## Summary

Phase 01 is a documentation and evidence gate, not a product implementation phase. The acceptance contract has two observable parts: every published FR has exactly one owner and phase, and repository navigation/status claims distinguish merged behavior from branch-only or historical evidence. Phase 01 itself has no FR requirement IDs. [VERIFIED: `.planning/ROADMAP.md:29-35`; `docs/PLAN.md:396-400`]

Use the existing source-of-truth split: PRD for behavior, architecture for boundaries, decisions for accepted choices, master plan for sequencing, and GSD for phase state/plans. Keep the evidence ledger as a dated snapshot and preserve the explicit distinction between merged `main`, draft PR #20, recorded live runs, and unverified claims. [VERIFIED: `.planning/phases/01-baseline-and-planning/01-CONTEXT.md:5-10,24-28`; `.planning/research/BASELINE-EVIDENCE.md:7-7`]

**Primary recommendation:** Treat Phase 01 completion as a traceability and navigation audit with `git diff --check`, a full FR-to-owner/phase comparison, active-link review, and explicit owner sign-off; do not treat baseline test results as evidence for PR #20 or live/device gates. [VERIFIED: `docs/PLAN.md:398-400`; `.planning/research/BASELINE-EVIDENCE.md:22-48`]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|---|---|---|---|
| User behavior and acceptance | Canonical product documentation | GSD requirement map | PRD owns behavior; GSD retains traceability to the primary FR identifiers. [VERIFIED: `01-CONTEXT.md:7-9`; `REQUIREMENTS.md:3-5`] |
| Evidence classification | Evidence ledger | Canonical docs and source tree | The ledger separates merged implementation, automated checks, PR work, and historical live evidence. [VERIFIED: `BASELINE-EVIDENCE.md:7-20,22-41`] |
| Phase order and owner gate | `docs/PLAN.md` | GSD roadmap/state | The master plan owns sequencing; Phase 02 is gated on Phase 01 owner review. [VERIFIED: `docs/PLAN.md:381-400`; `ROADMAP.md:38-46`] |

## Standard Stack

### Core

| Tool/source | Version | Purpose | Why standard |
|---|---|---|---|
| Canonical Markdown docs and `.planning/` artifacts | — | Reconcile source-of-truth docs, evidence, requirements and phase status | This is a documentation-only phase; no application package is needed. [VERIFIED: `01-CONTEXT.md:5-16`] |
| `rtk` command wrapper | Installed locally; version not recorded | Run repository inspection and existing checks through the configured command wrapper | Local RTK instructions require the wrapper; repository guidance defines the baseline task. [VERIFIED: local RTK instructions; `AGENTS.md:55-58`] |
| `mise run phase0-check` | Repo-pinned tool versions in `mise.toml` | Existing Kotlin/Swift contract baseline, only when code/baseline freshness is in scope | Existing configured task; it is not a documentation-link or requirement validator. [VERIFIED: `mise.toml:1-12`] |

### Alternatives Considered

| Instead of | Could use | Tradeoff |
|---|---|---|
| Existing Markdown and manual cross-reference audit | Add a markdown linter/link-check dependency | No existing documentation checker is configured in the inspected task file; adding a package for this phase would exceed its scope. [ASSUMED] |

**Installation:** None.

## Architecture Patterns

### Recommended project structure

Keep Phase 01 evidence and execution material in `.planning/`; edit canonical product and sequencing statements only in their existing owners under `docs/`. The phase's CONTEXT explicitly requires preserving `.planning/codebase/`. [VERIFIED: `01-CONTEXT.md:7-16`]

### Pattern 1: Claim-to-evidence ledger

**What:** For every implementation/status statement, identify the evidence class, source path or captured run, and remaining proof. A passing automated check may establish only the code/revision it ran against; a historical deployment record is not current acceptance evidence. [VERIFIED: `BASELINE-EVIDENCE.md:11-20,22-41`]

**When to use:** Updating current implementation statements, ingest summaries, or phase completion claims.

**Example:**

```text
claim -> evidence class -> source/revision -> check result -> remaining proof
```

### Pattern 2: Stable requirement traceability

Keep each published `FR-*` identifier intact and map it to exactly one owner and phase. The ingest `REQ-fr-*` strings are wrapper labels; published FR IDs remain primary. Phase 01 has no FR IDs of its own. [VERIFIED: `SYNTHESIS.md:5`; `PROJECT.md:39-40`; `ROADMAP.md:32-35`]

### Anti-patterns to avoid

- Promoting PR #20 draft code or its dirty worktree to merged/current behavior. [VERIFIED: `BASELINE-EVIDENCE.md:33-37`]
- Treating historical Azure/macOS/VPS runs as proof of device, cross-machine, recovery, or user-journey acceptance beyond their recorded scope. [VERIFIED: `BASELINE-EVIDENCE.md:39-41`]
- Restoring removed Superpowers artifacts to repair navigation; Phase 01 requires active links to point to GSD/canonical docs. [VERIFIED: `ROADMAP.md:34-35`; `docs/PLAN.md:391-400`]
- Reporting absent row-level PRD acceptance criteria as if they were defined; phase criteria and master-plan gates provide checks without implying implementation. [VERIFIED: `SYNTHESIS.md:5`; `REQUIREMENTS.md:70-72`]

## Don't Hand-Roll

| Problem | Don't build | Use instead | Why |
|---|---|---|---|
| Product requirements and decisions | A second conflicting FR/decision source | Canonical PRD/DECISIONS plus GSD traceability | The locked source ownership is explicit. [VERIFIED: `01-CONTEXT.md:7-9`] |
| Completion evidence | A new aggregate “pass” that conflates test, draft, and live evidence | Existing dated baseline ledger with limitations | The ledger's purpose is to keep these evidence classes distinct. [VERIFIED: `BASELINE-EVIDENCE.md:7-8,22-41`] |

**Key insight:** The main failure mode is evidentiary overclaim, not missing software; retain source, revision/date, result, and remaining proof at every transition from evidence to status. [VERIFIED: `BASELINE-EVIDENCE.md:7,31,35-41`]

## Project Constraints (from AGENTS.md)

- Preserve Lumen terminology and do not assume platform, Host location, feature scope, data sharing, or permission behavior. [VERIFIED: `AGENTS.md:5-9`]
- Keep canonical docs in their assigned owners; keep Space/protocol semantics independent of UI, platforms, Hermes, transport, and storage. [VERIFIED: `AGENTS.md:11-15`]
- Make the minimum change; every changed line must trace to request, requirement, decision, or test. [VERIFIED: `AGENTS.md:17-27`]
- Apply production principles: explicit ownership/trust boundaries, default deny, honest failure outcomes, data minimization, and evidence-backed acceptance. [VERIFIED: `AGENTS.md:29-53`]
- Use `mise run phase0-check` as the current cross-platform contract baseline; do not invent build/test commands for this documentation phase. [VERIFIED: `AGENTS.md:57-61`]
- Keep changes on the authorized short-lived docs branch, use Conventional Commits if later authorized to commit, and retain owner review gates. [VERIFIED: `AGENTS.md:63-81`; `01-CONTEXT.md:5-6`]

## Common Pitfalls

### Confusing baseline checks with phase acceptance

**What goes wrong:** A passing Go or cross-platform run is described as proof of PR #20, live deployment, or the user journey.

**How to avoid:** Cite the ledger's exact revision and scope; retain every “remaining proof” item. The recorded Go and `phase0-check` passes are for the merged baseline only. [VERIFIED: `BASELINE-EVIDENCE.md:22-31`]

### Broken navigation hidden by surviving files

**What goes wrong:** A page exists but active links still target removed planning documents.

**How to avoid:** Search active links in the named canonical docs and Hermes research, then resolve each target relative to its source. Do not treat a string search alone as link validation. [VERIFIED: `BASELINE-EVIDENCE.md:43-47`; `docs/PLAN.md:391-400`]

### Stale snapshot presented as live status

**What goes wrong:** PR state, test results, or branch/worktree status changes after the ledger's observation date.

**How to avoid:** Keep the `Observed` date visible and refresh volatile evidence before using it for an owner acceptance decision. PR #20's recorded facts are explicitly time-bound to 2026-09-26. [VERIFIED: `BASELINE-EVIDENCE.md:3,35-37`]

## Code Examples

There is no application code to add. Minimal repository checks:

```bash
rtk git diff --check
rtk rg -n 'superpowers|docs/superpowers' docs/PRD.md docs/ARCHITECTURE.md docs/DECISIONS.md docs/README.md docs/research/HERMES-LUMEN-INTEGRATION.md
rtk rg -n '^\| FR-' .planning/REQUIREMENTS.md
rtk rg -n '^\| [0-9][0-9]\.' .planning/ROADMAP.md
```

The searches locate candidates; a human must verify every FR has one owner/phase and every active reference resolves. This avoids claiming that grep alone validates Markdown links or semantic traceability. [VERIFIED: `REQUIREMENTS.md:9-68`; `ROADMAP.md:29-35`; `docs/PLAN.md:398-400`]

## State of the Art

Not applicable: this phase adds no runtime framework or package. The active workflow has moved from deleted Superpowers planning references to GSD artifacts and canonical docs. [VERIFIED: `01-CONTEXT.md:7-9,14-16`]

## Assumptions Log

| # | Claim | Section | Risk if wrong |
|---|---|---|---|
| A1 | A manual relative-target/link review is sufficient for this documentation-only phase; a dedicated link checker is not required. | Validation Architecture | A malformed but unreviewed link could survive; owner review should inspect active targets. |

## Open Questions

1. **Does the owner accept the reconciled document set and the evidence snapshot?**
   - What we know: Phase 02 is explicitly blocked on owner review, and planning files must not be committed before that review. [VERIFIED: `ROADMAP.md:38-46`; `01-CONTEXT.md:5-6,22`]
   - What's unclear: Owner acceptance is an external sign-off, not an automated check.
   - Recommendation: Keep Phase 02 blocked until the owner reviews the artifact set.

## Environment Availability

Skipped: documentation/evidence reconciliation uses repository files and existing local commands; it introduces no external service dependency. [VERIFIED: `01-CONTEXT.md:12-22`]

## Validation Architecture

### Test framework

| Property | Value |
|---|---|
| Framework | None for Markdown reconciliation |
| Config file | `mise.toml` provides `phase0-check`; a dedicated doc-validation task was not found in the inspected tasks. [ASSUMED] |
| Quick check | `rtk git diff --check` |
| Existing code baseline | `rtk go test ./...` and `rtk mise run phase0-check` were recorded as passed on the merged baseline; they are not doc acceptance checks. [VERIFIED: `BASELINE-EVIDENCE.md:22-31`] |

### Phase 01 acceptance → validation map

| Criterion | Behavior | Check | Evidence |
|---|---|---|---|
| ROADMAP 01.1 | Every FR has one owner and phase; repo claims distinguish merged and draft code | Enumerate all 58 FR rows; compare against PRD/master-plan assignments; review status claims against evidence ledger | `REQUIREMENTS.md`, `docs/PRD.md`, `docs/PLAN.md`, `BASELINE-EVIDENCE.md` |
| ROADMAP 01.2 | Navigation uses active GSD/canonical docs; removed Superpowers targets are inactive | Search named docs for old references; resolve all Markdown targets; inspect README/index entry points | `docs/PRD.md`, `docs/ARCHITECTURE.md`, `docs/DECISIONS.md`, `docs/README.md`, Hermes research |
| Owner gate | Phase 02 remains blocked until artifact review | Human owner review; do not infer completion from automated checks | `ROADMAP.md:38-46`; `01-CONTEXT.md:5-6,22` |

### Wave 0 gaps

- No dedicated Markdown link/FR traceability checker is configured. Use the searches above plus explicit human target and row review; do not add a dependency for this phase. [ASSUMED]
- Phase 01 plan/verification artifacts were absent when research began; the planner must add the execution plan and its verification artifact. [VERIFIED: `init.phase-op 1` returned `has_plans=false`, `has_verification=false`; phase directory initially contained only `01-CONTEXT.md`]

## Security Domain

Security enforcement is enabled, but Phase 01 changes documentation/evidence only and does not introduce a new runtime attack surface. [VERIFIED: `.planning/config.json:48-50`; `01-CONTEXT.md:12-22`]

| ASVS category | Applies to implementation in this phase | Required handling |
|---|---|---|
| V2 Authentication | No | Do not assert authentication behavior beyond cited implementation/live evidence. |
| V3 Session management | No | Preserve draft/live evidence distinction. |
| V4 Access control | No | Preserve documented default-deny and Host/node authority boundaries; no code changes. |
| V5 Input validation | No | No input-processing changes. |
| V6 Cryptography | No | Do not infer cryptographic acceptance from code existence alone. |

## Sources

### Primary

- `.planning/phases/01-baseline-and-planning/01-CONTEXT.md` — locked scope and source ownership.
- `.planning/research/BASELINE-EVIDENCE.md` — dated implementation, check, PR, live evidence, and open-gate ledger.
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md` — FR traceability and phase status.
- `.planning/intel/SYNTHESIS.md`, `.planning/INGEST-CONFLICTS.md` — ingest counts and conflict disposition.
- `docs/PLAN.md` — master Phase 01 acceptance and owner-review gate.
- `AGENTS.md`, `.planning/config.json`, `mise.toml` — project constraints, validation flags, and configured baseline task.

## Metadata

**Confidence breakdown:**
- Evidence and phase scope: HIGH — directly read the phase sources and ledger.
- Validation method: MEDIUM — repository facts are verified; manual audit sufficiency is a scoped recommendation.

**Research date:** 2026-09-26
**Valid until:** Recheck volatile branch/PR/live-run evidence before owner acceptance.
