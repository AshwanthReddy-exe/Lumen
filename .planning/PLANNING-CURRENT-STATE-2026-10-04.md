# Planning handoff delta — 2026-10-04

This is an additive status snapshot. It does not replace `docs/PLAN.md`, the 17-phase roadmap, or the 58 phase plans. When this snapshot conflicts with older planning audit text, use this snapshot for repository/PR state and use the owning phase plan plus current code for execution.

## Current authority

- PR #31 merged at `a6d2c658478ed8ea5437e778d9f9633506666fce`; its follow-up ledger/checkpoint correction PR #32 then merged at `5e05eea061527d4665bb3c398b4f09751a52e4a4`. PRs #30–#32 all landed on `main` on 2026-10-04. The local root checkout is clean and tracks `origin/main`; preserved side worktrees are listed separately below.
- PRs #22–#31 are merged; a live `gh pr list --state open` check after the two merges returned no open PRs. PR #20 remains closed unmerged; PR #18 is closed and its later release work landed through #19.
- GSD phases 01–02 remain complete; Phase 03 remains pending. The merge of chat/runtime code is partial implementation evidence, not Phase 03 acceptance.
- Local plan checks on the merged revision: all 73 GSD plans pass `verify plan-structure`; `scripts/check-plan-task-contracts.py` passes all 165 executable tasks; `scripts/check-plan-coverage.py` passes 58 FRs, 58 owners, 508 task source pointers and 459 GSD context pointers; `roadmap.validate`, `validate consistency`, and `git diff --check` pass. These checks establish document consistency, not implementation completeness or external review convergence.
- The five Phase 03 plans were revised on the current planning branch against the post-#29 source audit: E4 now has a reproducible hard gate; 03-02 reuses current merged source; 03-03/04 define Host asset serving/build order, HTTP bounds and browser storage lifecycle; 03-05 pins evidence/reviewer artifacts. All 58 plan schemas, coverage and consistency checks pass after these edits. The local findings are addressed in plan text; independent GSD convergence remains BLOCKED.

## Phase 03 source reconciliation after PR #29

The current plans were written before PR #29. Do not have an executor recreate the following existing source. First compare it to the owning contract and close only a demonstrated gap.

| Plan | Already present on `main` | Remaining handoff / acceptance work |
| --- | --- | --- |
| 03-01 | Space conversation state/reducers in `internal/space/{types.go,model.go,reducer.go,apply.go,state.go}`, tests in `internal/space/conversation_test.go`, shared fixture `protocol/fixtures/space-v1.json`, and durable encrypted Store integration; PR #26 also hardened encrypted backup replacement. | The merged plan now maps to those existing source paths, requires a field/invariant-to-symbol/test/fixture reconciliation first, and directs edits only for proven gaps. Run the specified E4 workload/crash matrix on the named Host and record evidence. Tests alone do not qualify physical power loss. |
| 03-02 | `internal/conversation/{service,projection,recovery}.go`, `internal/host/conversation.go`, certified runtime binding, bounded projection, serialized turns, recovery, receipts, and regression tests. | Audit each invariant in the plan against the merged code and run missing restart/runtime evidence. Do not rebuild these modules or treat synthetic adapters as live qualification. |
| 03-03 | Owner-local command/control socket and Host APIs exist, but no `internal/host/conversation_http.go`, `deploy/tailscale/lumen-web.sh`, or `test/contract/browser_tls_smoke.sh` exists on `main`. | Implement the plan's owner-issued browser session, loopback-only HTTP, private Serve/TLS and CSRF/origin protections. The browser route is not available until the live TLS and denied-public-route evidence passes. |
| 03-04 | No `apps/web/` package exists on `main`. | Select/reuse the repository-supported web toolchain, then build the authenticated client and accessibility/reconnect states. Host Go tests cannot satisfy this task's client test gate. |
| 03-05 | Local synthetic and provider journey scripts exist, but no `test/contract/conversation_journey_test.go` or Phase 03 acceptance evidence artifact exists. | Add a contract journey and record live provider, browser reconnect, separate Host/runtime restart and bounded-stop rows on named versions. Keep provider credentials/output out of evidence. |

