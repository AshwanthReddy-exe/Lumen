# Lumen onboarding summary

Observed 2026-09-26 on `docs/gsd-phase01` from `origin/main` at `ebe42cd`.

Lumen is a personal, multi-device Space. The always-on Go Host owns shared authority and state; Hermes is an authenticated reasoning/tool runtime, never the Space authority. Paired devices will supply UI and separately granted local capabilities. The defining laptop-to-phone-to-laptop journey is planned, not delivered on `main`.

## What exists

- The merged Go foundation has Space transitions, grants, approvals, tasks, encrypted whole-state persistence, Hermes Runs integration, setup/supervision, and recovery logic. See [the codebase map](../codebase/ARCHITECTURE.md) and [dated evidence](../research/BASELINE-EVIDENCE.md).
- The merged baseline passed `go test ./...` (463 tests in nine packages) and `mise run phase0-check` on the recorded environment. These do not certify deployment or devices.
- Draft PR #20 is separate branch-only conversation/memory/node work, with a dirty worktree and open certification gates. It was not merged or modified during onboarding.

## What is planned

- [Project](../PROJECT.md) states product scope and authority; [requirements](../REQUIREMENTS.md) map the 58 published `FR-*` IDs; [roadmap](../ROADMAP.md) preserves 17 phases; [state](../STATE.md) tracks progression.
- [The master plan](../../docs/PLAN.md) owns the full delivery sequence. PRD owns behavior, architecture owns boundaries, and the decision record owns accepted choices.
- [Phase 01](../phases/01-baseline-and-planning/01-CONTEXT.md) is evidence and document reconciliation only. Its three plans are awaiting owner review; Phase 02 is blocked until that explicit review.

## Current limits

No canonical conversation, accepted memory, reliable phone client, or useful cross-node action is merged on `main`. Historical live runs are scoped evidence, not proof of the end-to-end alpha. The seven-document ingest found no blocking conflicts; [its report](../INGEST-CONFLICTS.md) records one informational precedence resolution.
