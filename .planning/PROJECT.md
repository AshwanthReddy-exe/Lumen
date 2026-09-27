# Lumen

## What This Is

Lumen is a private, multi-device AI Space for one owner in its first personal alpha. An always-on Go Host owns canonical conversations, memory, permissions and task truth; Hermes supplies bounded intelligence, while paired nodes and optional companions provide interaction and locally permitted actions.

## Core Value

One conversation continues from laptop to phone after the laptop closes, then an approved phone request produces a verifiable action receipt from the awake, connected laptop in that same conversation.

## Business Context

- **Customer:** The first owner uses a personal alpha; broader self-hosted and managed users follow.
- **Revenue model:** Proprietary paid self-hosted and managed products after personal usefulness is demonstrated.
- **Success metric:** The exact laptop → phone with laptop off → approved phone-to-laptop action → receipt journey in one conversation, with Host reachable and laptop node awake for execution.
- **Strategy notes:** [Canonical release sequence](../docs/PLAN.md#release-sequence).

## Requirements

### Validated

None at product-journey level. Merged foundation evidence is recorded in [the baseline ledger](research/BASELINE-EVIDENCE.md), not treated as a completed alpha capability.

### Active

- [ ] Personal-alpha journey and recovery: Phases 01–10; see [requirement traceability](REQUIREMENTS.md).
- [ ] Expanded beta cohorts: Phases 11–14.
- [ ] Paid self-hosted and managed delivery, then public integrations: Phases 15–17.

### Out of Scope

- Multiple active Hosts, automatic failover and multi-user Spaces in the personal alpha — [the PRD](../docs/PRD.md#non-goals-for-the-first-launch) defers them.
- Messaging, Apple abilities, browser actions, coding and commercial infrastructure as alpha prerequisites — they follow the alpha gate in [the plan](../docs/PLAN.md#release-sequence).

## Context

- Merged `main` at `ebe42cd` has the Go Space authority, encrypted snapshot store, operator boundary, Hermes Runs adapter and setup foundations. Current Phase 01 branch documents and cleanup are uncommitted.
- Draft PR #20 on `feat/lumen-continuity` is candidate branch-only conversation/memory work with open live gates; its separate worktree has dirty edits. It is not merged or a completion claim.
- [docs/PRD.md](../docs/PRD.md) owns product behavior; [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md) owns boundaries; [docs/PLAN.md](../docs/PLAN.md) owns exact 17-phase sequencing and gates.
- The [ingest synthesis](intel/SYNTHESIS.md) contains 58 stable FR requirements. Its `REQ-fr-*` labels are ingest wrappers; the published `FR-*` IDs remain primary here.

## Constraints

- **Authority:** One active Host owns canonical state, grants, approvals, routing, task truth and audit; a node revalidates its exact invocation locally.
- **Runtime:** Hermes stays behind a versioned adapter and cannot grant authority or persist canonical Space state.
- **Security:** Default deny; one-time action-bound approval; minimum permitted context; encrypted sensitive state; honest uncertain outcomes.
- **Release:** Phase 01 is current. Phase 02 waits for the owner-review checkpoint in [the plan](../docs/PLAN.md).
- **Evidence:** Physical device, recovery and seven-day soak gates cannot be claimed from automated tests or draft PR code.

## Key Decisions

| Decision | Rationale | Outcome |
|---|---|---|
| One personal Space with one active Go Host | Keeps canonical authority in one place | Accepted: D-001, D-002, D-034, D-058 |
| Pinned Hermes behind Host-owned contracts | Reuse intelligence while preserving authority | Accepted: D-006, D-026, D-049–D-051 |
| Explicit memory acceptance in alpha | Owner controls retained facts | Accepted: D-059; D-046 is superseded |
| Physical mobile qualification with fallback | Shared client feasibility needs device evidence | Accepted: D-056 |
| Tailscale-assisted alpha reachability | Prove private reachability before relay investment | Accepted: D-057 |

All accepted entries in [docs/DECISIONS.md](../docs/DECISIONS.md) are locked. Its superseded entries are historical.

---
*Last updated: 2026-09-26 after document ingest and Phase 01 baseline review*
