# Milestone-driven development design

## Status

Approved by the owner on 2026-09-11. This design governs execution of the journey milestones in [PLAN.md](../../PLAN.md). It replaces the stale Block-oriented execution assumptions in earlier implementation plans without invalidating their completed commits or evidence.

## Purpose

Lumen development advances one useful, production-verifiable product journey at a time. A milestone is the unit of authorization, planning, implementation, evidence, and completion. Tasks are internal slices of the active milestone and cannot redefine its scope or exit gate.

## Authority chain

The execution contract is ordered:

1. `docs/PRD.md` owns user-visible requirements and acceptance.
2. `docs/ARCHITECTURE.md` and `docs/DECISIONS.md` own boundaries and durable choices.
3. `docs/PLAN.md` owns the active milestone, sequencing, and exit gate.
4. A milestone specification resolves the active milestone into explicit invariants and observable checks.
5. One implementation plan decomposes that specification into dependency-ordered tasks.
6. One SDD ledger records task state, commits, review rounds, rulings, and residual evidence.

When these disagree, work stops before mutation and the lower-level artifact is corrected. Historical plans never override the current milestone.

## Milestone lifecycle

### Entry

A milestone may start only when `PLAN.md` marks it active, its predecessor exit is recorded, its governing requirements and decisions are known, and its acceptance, negative, recovery, privacy, security, and operational checks are named.

### Execution

Each task is the smallest independently reviewable vertical slice. It declares exact files, consumed and produced interfaces, prohibited scope, a failing test, the minimum implementation, focused verification, and one coherent commit.

Tasks execute in dependency order. One implementation agent may mutate the shared worktree at a time. Parallel agents are limited to disjoint read-only discovery, evidence analysis, or independent review. Shared schemas, authority state, persistence, migrations, and security policy are always serialized.

Every task follows:

`brief → RED → minimal implementation → GREEN → commit → independent review → fix/re-review → ledger completion`

A downstream task cannot begin while an upstream task has dirty fixes, unresolved Critical or Important findings, or an unrecorded completion gate.

### Exit

A milestone completes only when all tasks and reviews are resolved, the full milestone gate passes, owner evidence is distinguished from synthetic evidence, canonical documentation is current, Graphify is updated, and verification-before-completion has fresh command output. Passing unit tests alone never proves a deployment, reboot, privacy, security, or recovery claim.

## Milestone 1 application

Milestone 1 qualifies the combined Linux/VPS reference deployment. Its blocking journey is clean setup, truthful diagnosis, safe rerun and interruption recovery, reboot survival, configured Host-to-Hermes execution, approval, cancellation, event-loss and Host-restart recovery, failed update, rollback, and preserved canonical state.

macOS and Termux retain shared contract, build, and platform-specific checks. Physical evidence gates claims about those profiles; it does not block the conversation nucleus after the combined Linux/VPS reference exits. Hardened or separated deployments require their own pinned mutual-TLS and isolation evidence.

Existing setup commits are inputs, not completion claims. The fixture manifest, synthetic runtime, command discovery, environment markers, or supervisor templates cannot independently establish release readiness.

## Agent policy

The coordinator owns contracts, sequencing, integration, and final verification. Luna agents may implement mechanically complete tasks and perform scoped reviews. Stronger reasoning is required for unresolved authorization, cryptography, concurrency, migration, or final whole-branch security judgments. No worker dispatches its own workers.

## Failure behavior

- Plan drift: freeze implementation, reconcile spec, plan, ledger, commits, and dirty state.
- Hidden architectural scope: return to brainstorming and approval before implementation.
- Missing external evidence: keep the relevant claim open without simulating success.
- Review disagreement: rule against the governing spec and record the cost if wrong.
- Five failed fix rounds: adjudicate every residual finding; never silently mark the task complete.