Merged implementation entry points include `conversation.NewService`, `(*Service).Send`, `(*Service).Show`, `(*Service).ReconcilePending`, `conversation.ProjectForTask`, `host.Service.ConversationCreate`, `ConversationSend`, and `ConversationShow`. Reuse those stable paths. The 03-03 browser protocol is a new adapter and must not move canonical state out of Host/Space.

The 03-REVIEWS and 03–10 handoff reports remain useful historical findings, but they predate #29. Re-review the current plans and source before using their unresolved counts as current findings. No current cross-AI convergence pass is recorded in this snapshot.

The current official Tailscale CLI reference documents `tailscale serve --bg <loopback-target>` as a backgrounded Serve configuration, HTTPS termination by the local Tailscale daemon, and `tailscale serve status --json` for machine-readable status. The tailnet must have HTTPS certificates enabled; access-control rules still apply. Before 03-03 implementation, capture `tailscale version`, inspect and validate the installed CLI's actual status JSON shape rather than hard-coding an undocumented field, and assert there is exactly the intended private HTTPS route and no Funnel route. See [Serve CLI](https://tailscale.com/docs/reference/tailscale-cli/serve.md), [Serve setup](https://tailscale.com/docs/features/tailscale-serve.md), and [Funnel](https://tailscale.com/docs/reference/tailscale-cli/funnel.md).

## Future OS source freshness

The checked-in reference remains pinned to `98f7f3a3385e12d38ee7fc75bdca2cc3856cf987`. The older analysis document examined `907f38b046b32ed3ac795c07b641e681d8e52101`. A read-only fetch and upstream lookup on 2026-10-04 found upstream `main` at `52328e8009817c5eca66e4461ee4cf55e23fd6c9`. The targeted source delta and Lumen adaptations are in [the upstream delta analysis](research/FUTURE-OS-UPSTREAM-DELTA-2026-10-04.md). The pin is intentionally unchanged; it is not a vendored runtime dependency.

## 2026-10-04 plan handoff corrections

- Phase 03-01 now maps its conversation contract to current `internal/space/reducer.go` and `protocol/fixtures/space-v1.json`; it prohibits a duplicate `conversation.go` reducer and requires a live-source reconciliation before edits. It still owns the E4 named-host gate.
- Phase 05-01 now owns binding Phase 10-04's phone epoch files and client-specific verification to the measured, owner-approved D-056 branch. Phase 10-04 is explicitly non-autonomous and cannot pass while the phone branch is BLOCKED or UNBOUND.
- Phase 10-01 now orders bootstrap trust analysis and an owner checkpoint before installer implementation. Its decision record states that a `curl | sh` bootstrap executes before internal signature checks, so repository transport/control is the initial trust anchor.
- Phase 12-01 now qualifies native versus Hermes browsing behavior before freezing the adapter contract and asking the owner to select; 12-02 is explicitly conditional and must be rewritten if the selected adapter changes.
- Phase 17-01 limits its owner gate to protocol compatibility; shared-space remains a separate gate. Phase 17-02 requires owner approval of exact release metadata before publishing and verifying the Go SDK.
- Detailed findings remain in the phase handoff reviews. PR #31 added task-specific `fails_when` and `result_contract` criteria to all 165 executable tasks; the repeatable validator also checks runnable automated checks and active-plan file ownership. This closes the missing-result-field gap only. Function/caller reconciliation, independent external convergence, and all live acceptance gates remain open; see `.planning/PLAN-REVIEW-STATUS.md`.

## Branch and worktree disposition

At the audit snapshot, 31 of 41 local branches were ancestors of `main`; 10 retained commits not reachable from `main`. Current PR metadata found no open PR requiring merge. “Not an ancestor” does not mean “safe to PR”: several branches are stale forks or implementation superseded by later merged work.

