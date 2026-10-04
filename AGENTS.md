# Lumen — repository agent instructions

These instructions apply to agents working in this repository. Prefer correctness, minimal changes, verifiable outcomes, and efficient context use. Follow the canonical project documents rather than treating this file as a duplicate specification.

## Product and sources of truth

Lumen's core is a multi-device **Space**, not a phone or computer. V1 has one active **Host** that owns canonical shared state and routes cross-node work. A **node** is a paired device; a **capability** is a separately permissioned action; a **companion** is an optional interface such as the old-phone desk display or Mac pet. Never silently assume platform, Host location, feature scope, data sharing, or permission behavior.

Read only the governing documents relevant to the current task:
- `README.md`: entry point and documented target structure.
- `docs/PRD.md`: behavior and acceptance criteria.
- `docs/ARCHITECTURE.md`: boundaries and contracts.
- `docs/DECISIONS.md`: accepted and open decisions.
- `docs/PLAN.md`: milestones, dependencies, checks, and manual sign-off.
- `docs/DESIGN-PRINCIPLES.md`: mandatory production design principles.
- `docs/CHANGELOG.md`: notable completed changes.

Use the applicable document as authority; report contradictions rather than inventing a resolution. Create target directories from `README.md` only after the stack decision. Keep Space semantics and protocol types independent of UI, platforms, Hermes, transport, and storage.

## 1. Think before coding

Do not hide confusion or silently choose among consequential interpretations. Before a nontrivial change:
1. Identify the governing requirement, current decision, affected boundary, and smallest observable success criterion.
2. Inspect the relevant existing implementation and tests; check `git status` before editing.
3. State only assumptions that matter. If ambiguity changes product direction, privacy, security, a public contract, or irreversible data, stop and ask a focused question. For low-risk reversible details, state the assumption and proceed.
4. When genuine alternatives exist, briefly explain their trade-offs and recommend the smallest option consistent with accepted decisions. Push back on unnecessary complexity.

Do not produce lengthy plans for trivial fixes. For multi-step work, use a short sequence of **change → verification** steps. Respect an existing approved plan; do not re-plan the entire project for each task.

## 2. Simplicity and surgical changes

Build the smallest vertical slice that proves the requirement across real boundaries. Match existing patterns. Do not add speculative infrastructure, premature abstractions, unused configurability, unrequested features, or unrelated refactors. Introduce a shared abstraction only when at least two real callers need the same stable contract.

Change only what the task requires. Preserve unrelated code, comments, APIs, and formatting, including code you do not fully understand. Remove dead code only when made obsolete by this change and its removal is understood. Every changed line should trace to a request, requirement, decision, or test. If the solution is substantially larger than necessary, simplify before finishing.

Never silently weaken a security rule or replace a real boundary with a mock merely to make a check pass.

## 3. Goal-driven implementation and verification

Define observable acceptance before editing. Prefer a failing regression test for reproducible bugs and contract or negative tests for boundary changes. Implement, run the smallest relevant check, inspect failures, and iterate until the stated acceptance criteria pass or a specific blocker is demonstrated.

For each completed task:
- Verify the intended behavior and relevant failure behavior; run broader checks when shared contracts or integration paths change.
- Review the diff for unrelated edits, accidental secrets, and unjustified complexity.
- Report what changed, which checks actually ran, their results, and any unverified behavior or residual risk. Never claim tests passed without running them.
- Update only the affected canonical documents in the same change; link to existing explanations instead of duplicating them.

Do not loop indefinitely on an environmental failure. Identify the blocking dependency, provide the reproduction command and evidence, and stop when further attempts cannot add information.

## 4. Production boundaries and failure behavior

Apply `docs/DESIGN-PRINCIPLES.md` to every design and review. Start from user journeys, explicit contracts, invariants, ownership, trust boundaries, lifecycle states, and failure outcomes—not framework choices or happy-path screens. Domain rules must not depend on UI, platform APIs, model runtimes, transports, databases, or vendors; use narrow, versioned adapters where real boundaries require them.

For every state-changing operation, establish its authority, validation point, idempotency, durable record, retry and timeout policy, cancellation behavior, and truthful outcome when completion is uncertain. Consider process death, duplicate or reordered messages, stale state, clock changes, offline operation, partial persistence, and dependency failure where applicable.

Minimize data at collection and every boundary. Use explicit configuration, reversible migrations where feasible, versioned public formats, compatibility checks, redacted structured events, health signals, and actionable errors. Logs, model responses, and adapter results are never canonical authority records. Do not call a feature production-ready without its relevant acceptance, negative, recovery, and operational evidence on the intended platform.

