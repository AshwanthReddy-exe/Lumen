# Repository Guidelines

## Product language

Lumen’s core is a multi-device **Space**, not a particular phone or computer. V1 has one active **Host** that owns canonical shared state and routes cross-node work. A **node** is any paired device. A **capability** is a separately permissioned action. A **companion** is an optional interface such as the old-phone desk display or Mac pet.

Never silently assume a platform, Host location, feature scope, data-sharing level, or permission behavior. State the ambiguity and ask when it changes product direction, privacy, or security.

## Project structure

`README.md` is the entry point. `PRD.md` owns user behavior and acceptance; `ARCHITECTURE.md` owns system boundaries; `DECISIONS.md` records accepted and open choices; `PLAN.md` owns delivery phases, tests, and manual sign-off; `CHANGELOG.md` records notable completed changes.

Create the target directories in `README.md` only after the stack decision. Keep Space semantics and protocol types independent from UI, platforms, Hermes, transport, and storage. Do not commit generated output, runtime state, credentials, personal identifiers, prompts, context records, or task artifacts.

## Working approach

Before implementation, state assumptions, alternatives, and observable success. Choose the smallest vertical slice that proves the requirement. Do not add speculative abstractions, unrelated cleanup, or features beyond the request. Match existing style and remove only orphans created by your change.

For multi-step work:

1. Identify the governing requirement or decision.
2. Define the validation check.
3. Make the minimum change.
4. Run the check and report any gap.

Reproduce bugs with a test when practical. Every changed line must trace to a request, requirement, decision, or test.

## Agents and model selection

Use multiple agents when a phase contains two or more independent, bounded workstreams. Keep one coordinator responsible for contracts, task assignment, integration, and final verification; use at most three workers as defined in `PLAN.md`. Give each worker exact files, dependencies, success criteria, and prohibited scope. Avoid parallel edits to shared schemas, state machines, migrations, or the same files. Do not delegate trivial work when coordination would cost more than execution.

Choose the lightest capable model for each assignment. Use fast, low-cost models for repository discovery, formatting, documentation consistency, fixture generation, and repetitive isolated changes. Use stronger reasoning models for architecture, product ambiguity, cryptography, authorization, concurrency, migrations, threat modeling, difficult debugging, and integration review. Increase model strength when a lightweight attempt fails or uncertainty remains; never trade correctness or security for cost.

The coordinator must inspect every agent result, run integrated checks, resolve contradictions, and remain accountable for the final change. A security-sensitive implementation cannot be approved solely by the agent that authored it.

## Security invariants

Default deny. Local execution may bypass the Host data path but never bypass local capability policy. Cross-node execution requires Host authorization. Adapters cannot expand grants. Approvals are one-time and action-bound. Treat model, tool, MCP, relay, node, and external content as untrusted. Hermes is an adapter, never the Space authority.

## Commands and documentation

No build or test tooling exists; do not invent commands. Use concise Markdown, sentence-case headings, and stable IDs. Validate links and cross-document references.

Update affected documents in the same change: behavior in `PRD.md`, system design in `ARCHITECTURE.md`, durable choices in `DECISIONS.md`, sequence and verification in `PLAN.md`, and completed work in `CHANGELOG.md`. Link to canonical explanations instead of copying them.

## Git and pull requests

Keep `main` releasable. After the bootstrap commit, work on short-lived branches named `feat/<topic>`, `fix/<topic>`, `docs/<topic>`, or `chore/<topic>`. Merge through reviewed pull requests after required checks pass; do not force-push or rewrite shared history on `main`.

Use Conventional Commits: `type(scope): imperative summary`. Prefer `feat`, `fix`, `docs`, `test`, `refactor`, `build`, `ci`, `chore`, `perf`, and `revert`; for example, `feat(protocol): define capability manifest`. Mark breaking changes with `!` and a `BREAKING CHANGE:` footer. Keep commits focused and independently understandable. Before committing, inspect the staged diff and run every available relevant check.

Pull requests must explain the problem and approach, link requirements or decisions, list validation evidence and residual risk, and include screenshots for visible UI changes. Require an independent review for protocol, cryptography, authorization, persistence, migration, or sandbox changes. Use annotated semantic-version tags for releases and maintain release notes in `CHANGELOG.md`.