| Local branch/worktree | Current observation | Action |
| --- | --- | --- |
| `feat/lumen-continuity-integration` | PR #29 merged; branch head is its reviewed source lineage. | No duplicate PR. |
| `feat/lumen-continuity-preserved`, `feat/lumen-continuity`, `feat/m2-conversation-nucleus` | 26–27 commits and large diffs relative to current `main`; PR #20 is closed unmerged. `git cherry` reports 24, 25 and 7 patch-distinct commits respectively, but the shared conversation implementation was superseded/integrated by #29. | No duplicate/branch-wide PR. Compare any future bug fix against #29 and extract only missing tested behavior. Preserve histories for now. |
| `feat/phase03-foundations` worktree `/Users/ashwanthreddyboddireddy/.codex/worktrees/90ac/lumen` | 45 commits behind `main`, with 16 modified tracked files and three untracked paths (a new Phase 03 plan directory and two test files) at inspection. Contains mixed Go store/runtime and planning changes. | Preserve the checkout. Do not stage the tree wholesale. Audit each code change against #26/#29 and current plans before proposing a separate PR. |
| `feat/host-execution-contract` | One patch-distinct commit: a Kotlin execution contract checkpoint against an old architecture. | No PR: current source of truth uses the Go Host and versioned cross-language fixtures; assess any still-missing invariant in its owning phase rather than reviving Kotlin architecture. |
| `feat/m1-closure-finish`, `feat/mac-host-scripts`, `feat/phase-1-restart-recovery`, `feat/setup-artifacts`, `fix/mac-host-observability`, `fix/termux-pie-build` | `git cherry` reports all 46 branch commits patch-equivalent to current `main` (37 setup-artifact, six mac-observability, and one each for the other branches). | No PR: changes are already represented in `main`; no open PR exists for these heads. |
| `feat/lumen-continuity-preserved` worktree | Source branch matches the row above; only observed untracked path was Python `__pycache__`. | Preserve; do not create a duplicate PR or delete a user's checkout as cleanup. |
| `fix/store-durable-backup`, `feat/runtime-inventory` worktrees | Their code is represented by merged PRs #26 and #27; clean worktrees. | No duplicate PR. Archive only through managed-worktree lifecycle when no process needs them. |
| detached `070a` worktree | Clean, detached at an old main ancestor. | Retain until workspace ownership/lifecycle is explicitly accounted for. |
| `/private/tmp/lumen-block-execution-plan`, `/private/tmp/lumen-termux-pie` | Git reports both worktree paths prunable because their directories no longer exist. | Git metadata cleanup can be done with `git worktree prune` after confirming no process needs those paths; no source checkout was found there. |

The root checkout being clean does not mean every preserved worktree is clean. Keep user work recoverable; only remove generated or stale state after verifying exact ownership and the saved source.

## Next phase-entry sequence

1. Reconcile 03-01/03-02 plan tasks to existing #29 source: identify acceptance item → source symbol/test → missing check/evidence; patch only missing behavior.
2. Complete 03-03 browser session/API/Tailscale Serve implementation and live TLS/CSRF/origin/public-route qualification.
3. Build 03-04 browser UI against the pinned 03-03 protocol, then add 03-05 journey acceptance and independent current-source review.
4. Run the full phase entry pipeline using GSD: `gsd-discuss-phase` for material open choices, `gsd-pattern-mapper`, `gsd-phase-researcher` when library/platform/provider uncertainty exists, `gsd-plan-phase --reviews` for actionable findings, `gsd-plan-review-convergence` with a detected independent reviewer, then execution → code review → security review → verification/UAT. Do not claim convergence when the reviewer lane is the authoring model or fails to return.
5. Keep Phases 05, 08, 10, and 15–17 gated on their recorded physical, provider, release, and owner decisions. Their provisional file paths are not permission to pre-build unselected platform/business branches.

## Post-merge PR and checkout verification — 2026-10-04

- PR #30 merged into `main` at `228c120cb5a9d4d302ea32def3cafb52e1ba6a0b`. PR #31 was then retargeted to `main` and merged at `a6d2c658478ed8ea5437e778d9f9633506666fce`; GitHub reported `MERGEABLE` before each merge. The merge order preserves the stacked history and avoids merging the parent changes twice.
- PR #32 merged the post-merge audit refresh at `5e05eea061527d4665bb3c398b4f09751a52e4a4`.
- Neither PR had CI checks or an independent review decision recorded at merge time. The owner explicitly requested both merges. This does not satisfy the independent GSD cross-AI convergence gate; plans remain marked unconverged and must not be treated as security-reviewed solely because they are merged.
- A post-merge full GSD sweep found Phase 17-02 had a blocking owner checkpoint while `autonomous: true`; the plan now declares `autonomous: false`. Final local validation passes all 73/73 plan structures, task contracts 165/165, coverage 58/58 requirements and owners with 508 task source pointers and 459 context pointers, both consistency/roadmap checks, and `git diff --check`.
- The local root checkout is clean on `main` and matches `origin/main`. Side worktrees and unrelated retained branches remain preserved; the planning merges did not merge any implementation branch.