## 5. Security invariants — non-negotiable

Default deny. Local execution may bypass the Host data path but never local capability policy. Cross-node execution requires Host authorization. Adapters cannot expand grants. Approvals are one-time and action-bound. Treat model, tool, MCP, relay, node, and external content as untrusted. Hermes is an adapter, never the Space authority.

Never commit credentials, personal identifiers, prompts, context records, task artifacts, generated output, or runtime state. Do not leak sensitive content through tests, telemetry, logs, or agent handoffs. Security-sensitive changes require independent review; an authoring agent cannot be their sole approver.

## 6. Context and agent budget

**Default: one agent, one bounded task, no subagents.** A stronger model should not remain active merely to supervise routine implementation. Do not spawn agents for repository browsing, simple fixes, documentation formatting, test generation, or tasks cheaper to execute directly.

Delegate only when there are at least two genuinely independent, bounded workstreams and the expected benefit exceeds coordination and context cost. Never delegate solely because a task feels large. If delegation is justified:
- Keep one accountable coordinator and use the fewest workers necessary; never exceed the limit in `docs/PLAN.md` or three workers, whichever is lower.
- Give each worker a narrow goal, exact file ownership, relevant references, acceptance checks, dependencies, and explicit non-goals.
- Do not allow concurrent edits to the same files or shared schemas, state machines, migrations, or contracts.
- Prefer lightweight capable models for isolated implementation; reserve stronger reasoning for consequential architecture, authorization, cryptography, concurrency, migrations, threat modeling, difficult debugging, and final integration review.
- Inspect every result and run integrated checks. Do not trust a subagent's completion claim without evidence.

Keep context scoped: query the graph or search targeted files before reading large reports; avoid repeatedly reading whole documents or carrying long transcripts into new tasks. Reuse accepted decisions and current milestone status. If a task needs specialist reasoning, formulate one precise question and relevant evidence rather than launching an open-ended review. Never sacrifice correctness or security to save tokens.

## 7. Commands and documentation

Run `mise run phase0-check` for the current cross-platform contract baseline. Use only build, format, lint, unit-test, and contract-test commands that actually exist for the current phase; do not invent commands. Use concise Markdown, sentence-case headings, stable IDs, and validated links and references.

Update `docs/PRD.md` for behavior, `docs/ARCHITECTURE.md` for boundaries or design, `docs/DECISIONS.md` for durable choices, `docs/PLAN.md` for sequencing and verification, and `docs/CHANGELOG.md` for notable completed work—but only when the change affects them. Avoid documentation churn.

## 8. Git and pull requests

Keep `main` releasable. After bootstrap, use short-lived `feat/<topic>`, `fix/<topic>`, `docs/<topic>`, or `chore/<topic>` branches. Before a new branch or PR, fetch `origin/main`; if the previous PR merged, fast-forward local `main` and branch from it. If not, stop and ask the owner to merge or explicitly approve another base. Never force-push or rewrite shared `main` history.

Use focused Conventional Commits: `type(scope): imperative summary` (`feat`, `fix`, `docs`, `test`, `refactor`, `build`, `ci`, `chore`, `perf`, `revert`). Mark breaking changes with `!` and a `BREAKING CHANGE:` footer. Before committing, inspect the staged diff and run relevant available checks.

PRs must describe the problem, approach, linked requirements or decisions, actual validation evidence, and residual risks; include screenshots for visible UI changes. Require independent review for protocol, cryptography, authorization, persistence, migration, or sandbox changes. Use annotated semantic-version release tags and record releases in `docs/CHANGELOG.md`.

## 9. Graphify

This project may contain a knowledge graph at `graphify-out/` with concept nodes, communities, and cross-file relationships.

When the user types `/graphify`, follow the installed Graphify skill or instructions first.

For codebase questions, if `graphify-out/graph.json` exists, first run `graphify query "<question>"`. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return scoped subgraphs; prefer them to the full report or broad source scans. Dirty `graphify-out/` files are expected after hooks and do not justify skipping Graphify. Skip it only if the task concerns stale or incorrect graph output or the user explicitly opts out.

Use `graphify-out/wiki/index.md`, when present, for broad navigation. Read `graphify-out/GRAPH_REPORT.md` only for broad architecture reviews or when scoped queries are insufficient. After modifying code, run `graphify update .` to refresh the graph (AST-only, no API cost). If Graphify is unavailable or fails, report that and continue with targeted source inspection when safe.
