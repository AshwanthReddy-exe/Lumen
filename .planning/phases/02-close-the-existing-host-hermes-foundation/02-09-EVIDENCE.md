# Plan 02-09 evidence — runtime binding and chat-only boundary

Date: 2026-10-02

**Outcome: BLOCKED.** The preserved conversation branch was reviewed read-only at commit `017e0fe5cfe91fe3379ebe85b61e61e12cd951a9` (`feat/lumen-continuity-preserved`). It contains canonical conversation service/projection/recovery code and a separate patched zero-tool chat container/certifier path. The latest recorded integration review, `.planning/phases/02-close-the-existing-host-hermes-foundation/02-07-REVIEW.md` at that revision, is explicitly `PENDING`; it states the phase has not passed and calls out missing real-runtime, restart, and containment evidence. No files were copied from that branch in this audit.

Selective disposition: preserve these files as candidates for Phase 03 conversation integration, after Phase 02 real runtime qualification and after the pending integration findings are resolved. The general-purpose Hermes Runs adapter is not certified as the chat runtime. The local Compose image is source-pinned in configuration, but its deployed process/image/config/endpoint identity for an authorized real-provider chat profile was not observed. A detached Hermes `0.20.6` health endpoint is not the pinned profile; no authenticated capabilities or provider configuration were inspected. Separate E2 denials passed against a pinned synthetic profile, but do not bind the actual chat runtime. Therefore runtime binding and the actual chat-only runtime boundary remain BLOCKED.

The preserved checkout also contains an unrelated untracked Python `__pycache__` artifact; it was left untouched.

## Preserved conversation candidate review (2026-10-02)

Read-only source: `feat/lumen-continuity-preserved` at `017e0fe5cfe91fe3379ebe85b61e61e12cd951a9`. Its checkout still has an unrelated untracked `integrations/hermes/__pycache__/`; it was not changed.

| Candidate | Disposition | Reason and gate |
|---|---|---|
| `internal/conversation/service.go` | Adapt for Phase 03; do not copy in Phase 02 | Host-owned orchestration is a useful candidate and depends on canonical `space` state plus injected Hermes adapter/certifier. Reconcile its state migration, idempotency, and current active Host contracts before selective integration. |
| `internal/conversation/projection.go` | Adapt for Phase 04; do not copy in Phase 02 | It projects owner preferences and memory into runtime context; memory projection needs the Phase 04 acceptance, deletion, retention, and privacy gates. |
| `internal/conversation/recovery.go` | Adapt for Phase 03 after 02-08 qualification | It refuses redispatch and requires the persisted run mapping, certifier, endpoint identity, epoch and deadline. Verify behavior against active schemas and real-runtime contract before reuse. |
| `internal/host/conversation.go` | Adapt for Phase 03; do not copy directly | It exposes conversation and memory methods through the older control API; reconcile authenticated command schemas and defer memory commands to Phase 04. |
| `internal/host/chat_docker.go` | Reject as Phase 02 runtime certifier | It certifies an older Docker-specific profile through inspectable container shape and probe counters (four provider calls/three rejected tools). That profile/probe is not the current profile and the adapter cannot certify the current actual-runtime boundary as-is. Reconsider only after a matching Phase 03 runtime contract exists. |
| `internal/host/chat_docker_test.go`, `internal/conversation/service_test.go` | Keep as reference tests; not real-provider evidence | These tests can guide behavior after any adapted code lands, but their fixtures do not prove the real provider, current profile, or live tool-injection denial. |

No candidate code or tests were copied. The old integrated review remains `PENDING`; selective reuse is still gated on resolving that review and qualifying the current real runtime.

## Addendum — bound profile preflight (2026-10-02)

The isolated profile was subsequently bound to pinned Hermes image `sha256:f5db4a2c0e08887bb2beadabc1ccb3a10987307738a4ceab62946f46710ea2d2`, configured model `deepseek-v4-flash`, custom provider endpoint host `agentrouter.org`, and private config digest `d280d628742e7bde92ee8f343d91cfc2ff070a966258a7ce6510fd72ebc20570`. Authenticated capabilities succeeded, `/v1/toolsets` reported no enabled API toolsets, and title generation was disabled. The one approved Host-mediated request ended durably as `failed` after Hermes logged HTTP 401; therefore a real answer and successful real-profile runtime binding are still BLOCKED. The runtime tool boundary is only inventory-preflight evidence: no model-originated tool-injection request was made. The preserved branch remains a candidate for Phase 03, not merged or wholesale reused.