## GSD workflow matrix for all 17 phases

The project config has plan research/checking, verification, Nyquist validation, pattern mapping, UI safety, AI integration, code review, security enforcement, and plan-review convergence enabled. Keep `auto_advance` off so owner gates stay visible. Apply only workflows relevant to a phase; the matrix makes that choice explicit for the next executor.

| Phase | Required GSD workflow before/during/after execution |
| --- | --- |
| 01 | Complete. Retain `gsd-ingest-docs`, `gsd-map-codebase`, plan-check and verifier artifacts as baseline. |
| 02 | Complete. Retain review, code/security review, phase verification and evidence labels; deployment gates remain assigned to Phase 10. |
| 03 | `gsd-pattern-mapper` → AI integration spec for model/runtime boundaries → UI spec for browser surface → `gsd-plan-review-convergence` → execute → code review + `gsd-secure-phase` → Nyquist validation → verifier + `gsd-verify-work`/UAT. |
| 04 | AI integration spec + `gsd-eval-planner` for memory usefulness/privacy → UI spec for memory controls → convergence → execute → code/security review → eval review → verifier/UAT. |
| 05 | Resolve D-056 with `gsd-discuss-phase`; platform research before choosing client; UI spec for pairing/chat; convergence; execute only bound paths; independent security review of keys/protocol; physical UAT on named devices. |
| 06 | Qualify Hermes interaction API through research; UI spec for approval/question states; convergence; execute; code/security review of one-time action binding; Nyquist tests for races/replay; verifier/UAT. |
| 07 | `gsd-ai-integration-phase` for tool/capability intent and eval/guardrails; target-platform research; convergence; execute; independent security review for authorization/filesystem/protocol; integration checker and real two-node UAT. |
| 08 | Research licensed speech/wake options; AI integration + eval planner for recognition and response boundaries; UI spec for listening/mute/stop; convergence; execute; security/privacy review; device UAT and measured E6/E7. |
| 09 | Integration checker plans end-to-end seams; AI eval review for the shared response path; UI review for accessibility; execute fault matrix; verifier, UAT and evidence reconciliation. |
| 10 | Release/platform research and explicit support-matrix decision; convergence; execute; security review for install/update/restore; integration checker; verifier, clean-machine UAT, soak review and owner sign-off. |
| 11 | Provider API research and AI integration spec for untrusted inbound content; convergence; execute; security review for account/webhook identity and credential scope; provider live UAT; docs update. |
| 12 | Platform/browser research; AI integration spec for tool policy and prompt injection; UI spec for review/approval surfaces; convergence; security review; target-platform/browser UAT. |
| 13 | Coding-runtime/tool research + AI integration/eval plan; UI spec for diff/application review; convergence; independent security review of sandbox/apply; OS-specific Nyquist tests; verifier/UAT. |
| 14 | AI integration + eval planner for long-run/delegation guardrails; convergence; execute; independent security review of leases, grants, child scope and effects; failure/restart UAT; eval review. |
| 15 | Research and discuss commercial/release decisions; convergence for implementation-bearing plans only after owner-approved proposal; security review; docs update; independent clean-machine UAT before support claims. |
| 16 | AI/data residency research and managed trust decision; AI integration spec only for actual hosted inference boundaries; convergence; independent multi-tenant security review; staging integration/UAT; docs update. |
| 17 | Public API/compatibility research; AI integration review for extension trust where applicable; convergence; independent protocol/shared-data security review; independent SDK consumer check and shared-space UAT; docs update. |

For each phase, run the planner's goal-backward verification; when convergence is enabled, use the external review loop as the required plan gate and re-run the plan checker after each replan. At completion run the phase verifier, Nyquist gap audit, outstanding-UAT audit, and extract learnings; run `gsd-audit-milestone` before archiving. `gsd-ship` creates a focused PR only after validation; keep plan/evidence PRs separate from implementation PRs. Ponytail applies to implementation choices: reuse existing contracts and stdlib/native behavior, add no speculative dependency or abstraction, but retain every requested security, recovery, accessibility and live-proof gate. MemPalace capture and graphify maintenance run only when those integrations are configured/needed; they do not replace checked-in decisions or evidence.
