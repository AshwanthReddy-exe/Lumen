# Phase 02 independent review — interim audit

Date: 2026-10-02

**Verdict: BLOCKED / review incomplete.** An independent read-only review of the recent active-branch changes found one actionable P2 finding: a macOS LaunchAgent required the immutable setup digest after an explicit executable update, so the updated binary could not restart. The fix now resolves the authorized digest from the committed release record, still validates the record against the immutable binding, and hashes the running executable. A regression test covers rejection of the old digest and acceptance of the committed current generation. The reviewer found no additional actionable finding in the scoped review.

The fix was verified by 283 passing tests across `cmd/lumen`, `cmd/lumen-host`, and `internal/setup`; the complete Go suite passed 500 tests across nine packages; `mise run phase2-check` passed with Android SDK configured.

This is a scoped recent-change review, not the full Phase 02 security review in Plan 02-12. Since the prior review, a current-tree pinned synthetic lifecycle journey passed and its disposable Docker resources were verified absent; a separate Docker Desktop E2 run passed file/network/tool denials with UID `0:0` and its cleanup was later independently verified. The independent review also confirms the previously recorded macOS LaunchAgent CR-02 (login/reboot readiness) and CR-03 (durable activation after publication interruption) remain open. These apply to an installed macOS auto-start profile and are explicitly assigned to Phase 10's named-profile login/reboot and publication-failure recovery gates; they do not certify the Phase 02 manual development Host. A detached Hermes `0.20.6` health endpoint responded, but its service is not loaded, the Lumen Host socket is absent, and it is not the pinned authorized real-provider profile. Real-provider qualification and actual runtime binding remain blocked; memory denial remains unsupported until Lumen memory exists. No Phase 02 PASS verdict is made. Resume a full independent review after Plans 02-08 through 02-11 have acceptable evidence for the declared local profile.

## Independent read-only review — 2026-10-02

An independent reviewer re-audited the active uncommitted source changes and current Phase 02 evidence; it made no edits and ran no provider/deployment probes. It found no basis to approve the implementation from the source diff alone, and agreed that real-provider qualification/runtime binding and the declared E2 profile remain incomplete. It also pointed back to the open historical setup findings in `02-REVIEW.md`:

| Finding | Present scope | Required disposition |
|---|---|---|
| CR-02: launchd GUI-domain/boot evidence does not prove next-login persistence or pre-login availability. | Installed macOS auto-start profile only; the Phase 02 manual Host does not claim boot persistence. | Phase 10 must demonstrate canonical LaunchAgent location, truthful status, and actual login/reboot for each supported profile, or mark it unsupported. |
| CR-03: failure/interruption around final LaunchAgent publication can leave uncertain deferred auto-start. | Persistent macOS setup/publication, not foreground development execution. | Phase 10 must inject failure around publication/sync, define durable activation and recovery, then verify rerun/next-login behavior; do not claim closure from unit tests alone. |

The exact Phase 10 tasks now name these gates and the implementation files in `10-01-PLAN.md`. The reviewer noted that the release-digest helper and Host executable check match the intended update/restart fix, but it did not execute tests; focused/full tests were run separately by the coordinator and are recorded in the evidence ledger. This is not a full passing review: Phase 02 remains executing until real-provider and runtime-profile acceptance is met, and the entire Phase 02 code/security review is rerun against that named profile.


## Independent review of the NIM evidence and current ledger — 2026-10-02

**Verdict: BLOCKED.** The independent reviewer examined the current evidence and found the successful NIM Host result credible for the named profile. The bounded terminal-canary check establishes no observed side effect with empty API toolsets, but does not establish that Hermes emitted an actual tool-call event, that the model refused in natural language, or how such an event would be handled.

The reviewer found these Phase 02 exit items still open:

| Gate | Current evidence | Disposition |
|---|---|---|
| Real-provider answer | One NIM task completed with non-empty durable output; response content discarded. | PASS for task completion on this profile; answer quality is unassessed. |
| Runtime lifecycle on NIM | Restart, stream loss, cancellation, and approval behavior passed only on the synthetic profile. | OPEN; qualify these behaviors against the named NIM profile. |
| Restricted-profile containment | File, network, and tool checks passed with a synthetic provider; NIM check was limited to empty tool inventory and no canary effect. | OPEN for the declared NIM profile; do not extend synthetic E2 claims. |
| Lumen-owned memory denial | Memory subsystem not implemented. | UNSUPPORTED here; retain for its owning later phase. |
| Installed login/reboot and setup publication recovery | Historical LaunchAgent findings CR-02/CR-03. | Phase 10 gates; they do not block the Phase 02 manual foreground Host. |

No further code defect was identified in this scoped review. This does not constitute the full passing Phase 02 review required by Plan 02-12. Keep the phase executing and complete the named profile lifecycle and containment evidence before final ledger reconciliation.

## Coordinator verification refresh — 2026-10-02

The current active working tree passed `rtk go test ./... -count=1`: 502 tests across nine packages. The focused lifecycle contract selection passed 42 tests, and the E2/recovery contract selection passed four tests with `sh -n scripts/lumen-e2-check` also passing. These are local implementation/harness checks; they do not resolve the missing evidence-linked NIM lifecycle probe or profile-specific containment. The independent Phase 02 verdict remains BLOCKED.

## Coordinator verification refresh — 2026-10-02

The owner reconfirmed the NIM default; read-only config inspection confirmed provider/model, mode `0600`, and absence of AgentRouter endpoint/key fields. The local Hermes CLI remains unpinned v0.20.6. Current provider-free lifecycle tests passed both `internal/host` and `internal/hermes` after allowing their ephemeral loopback servers. The E2 contract tests, script syntax, Compose config validation, Phase 02-11 plan-structure validation, roadmap validation, requirement consistency, and `git diff --check` passed. Docker daemon access was denied, so no live NIM allowlist probe ran. Phase 02 remains BLOCKED: no evidence-linked NIM restart/stream-loss/approval/cancellation matrix and no same-runtime NIM-positive/controlled-destination-negative network result. The earlier cancellation attempt remains unrecoverable and was not retried.

The later bounded Docker permission grant enabled a fresh isolated synthetic E2 run (`scripts/lumen-e2-check`) on Docker Desktop Linux/arm64. It passed Host file/secret separation, internal-network sink denial with positive control, runtime tool denial, and Hermes session-context isolation; UID/GID `0:0` was observed with all capabilities dropped and `no-new-privileges`. The unique project was independently absent after cleanup, and the pre-existing `lumen-macos-check-85217` project remained running. See the new 02-11 evidence addendum. This does not change the NIM gate: E2 deliberately has no external network, and the local Hermes CLI remains unpinned. Phase 02 remains BLOCKED pending NIM-profile lifecycle and same-runtime allowlist evidence.

A current-tree `rtk proxy mise run milestone1-macos-check` also passed all nine Go packages and the pinned Hermes 0.21.1 synthetic lifecycle journey. It refreshed denial, cancellation, gateway restart, Host/SSE restart, duplicate/reordered events, and exact held-run stop behavior; the unique project was independently absent afterward and the prior test project remained untouched. See the 02-10 evidence addendum. This substantially refreshes implementation and synthetic-runtime evidence but does not supply NIM-profile lifecycle outcomes or same-runtime NIM allowlist controls; the Phase 02 verdict remains BLOCKED.

## Independent review of refreshed synthetic evidence — 2026-10-02

The independent reviewer confirmed the fresh 02-10 synthetic lifecycle and 02-11 synthetic E2 claims match the recorded observations and cleanup checks. It agreed Phase 02 remains **BLOCKED**. It found a plan wording defect: the previous 02-11 result contract required “all implemented classes” to deny without explicitly separating unsupported memory. The plan now says PASS requires every implemented and supported class to deny, requires unsupported classes to be named and assigned to their later owner, binds file/tool results to the runtime actually tested, and routes Lumen memory to Phase 04/10. The evidence table now explicitly keeps NIM file/secret and tool/self-grant checks BLOCKED because the observed denials were synthetic-profile only.