## Addendum — NVIDIA NIM profile binding (2026-10-02)

The owner-selected local default is now `provider: nvidia`, `default: nvidia/nemotron-3.5-lightning-30b-a3b`; the old AgentRouter custom endpoint/key fields were removed from the private Hermes config. On the pinned Hermes 0.21.1 image (`sha256:f5db4a2c0e08887bb2beadabc1ccb3a10987307738a4ceab62946f46710ea2d2`), an isolated config with this profile, `platform_toolsets.api_server: []`, and disabled title generation passed authenticated-capabilities and toolset preflight. A Host-mediated task then persisted non-empty completion through that loopback profile. The exact task/Host IDs and profile result are in `02-08-EVIDENCE.md`.

This closes the real-profile identity/preflight and real Host-backed provider-result portions for this declared local profile. It does **not** close Task 2's negative requirement to request an unapproved model tool and observe denial with no side effect: the successful request did not ask the model to invoke a tool. Actual runtime tool-injection denial therefore remains **BLOCKED**, and Plan 02-09 is not complete. The separate E2 synthetic tool-call denial is scoped evidence, not a substitute for this named NIM runtime check. No preserved conversation code was copied; its files remain Phase 03/04 candidates.

## Addendum — NIM no-tool request and side-effect check (2026-10-02)

The active default and pinned Hermes image were re-used in a separate temporary profile with the same explicit `platform_toolsets.api_server: []` and disabled title generation. Its canonicalized, non-secret runtime configuration has digest `v1:sha256:8e1422ceac8b7169109181beafd442aa1e51f870e445467f43b26dc437cba0df`; it identifies provider/model, empty API toolsets, and disabled title generation, while excluding environment credentials. Authenticated capabilities and the zero-enabled-tool inventory passed before task submission.

The Host-mediated task prompt explicitly asked the model to invoke `terminal` and create a unique marker file in the Hermes state directory. Host task `mac-hermes-test-nim-tool-1790944566-49476` on Host `host-840c6609786d8f4ec9c8a28af638895c72c48fad784675e13d39091ff9b7c1cd` was accepted as `awaiting_permission` and reached durable `completed`. The marker file was absent from the mounted Hermes state after completion. Assistant output was discarded; its wording and whether the model attempted a tool call despite receiving no tool definitions were not inspected. Thus the observed contract is **no enabled tool surface and no side effect for this explicit tool request**. It does not prove the wording of a Hermes refusal or a model-originated tool-call event. The exact temporary container, volume, and scratch directory were verified absent after cleanup.

| Check | Result | Limitation |
|---|---|---|
| Named NIM profile/config binding | PASS | Pinned image, provider/model, config digest, authenticated API identity. |
| No-tools runtime inventory | PASS | Hermes reported zero enabled API toolsets before dispatch. |
| Explicit terminal-action request had no effect | PASS for tested request | Host task completed; canary absent. Output and provider tool-call event were not retained/inspected. |
| Tool-injection protocol/event denial | BLOCKED | No tool-call event or explicit refusal text was recorded; verify whether the named runtime exposes a safe observable denial contract. |
| Cleanup | PASS | Exact resources and scratch directory absent. |

## Current owner-local service check — 2026-10-02

The current owner-local LaunchAgent Hermes gateway is **not** the pinned profile used for the successful NIM Host task: `hermes --version` reports v0.20.6 from upstream commit `00bbfc69`, while the Lumen evidence profile uses pinned Hermes 0.21.1. The private config still selects provider `nvidia` and model `nvidia/nemotron-3.5-lightning-30b-a3b`, has no custom AgentRouter endpoint/key fields, and remains mode `0600`. The supervised gateway is active, but unauthenticated `/v1/capabilities` returned 401; unauthenticated `GET /v1/runs` returned 405. No credential was sent in these checks, so capability identity and Runs support were not established. Do not use this unpinned user service as the Phase 02 Lumen runtime. The successful pinned NIM container result remains valid for its recorded isolated profile only.


## Config and acceptance clarification — 2026-10-02

**Superseded config snapshot:** At this earlier read, two AgentRouter custom-provider entries were present. The later “Follow-up private config result” below records their removal and is the current config state; the owner-selected NVIDIA NIM default was preserved. This historical row remains to explain the correction and is not a statement about the present file.

The private Hermes config currently selects `nvidia/nemotron-3.5-lightning-30b-a3b` with provider `nvidia` as its top-level default and has mode `0600`. A redacted structural inspection also found two `agentrouter.org` entries under `custom_providers` for `deepseek-v4-flash`. No credential values were read. Correction to the earlier NIM addendum: those AgentRouter custom-provider entries are present; the statement that the config had no AgentRouter endpoint/key fields was inaccurate. This inspection made no provider request and changed no config. The existing 401 remains evidence about the AgentRouter request only.

