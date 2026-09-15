# Lumen–Hermes conversation nucleus implementation plan

> **For Codex:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` to execute this plan task by task.

**Goal:** Make ordinary Lumen chat a Host-owned, personalized, canonical conversation backed by an unmodified Hermes runtime, with no implicit Hermes tools or memory.

**Architecture:** The Go Host owns conversation state, persona, bounded context, runtime policy, certification and terminal outcomes. It calls Hermes through the existing Runs adapter. A separately versioned native Hermes plugin calls a dedicated Host broker through an owner-restricted Unix socket (or pinned mTLS remotely); it never reads the Space store or grants authority. The first deployment proof uses an existing co-located Hermes installation, and the combined-container path reuses the identical contracts.

**Tech stack:** Go standard library and existing project packages; Python using Hermes's supported plugin API; Docker Compose; shell journey scripts; `mise` verification.

**Governing design:** [Lumen–Hermes conversation nucleus design](../specs/2026-09-15-lumen-hermes-conversation-nucleus-design.md)

## Execution constraints

- Do not fork or patch Hermes.
- Keep schema, persistence, migration and authority edits serialized under one implementer.
- Use tests first for every behavior change and capture the failing test before implementation.
- Use the standard library or existing dependencies unless a concrete requirement proves otherwise.
- Never send conversation content to an uncertified runtime.
- Do not describe generic `agent.run/execute` as Lumen chat.
- After each task, run its focused checks, inspect the diff, and commit one independently understandable change.

## Task 1: Add canonical state records and migration

**Files:**

- Modify: `internal/space/types.go`
- Modify: `internal/space/state.go`
- Modify: `internal/store/store.go`
- Test: `internal/space/state_test.go`
- Test: `internal/store/store_test.go`

**Steps:**

1. Write failing tests for canonical conversations, ordered messages, immutable personas, accepted context, runtime profiles, certifications and session mappings.
2. Write a failing v1-to-v2 migration test that preserves every existing identifier and authority record, is idempotent, rejects future schemas, and leaves the old file readable when replacement fails.
3. Add only the concrete versioned records required by the design. Use bounded constants: 64 KiB per chat message, 512 KiB per conversation, 20 projected messages and 32 KiB projected context.
4. Implement pure migration plus verified atomic encrypted replacement using the existing store durability pattern.
5. Run `go test ./internal/space ./internal/store` and commit `feat(space): add canonical conversation state`.

## Task 2: Add atomic conversation transitions

**Files:**

- Modify: `internal/space/commands.go`
- Modify: `internal/space/reducer.go`
- Modify: `internal/space/results.go`
- Test: `internal/space/conversation_test.go`

**Steps:**

1. Write failing tests for create, send intent, verified completion, duplicate request IDs, conflicting reuse, oversize input, sequence ordering and unknown outcome.
2. Add commands that atomically persist the user message, task, idempotency record and runtime intent before dispatch.
3. Derive the assistant message ID from the task ID; append it only on verified completion and never on `unknown_outcome`.
4. Preserve monotonic transitions under duplicated or reordered completion observations.
5. Run `go test ./internal/space` and commit `feat(space): add conversation transitions`.

## Task 3: Add Host-owned chat orchestration

**Files:**

- Add: `internal/conversation/service.go`
- Add: `internal/conversation/projection.go`
- Add: `internal/conversation/persona.go`
- Test: `internal/conversation/service_test.go`
- Modify: `cmd/lumen-host/main.go`
- Test: `cmd/lumen-host/main_test.go`

**Steps:**

1. Write failing tests proving public commands reject caller-supplied instructions, Hermes session IDs, provider/model overrides and profile digests.
2. Implement deterministic bounded projection from canonical messages and accepted `user.preferences/v1` records.
3. Compile an immutable Lumen persona that identifies Lumen as the user's private Space intelligence and forbids claims unsupported by Host records.
4. Orchestrate persist-before-I/O, exact certification, Runs submission, reconciliation and verified assistant append through existing adapters.
5. Add `conversation create`, `conversation send`, `conversation show` and `preference set` without exposing runtime knobs.
6. Run `go test ./internal/conversation ./cmd/lumen-host` and commit `feat(host): add canonical Lumen chat`.

## Task 4: Add the dedicated plugin broker

**Files:**

- Add: `internal/pluginbroker/protocol.go`
- Add: `internal/pluginbroker/server.go`
- Add: `internal/pluginbroker/auth_unix.go`
- Test: `internal/pluginbroker/server_test.go`
- Modify: `cmd/lumen-host/main.go`

**Steps:**

1. Write failing protocol tests for version, plugin identity, Space reference, Host epoch, request ID, nonce, expiry and payload digest.
2. Write negative tests for replay, stale epoch, altered digest, unknown plugin and socket permissions.
3. Implement only the approved narrow operations: capabilities list, conversation turn, context request, action request, task status, task cancel and approval submit.
4. Ensure proposals pass through normal Host policy; expose no raw state mutation, storage, shell, broadcast or target selection.
5. Run `go test ./internal/pluginbroker ./cmd/lumen-host` and commit `feat(host): add Hermes plugin broker`.

## Task 5: Ship the pinned native Hermes plugin and skill

**Files:**

- Add: `integrations/hermes/lumen-plugin/plugin.yaml`
- Add: `integrations/hermes/lumen-plugin/__init__.py`
- Add: `integrations/hermes/lumen-plugin/lumen_client.py`
- Add: `integrations/hermes/lumen-plugin/skills/lumen/SKILL.md`
- Add: `integrations/hermes/tests/test_plugin.py`
- Add: `integrations/hermes/README.md`

**Steps:**

1. Write failing Python tests against a small fake plugin context for exact tool registration and broker request serialization.
2. Implement the plugin only through Hermes's documented plugin interface (`ctx.register_tool`, supported hooks and bundled skills).
3. Add a fail-closed `pre_tool_call` guard: while `lumen.chat.default/v1` is active, every non-Lumen tool is denied.
4. Make the onboarding skill explain immutable source pinning, requested access, pairing and restart, and require explicit owner confirmation before installation or enablement.
5. Prove disabling or uninstalling the plugin cannot touch the Space store.
6. Run the Python tests and commit `feat(hermes): add separate Lumen plugin`.

## Task 6: Certify the zero-tool runtime profile

**Files:**

- Add: `internal/hermes/profile.go`
- Add: `internal/hermes/certification.go`
- Test: `internal/hermes/certification_test.go`
- Modify: `internal/conversation/service.go`
- Test: `internal/conversation/service_test.go`
- Modify: `internal/setup/doctor.go`

**Steps:**

1. Write failing tests for `lumen.chat.default/v1`, including zero tools/memory, one turn and bounded model, token and deadline policy.
2. Bind certificates to endpoint identity, Hermes version, plugin commit, profile/config digests and expiry.
3. Require an exact fresh certificate before any Runs POST; a missing or mismatched certificate must yield durable `runtime_profile_unverified` with zero disclosed conversation bytes.
4. Reject unexpected tool, memory or budget-expansion events, stop the run and invalidate the certificate.
5. Extend doctor with independent store, conversation schema, Runs, plugin and profile readiness.
6. Run `go test ./internal/hermes ./internal/conversation ./internal/setup` and commit `feat(hermes): enforce certified chat profile`.

## Task 7: Add existing-Hermes and combined conformance journeys

**Files:**

- Add: `scripts/lumen-hermes-plugin-install`
- Add: `scripts/lumen-conversation-check`
- Modify: `scripts/lumen-mac-test`
- Modify: `deploy/compose.yaml`
- Modify: `mise.toml`
- Add: `test/contract/conversation_journey_test.go`

**Steps:**

1. Add a failing contract test showing the old generic wrapper cannot satisfy the Lumen conversation journey.
2. Replace the manual script path with Host conversation commands and make unsafe generic mode explicit and separate.
3. Add immutable plugin source/ref verification and separate co-located lifecycle ownership.
4. Run the same acceptance, restart, session-loss, zero-tool and plugin-removal journey against existing co-located Hermes and isolated combined containers.
5. Ensure external/co-located service commands control only Lumen; combined commands control exactly the declared services.
6. Run `go test ./test/contract`, the new `mise` gate and existing Milestone 1 gates; commit `test(conversation): add Hermes topology journeys`.

## Task 8: Record real evidence and reconcile canonical docs

**Files:**

- Modify: `docs/PRD.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/DECISIONS.md`
- Modify: `docs/PLAN.md`
- Modify: `docs/THREAT_MODEL.md`
- Modify: `docs/PHASE-2-HOST-HERMES.md`
- Modify: `docs/CHANGELOG.md`
- Add: `docs/evidence/m2-conversation-nucleus-*.md`

**Steps:**

1. Run the real existing-Hermes journey with a non-sensitive model credential supplied through Hermes's own configuration, never through the repository or prompt.
2. Record versions, commit, topology, artifact/profile digests, expected/actual outcomes and redacted evidence for identity, accepted preference, canonical continuity, restart, session loss and zero-tool behavior.
3. Run combined conformance separately; do not let Docker evidence satisfy real-machine or reboot gates.
4. Reconcile all canonical documents without declaring M1 or M2 complete where evidence remains.
5. Run `graphify update .`, placeholder/link checks, `go test -race ./...`, all relevant `mise` gates and secret scanning.
6. Request independent correctness/security review, resolve all Critical and Important findings, then commit `docs(m2): record conversation nucleus evidence`.

## Final verification

From the worktree root, record fresh output for:

```sh
git status --short --branch
go test -race ./...
mise run phase0-check
mise run milestone1-linux-check
mise run conversation-nucleus-check
graphify update .
```

Do not claim completion if the real-model proof, plugin/profile enforcement, canonical restart continuity or combined conformance is missing. Use `superpowers:verification-before-completion`, then `superpowers:requesting-code-review`, and only then `superpowers:finishing-a-development-branch`.
