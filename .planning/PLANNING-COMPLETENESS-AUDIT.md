# Planning completeness audit — 2026-10-03

## Verdict

**The roadmap is complete in coverage, with explicit branch and evidence gates; unconditional execution handoff remains open.** There are 17 phases and 58 GSD plans containing 143 implementation/checkpoint tasks for Phases 03–17. All 58 pass GSD syntax/structure checks. Those checks establish document shape, not that a basic executor can implement every task without a qualified platform/provider or owner decision. No plan has passed the invoked external cross-AI convergence loop. None of Phases 03–17 has live product acceptance evidence.

The 58 plans list 347 file entries, representing 296 distinct paths. Of those, 51 exist in the current checkout and 245 are proposed. Proposed paths are ownership targets, not proof that a package compiles, a function contract fits current callers, or a device/provider works. The [generated inventory](FILE-INVENTORY.md) is conditional on the D-056 client branch.

**Requirement trace check:** `python3 scripts/check-plan-coverage.py` compares the PRD IDs, requirements register, roadmap phase ownership, and Phase 03–17 plan frontmatter. It passes for all 58 FRs and 58 phase owners; the five Phase 02 requirements are deliberately outside the Phase 03–17 plan check. This checks assignment only, not implementation or the adequacy of each task's acceptance evidence.

## Handoff gaps by phase band

| Phases | What is already decided | What still requires design or proof before a basic executor can safely finish |
| --- | --- | --- |
| 01–02 | Existing baseline and Host/Hermes plans/evidence | Phase 02 deferred live installation/recovery/runtime boundaries must remain separate from its local completion label. |
| 03–04 | Detailed Space/run/memory contracts, proposed functions, E4/E8 gates, concrete loopback/Tailscale Serve browser profile, frozen E8 accepted-record denominator/threshold | E4 may trigger a whole-authority storage migration; actual proxy/session, Hermes Run lookup and E8 quality still require implementation and named live proof. |
| 05–06 | Signed enrollment and attention authority, disposable D-056 feasibility fixture, exact client branch binding checklist | The chosen phone stack and its product source tree/tests remain UNBOUND until physical Android+iPhone evidence; missing hardware is BLOCKED. Phase 06 reply route depends on pinned Hermes support. |
| 07–10 | Cross-node, voice, integrated local and release intent plus task-level file responsibilities, transaction order, failure matrix and live gates | The selected client path for migration, voice engine/license, Mac OS API behavior, release-specific asset digests, physical acceptance and seven elapsed days of soak still need qualification before a basic executor can claim exit. |
| 11–14 | Provider/action/coding/automation boundaries and candidate files | The 16 Phase 11–14 plans now split into bounded contract/effect/recovery tasks, with `PHASE-11-17-CONTRACTS.md` mapping file/function/transaction and first negative proof. Provider API, native-vs-runtime action adapter, coding apply atomicity and automation runtime behavior still require qualification. |
| 15–17 | Commercial, managed and shared-Space decision gates | The 12 Phase 15–17 plans now split into bounded proposal, implementation and recovery/proof tasks. Terms, support matrix, account identity, cloud/region/key custody, public SDK publication and shared role rules still require owner decisions and clean-environment proof; some paths after those gates remain provisional. |

## Definition of a handoff-ready plan

For each GSD task, require all of the following **before** marking it ready for routine implementation:

1. **Input:** governing FR/decision, exact current source file/caller, starting revision, prior task output and selected branch. Distinguish existing files from new files.
2. **Owned edit:** one bounded behavior across real boundaries; exact package/file and proposed function signature or stable existing API; schema fields and migration if durable; route/fixture if public. Reuse existing code/stdlib/native capability before adding an abstraction or dependency.
3. **State and failure:** authority/validation point, legal transitions, transaction order, idempotency/receipt, timeout/retry/cancel/unknown outcome, privacy classification and denial behavior. Give one concrete negative case and one crash/reconnect case for stateful effects.
4. **Proof:** a check command that will exist after that task creates its build scripts; exact assertion and failure predicate; physical/provider/manual gate with device/version/denominator and `PASS/FAIL/BLOCKED/UNSUPPORTED` evidence row. A unit test cannot certify a live journey.
5. **Coordination:** no shared schema or authority file edited concurrently; declared dependency on every prior contract; independent security review for protocol, auth, persistence, migration and sandbox changes; explicit owner checkpoint for product/commercial decisions.

“Every possible case” cannot be exhaustively predicted. The plan should cover **classes** of failures and enforce a phase-entry review for new evidence. Unknown provider behavior, platform limits and future APIs must remain qualification gates, not invented certainty.

## GSD + Ponytail sequence

At each phase entry: `gsd-spec-phase`/`gsd-discuss-phase` only for unresolved decisions → inspect current source and reuse opportunities → `gsd-plan-phase` to split broad tasks → `$gsd-plan-review-convergence` with a genuinely external reviewer → implement dependent plans via `gsd-execute-phase` → `gsd-code-review`, `gsd-secure-phase` where applicable, `gsd-verify-work`, `gsd-audit-uat`. Stop at a failed gate. Ponytail requires omitting speculative modules and keeping the minimal source change that proves each vertical slice; it does **not** waive any security, recovery, accessibility or live acceptance requirement.

The current plan set is a valuable architecture and dependency map. It is **not yet** a fully executable instruction manual for a basic model, nor evidence that one-command installation, talk interaction or cross-node execution works. Update this verdict only after the handoff criteria and external review pass for each phase.
