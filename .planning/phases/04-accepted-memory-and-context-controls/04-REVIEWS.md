---
phase: 04
reviewers: []
review_status: manual-source-review-external-pending
---

# Phase 04 plan review

External GSD review remains pending; this is a source-grounded manual review. The external Claude CLI probe hung and was interrupted, and the Codex CLI is not independent of the current agent.

## 04-01

- **HIGH:** FR-61 requires provenance, classification, retention, scope, inspection, correction, deletion, and export (`.planning/REQUIREMENTS.md:45`). The plan needs an exact memory record schema, classification enum, deletion semantics, and state transition table before implementation. A generic `Memory` file assignment leaves consequential policy choices to the executor.

## 04-02

- **HIGH:** Context projection must enforce per-capability `none`/`metadata`/`summary`/`content` synchronization (FR-33, `.planning/REQUIREMENTS.md:34`) and task-relevant scope (FR-63, `.planning/REQUIREMENTS.md:47`). Specify a deterministic allow/deny matrix, byte budget, precedence when grant/policy/memory scope disagree, and the projection revision bound to each run.

## 04-03

- **HIGH:** The memory UI plan's `apps/web/src/memory.test.tsx` cannot be verified by a Go Host test. Pin the web test/build command and require evidence for correction, deletion, and export from the actual browser route.

## 04-04

- **MEDIUM:** The evaluation task correctly calls for independent queries, but its `internal/conversation/eval_memory_test.go` location risks treating a hand-authored test fixture as retrieval evidence. Specify public or independently sourced dataset, qrels, denominator, threshold, and separate synthetic failure tests; FR-63 needs measured usefulness, not a passing unit test.
- **MEDIUM:** Active-run deletion says already delivered bytes cannot be recalled, but does not define cancellation outcome if Hermes has already accepted a run. Require a state-specific cutoff and evidence for later-run exclusion.

## CYCLE_SUMMARY

Status: **not converged**. Unresolved actionable findings: 3 HIGH, 2 MEDIUM. Replan with exact schema, policy matrix, client checks, and evaluation gate, then rerun independent GSD review.