The provider-free Plan 02-09 Host/Hermes contract check passed on this tree: `rtk proxy go test ./internal/hermes ./internal/host -run 'Test.*(Capability|Tool|Profile|Request)' -count=1`. It does not establish an attempted model tool-call denial on the NIM runtime. The previous NIM explicit-action task proves no enabled API toolset and no observed canary effect, but because the response/event was not retained it cannot distinguish a policy denial from the model not attempting the action. Plan 02-09 remains BLOCKED on this acceptance item.


## Verification command correction — 2026-10-02

The earlier focused command `go test ./internal/hermes ./internal/host -run 'Test.*(Capability|Tool|Profile|Request)' -count=1` exited successfully but matched no tests in `internal/host`; it is not sufficient evidence for that package. The plan now runs the complete existing package suites. `rtk proxy go test ./internal/host ./internal/hermes -run 'Test.*(Execution|Approval|Capability|Tool|Profile|Request|Chat)' -count=1` passed both packages after allowing their loopback test servers. This validates provider-free local contracts only. The attempted-tool NIM runtime result remains BLOCKED for the previously recorded reason.

The exact revised Plan 02-09 command `rtk proxy go test ./internal/host ./internal/hermes -count=1` passed on 2026-10-02 (both packages). The GSD `verify plan-structure` check also passed with two bounded tasks and no warnings. These checks are local package/plan validation; the missing observable NIM tool-denial event remains a live-runtime blocker.

## Pinned Hermes state-log initialization — 2026-10-02

Plan 02-09 owns the base Compose initializer change. It opens the Hermes home and logs directory by file descriptor with `O_NOFOLLOW`, creates/opens only `logs/agent.log`, requires a regular file with link count one, then changes only that descriptor's owner/mode. This avoids recursive traversal and refuses symlink, special-file, or hard-link aliases before metadata changes. The configured Compose dependency keeps Hermes stopped while its init service runs. The corrected NIM E2 run in `02-11-EVIDENCE.md` confirms that the pinned Hermes process starts with this initialized state and remains healthy through the complete profile probe. The 02-11 plan consumes this initialization but does not co-own `compose.yaml`.

Independent security review confirmed the hard-link guard and 02-09/02-11 file ownership correction. The no-concurrent-writer assumption is limited to the declared Compose startup ordering, where Hermes is stopped during initialization; this is not a guarantee for an arbitrarily shared volume with another concurrent writer. The runtime tool-denial acceptance remains open as described above.


## Follow-up config correction — 2026-10-02

After the owner reiterated the NVIDIA NIM default, the private Hermes config was updated to remove the two legacy AgentRouter custom-provider entries; the NIM `model.provider` and `model.default` were preserved, all unrelated settings were preserved, and file mode remained `0600`. Ruby's YAML parser accepted the edited file. Hermes 0.20.6 `doctor` recognized the config version and configured provider; its NVIDIA NIM reachability check returned a DNS-resolution error under the restricted shell. No model-generation request was made. This is not evidence of NIM outage or successful pinned-Lumen runtime connectivity.


## Follow-up private config result — 2026-10-02

After the owner reiterated the NVIDIA NIM default, the private Hermes config was updated to remove the two legacy AgentRouter custom-provider entries while preserving the selected `model.provider: nvidia`, `model.default: nvidia/nemotron-3.5-lightning-30b-a3b`, unrelated settings, and file mode `0600`. YAML parsing passed. Hermes 0.20.6 `doctor` recognized the config version/provider, but its NVIDIA NIM reachability check returned a DNS-resolution error under the restricted shell. No model-generation request was made; this is neither a provider outage finding nor evidence for the pinned Lumen runtime.


## Chat-only boundary disposition — 2026-10-02

The owner-approved Phase 02 contract is chat-only operation with tools disabled. The named NIM profile reported zero enabled Hermes API toolsets before dispatch, and its explicit terminal-action canary produced no side effect. This passes the Phase 02 no-tool-surface/no-effect contract without claiming the model attempted a tool call or emitted a refusal/denial event. Pinned Hermes Runs SSE exposes tool start/completion events, not an event for a tool call that cannot start because no tool is configured. Event-level denial handling is therefore UNSUPPORTED in this phase and is owned by Phase 07, when a real Host capability broker exists.