Remaining Phase 02 gates: NIM-profile lifecycle (restart, stream loss, approval, cancellation/race) and an enforceable same-runtime NIM endpoint positive plus controlled-destination negative network result; NIM file/secret and tool/self-grant boundaries also need evidence tied to that actual NIM profile. The current synthetic passes do not transfer across profiles. Memory denial remains UNSUPPORTED until Lumen-owned memory exists. Phase 10 login/reboot and setup-publication recovery remain correctly deferred and are not blockers for this local manual-Host phase.

## Current disposition after the combined NIM profile run — 2026-10-02

The paragraph immediately above records the state before the combined NIM probe and is superseded for file/secret and network controls by the post-hardlink-guard run in `02-11-EVIDENCE.md`. Its exact project, image digest, config digest, outcomes and cleanup are recorded there. Current independent disposition:

| Plan | Current result | Remaining acceptance |
|---|---|---|
| 02-08 | PASS, scoped | One pinned NIM Host task durably completed; answer quality was not assessed. AgentRouter HTTP 401 and the unrecoverable AgentRouter task remain separate historical outcomes. |
| 02-09 | BLOCKED | Profile/config identity and empty tool inventory are recorded; the explicit terminal request had no effect, but no observable runtime denial for an attempted tool call was retained. |
| 02-10 | BLOCKED for NIM | Full Go suite and synthetic lifecycle journey pass. NIM restart, stream loss, held-run cancellation, exact approval/denial and terminal-race observations remain absent. |
| 02-11 | PARTIAL PASS / BLOCKED overall | Same-runtime NIM file/secret and tested endpoint allowlist controls pass. NIM tool/self-grant denial remains blocked; memory is unsupported until Phase 04. |
| 02-12 | BLOCKED | Complete only after 02-09 through 02-11 have final evidence and a final independent goal-backward audit. |

The current working tree's full Go suite passed across all nine packages. Roadmap validation, requirement consistency and plan structure checks passed; Phase 01 verification is current. The latest scoped independent review found the hard-link guard and plan ownership correction sound and confirmed that the NIM evidence matches its bounded claims. The overall Phase 02 goal remains **BLOCKED**, not complete, by the observable NIM tool-denial and NIM lifecycle requirements. Lumen-owned memory remains UNSUPPORTED here; Phase 10 login/reboot, installer, update/rollback and laptop-off acceptance stay deferred as planned.


## Independent acceptance and config correction — 2026-10-02

The independent acceptance review confirms Plan 02-09 requires an observable denial for an attempted unapproved tool action as well as no side effect. The retained NIM run shows zero enabled API toolsets and an absent canary, but retained neither assistant output nor a tool-call/denial event. This is insufficient to determine whether policy denied an attempt or the model did not attempt it. Keep the item BLOCKED; do not infer success from the absent canary alone.

A later redacted config inspection corrected the NIM addendum: the selected top-level default is NVIDIA NIM, and two AgentRouter custom-provider entries remain in the private config. No credential values were read and the config was not changed.


## Follow-up config correction — 2026-10-02

After the owner reiterated the NVIDIA NIM default, the private Hermes config was updated to remove the two legacy AgentRouter custom-provider entries; the NIM `model.provider` and `model.default` were preserved, all unrelated settings were preserved, and file mode remained `0600`. Ruby's YAML parser accepted the edited file. Hermes 0.20.6 `doctor` recognized the config version and configured provider; its NVIDIA NIM reachability check returned a DNS-resolution error under the restricted shell. No model-generation request was made. This is not evidence of NIM outage or successful pinned-Lumen runtime connectivity.


## Follow-up private config result — 2026-10-02

After the owner reiterated the NVIDIA NIM default, the private Hermes config was updated to remove the two legacy AgentRouter custom-provider entries while preserving the selected `model.provider: nvidia`, `model.default: nvidia/nemotron-3.5-lightning-30b-a3b`, unrelated settings, and file mode `0600`. YAML parsing passed. Hermes 0.20.6 `doctor` recognized the config version/provider, but its NVIDIA NIM reachability check returned a DNS-resolution error under the restricted shell. No model-generation request was made; this is neither a provider outage finding nor evidence for the pinned Lumen runtime.


## Scope reconciliation after the approved product-first clarification — 2026-10-02

The preceding BLOCKED rows are historical snapshots and are superseded by the following current scope, pending the final Plan 02-12 code/evidence review:

| Gate | Current scoped result | Remaining disposition |
|---|---|---|
| 02-08 real provider | PASS for one Host-mediated NIM completion and truthful provider-failure receipt; response quality not assessed. | Keep the selected NIM default; no AgentRouter fallback. |
| 02-09 chat-only boundary | PASS for observed runtime identity, zero enabled toolsets, and no-effect explicit action canary. | Do not claim an attempted model tool-call denial; event-level semantics are UNSUPPORTED until Phase 07 adds an actual broker. |
| 02-10 lifecycle | PASS on pinned Hermes 0.21.1 with deterministic synthetic provider for mapping, approval, denial, cancellation, stop, stream loss and restart. | This is runtime lifecycle evidence, not NIM-specific behavior. NIM is separately qualified for one real Host result. |
| 02-11 containment | PASS for same-runtime NIM file/secret separation and tested endpoint allowlist/controlled-destination behavior; synthetic E2 tool enforcement remains separate. | Memory is UNSUPPORTED until Phase 04; model-call event denial is UNSUPPORTED until Phase 07. |
| 02-12 final audit | OPEN. | Review implementation, exact evidence/revision identity, all owned FRs and planning contradictions; only then record phase outcome. |

The user-approved Phase 02 goal is the local Host/Hermes foundation before conversation/product capabilities rely on it. NIM-specific restart/approval/cancellation testing and an event for a tool that does not exist in the chat profile are not added exit requirements. VPS, installer, reboot, update/rollback, laptop-off and soak remain Phase 10.


## Final independent review — 2026-10-02

**Verdict: PASS for the bounded Phase 02 local Host/Hermes foundation.** The independent read-only verifier reviewed the current active branch's Host execution, Hermes adapter, Space approval validation, Compose runtime binding, and current evidence under the clarified product-first scope. It found no new source-level security failure. Its source inspection confirmed durable create intent and idempotency, honest ambiguous outcomes, exact Run reconciliation, action/run/profile/expiry-bound approval, cancellation requiring stop/reconciliation evidence, restart recovery, secret file validation, and a zero-tool chat configuration. The review specifically accepted the no-tool/no-effect canary as evidence for chat-only no-transitive-authority; it did not claim model-originated tool-denial events.

| Requirement | Phase 02 outcome | Evidence |
|---|---|---|
| FR-01 | PASS for local Space/Host identity and durable recovery contract | `02-01-EVIDENCE.md`, `02-10-EVIDENCE.md`; current host/setup tests |
| FR-06 | PASS for local foreground Host and supervisor contract | `02-01-EVIDENCE.md`; setup/supervisor tests and `mise run phase0-check` |
| FR-38 | PASS for pinned Hermes adapter discovery, runs, events, approvals, cancellation and recovery contract | `02-08-EVIDENCE.md`, `02-10-EVIDENCE.md`; Host/Hermes tests |
| FR-41 | PASS for Host-local chat with no transitive authority in the declared zero-tool profile | `02-09-EVIDENCE.md`, `02-11-EVIDENCE.md`; zero toolsets and no-effect action canary |
| FR-69 | PASS for the owner-selected Hermes/NIM runtime qualification | `02-08-EVIDENCE.md`; one durable NIM Host result, response quality unassessed |

**Evidence boundary:** the reviewer did not independently rerun tests, provider generation, Docker containment probes, or teardown queries. The coordinator's current `rtk go test ./... -count=1` passed 502 tests across nine packages; `rtk mise run phase0-check`, roadmap validation, requirement consistency, plan structure, summary verification, and the 58-ID ownership audit passed. Live results remain those in the append-only 02-08 through 02-11 evidence. NIM provider response, pinned-Hermes synthetic lifecycle, zero-tool/no-effect canary, and scoped NIM file/network containment are separate claims.

**Deferred, not passed:** model-originated tool-event denial belongs to Phase 07; Lumen-owned memory denial belongs to Phase 04; installer/publication, login/reboot, update/rollback, external Runs, backup/restore, laptop-off continuity and soak belong to Phase 10. No VPS or deployment gate is represented as complete here. The stale BLOCKED snapshots above are historical and superseded by this final scoped disposition; their underlying observations remain unchanged.
