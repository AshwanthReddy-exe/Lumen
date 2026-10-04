---
gsd_state_version: "1.0"
current_phase: 02
current_phase_name: Local Host/Hermes product foundation
status: complete
stopped_at: "Plan 02-12 recorded the independent bounded PASS; all six scoped Phase 02 plans are complete. The next phase is conversation and web."
last_updated: "2026-10-02T15:42:45Z"
last_activity: 2026-10-02
last_activity_desc: "Completed Plan 02-12 after independent read-only review found no new source-level security failure. Full Go suite (502 tests), phase0-check, GSD plan/summary validators, roadmap/requirements consistency, and the 58-ID ownership audit passed. Phase 02 is complete with deferred gates assigned to Phases 04, 07 and 10."
state_head: ff12f4c1b0644f73dd2488f014150d1b232ee241
progress:
  total_phases: 17
  completed_phases: 2
  total_plans: 17
  completed_plans: 9
  percent: 53
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-26)

**Core value:** One conversation continues from laptop to phone after laptop shutdown, then an approved phone request produces a verifiable laptop receipt in that conversation.
**Current focus:** Phase 02 — Local Host/Hermes product foundation

## Current Position

Phase: 02 (Local Host/Hermes product foundation) — COMPLETE
Plan: 06 of 06 — all scoped plans are complete: 02-06 and 02-08 through 02-12.
Status: Phase 02 complete; Phase 03 is next.
Last activity: 2026-10-02 — Independent review passed the bounded local foundation scope. Real NIM response, pinned-Hermes synthetic lifecycle, no-tool/no-effect chat boundary, and scoped NIM containment remain separate evidence claims.

Progress: [█████░░░░░] 53%

## Performance Metrics

**Velocity:** Three Phase 01 documentation/evidence plans completed; subsequent-phase timing unavailable.

## Accumulated Context

### Decisions

- Accepted decisions in [docs/DECISIONS.md](../docs/DECISIONS.md) are locked; superseded entries are historical.
- Personal alpha uses one always-on Host and a pinned Hermes runtime; canonical Space authority stays in the Host.
- Shared mobile UI remains experiment-gated with documented Android-native plus phone-web fallback.

### Pending Todos

- Phase 02 is local Host/Hermes qualification. The owner selected NVIDIA NIM `nvidia/nemotron-3.5-lightning-30b-a3b` as the local default. Plan 02-08 has one successful pinned Hermes 0.21.1 Host-mediated completion with non-empty durable output; the response was discarded and answer quality was not assessed. A separate terminal canary request completed with no side effect while the configured API toolsets were empty; response text and model tool-call events were not retained. This supports only the bounded no-effect observation, not proof of a model refusal or event handling. AgentRouter HTTP 401 and the subsequent unrecoverable `task_unknown` attempt remain historical. Do not print or persist credentials, prompts, or assistant content.
- Existing setup/recovery evidence remains in `02-01-EVIDENCE.md`. Service readiness polls for all owned services and authenticated Host readiness; live installer publication/login/reboot evidence remains a Phase 10 gate. The pinned Runs API and `hermes serve` are different services and must not be conflated.
- Historical VPS setup and pinned-Hermes evidence remain linked in `02-02-EVIDENCE.md`; the latest bounded retry was cancelled before setup, so it produced no result and no reboot. A 2026-10-02 anonymous GHCR manifest check was denied; the Hermes **user** gateway was active at unpinned revision `37daf85`, with no Runs listener. Neither was changed. Deployment, reboot, update/rollback and laptop-off gates move to Phase 10 under the product-first sequence. Do not add `ubuntu` to the root-equivalent docker group.
- Phase 02 completed with a bounded local Host/Hermes foundation PASS. The NIM response is real and durable; response quality was not assessed. Lifecycle semantics passed on pinned Hermes with a deterministic synthetic provider, not NIM. NIM file/secret and tested network controls pass for the recorded profile. Model tool-event denial is UNSUPPORTED until Phase 07; memory denial is UNSUPPORTED until Phase 04. Phone, physical-device, Mac-node, voice, installer, laptop-off and soak work remains in later phases.

### Blockers/Concerns

- The owner approved Phase 01 and PR #21 merged on 2026-09-27; the Phase 02 branch starts at `81b60e3`.
- Historical VPS verification first returned HTTP 401 for the pinned Hermes GHCR digest, then a 2026-09-28 anonymous check with a fresh empty `DOCKER_CONFIG` retrieved the exact OCI index and both Linux child manifests. The later 2026-10-02 check was denied. Current anonymous visibility and full image-layer pull are therefore unverified; do not treat the September result as current availability. Evidence is in `02-02-EVIDENCE.md`.
- Phase 10 live gates require supported pinned Hermes images, update/rollback, and a second machine for external Runs; no synthetic probe can stand in for those results.
- PR #20 is closed after useful dirty work was preserved on a verified remote branch. Its AgentRouter HTTP 401 and later `task_unknown` attempt are historical; the named NIM profile has since returned a persisted Host result.
- Plan 03 exact-run Host/SSE interruption passed on macOS: task `macos-e1-1790533614-80598`, Run `run_a3fd9ed35bc7421fbdae0e1b0aca7937`, pre-kill `running`, same durable mapping after restart, one create, and unchanged encrypted state across SIGKILL; log `/Users/ashwanthreddyboddireddy/Library/Application Support/rtk/tee/1790533636_mise_run_f350c7.log`. The probe binds the SSE cut to the exact Run returned for the armed idempotency key, preventing unrelated streams from consuming it. Full macOS journey and all nine Go packages passed; this is pinned synthetic-provider lifecycle evidence only.
- Duplicate/reordered SSE handling was fault-injected on active pinned Run `run_2fb9c88e3e024e3485a93bfef9bb5c1c`: Host consumed out-of-order IDs 2,1, ignored conflicting duplicate ID 2 while Hermes/provider remained held, accepted barrier ID 3, then converged to completed after provider release; one Run create. Full journey passed in log `/Users/ashwanthreddyboddireddy/Library/Application Support/rtk/tee/1790535003_mise_run_f350c7.log`. This does not certify Hermes-native event IDs or replay semantics. Independent security review passed for the probe and bounded loopback test endpoints.
- Plan 03 stop is now verified on a held pinned Run: task `macos-reordered-1790535955-2533`, Run `run_f78dc17659df4f38a6fc80c2f8d85042`; Host and Hermes were both `running` immediately before stop, both became `cancelled` while the provider remained held, and stayed cancelled after release. One Run create; full native journey and nine Go packages passed in `/Users/ashwanthreddyboddireddy/Library/Application Support/rtk/tee/1790535984_mise_run_f350c7.log`. This is a pinned synthetic-provider result only.
- Phase 02 plan 01 restart-identity probe passed on the named Mac: E1 task `macos-e1-1790536770-9936`, Run `run_6205919d35404c3486a9935289d78de7`; Space/owner/Host IDs, initialized-marker digest, operator credential digest/mode, and doctor initialized observation were preserved across foreground Host `SIGKILL`/restart. The task reconciled once to completed. Evidence log `/Users/ashwanthreddyboddireddy/Library/Application Support/rtk/tee/1790536805_mise_run_f350c7.log`; this does **not** prove interrupted public setup/rerun or a real provider result.
- Historical independent LaunchAgent review `.planning/phases/02-close-the-existing-host-hermes-foundation/02-REVIEW.md` closed CR-01 and exact-binding journal authorization; CR-04 remains closed. Home lookup uses the OS account record and fails closed for `osusergo`. The staged-to-published transition reloads only an owned definition, rejects uncertain launchctl outcomes, and checks Launchd plus authenticated Host readiness. CR-03 remains open for legacy plist migration and post-publication recovery semantics; CR-02 login scope/live login-reboot evidence and WR-02 readiness polling remain open. See the 2026-09-28 gate follow-up below for the narrowed activation rule. Ordinary clarification remains unsupported. VPS reachability was proven earlier, but current SSH checks time out; deployment/update probes still need a confirmed remote state and runnable OCI client. The 2026-10-02 anonymous manifest check was denied, so the September image visibility result is historical only.
- 2026-09-28 activation-gate follow-up: `lumen-host serve` denies ungated macOS LaunchAgent activation; `serve --manual` preserves intentional foreground use. Linux/Termux are unaffected when the validation variable is absent. Generated staged LaunchAgents use `serve --setup-staging`, requiring the exact external/Launchd journal binding and setup-stage window; if terminal validation evidence exists, the exact validated binding is required. Published plists carry the validated-setup gate and no staging argument. Independent security review passed this narrow change. Focused checks passed: `go test ./cmd/lumen-host -run '^TestLaunchActivationGate|^TestCLIProcessLifecycleAndBoundary$' -count=1` (4 tests), `go test ./cmd/lumen -run '^TestWriteLaunchAgentIsPrivateEscapedAndIdempotent$' -count=1` (1 test), `git diff --check`; Graphify refreshed. Legacy plist migration and live macOS behavior remain separate open gates.
- VPS Hermes CLI install completed on 2026-09-28 using the persistent SSH PTY. The current public installer was incompatible with Lumen's older pinned commit (it requires a `pm/` tree absent at that SHA); the official `scripts/install.sh` from the exact pinned checkout was used with setup/browser/computer-use skipped. Re-verified `hermes --version` = `v0.21.1 (2026.9.7)`, Python `3.11.16`, checkout SHA `2237be355906fbe6065ce1815711eee52b2d646e`, and successful `hermes gateway --help`. No provider config, gateway, service, or model request was made. Removed partial `.hermes`, test temp, rootless Docker unit/state, Docker apt source/key, and rootful Docker service/socket plus test data directories. Then purged only the six task-installed Docker packages `docker-ce`, `docker-ce-cli`, `docker-ce-rootless-extras`, `docker-buildx-plugin`, `docker-compose-plugin`, and `containerd.io`; exact post-checks confirmed these packages, Docker system/rootless data and apt source/key are absent, with all Docker units inactive and not-found. The task-installed Python development residue initially blocked direct `dpkg` purge. A package-specific APT simulation showed that reinstalling `python3-dev` would configure exactly the six affected development packages without removals; that targeted reinstall completed, processed `man-db`, and was followed by a second simulation showing purge of exactly those six packages. The exact purge succeeded. Final `sudo dpkg --audit` output was empty, Hermes remained at SHA `2237be355906fbe6065ce1815711eee52b2d646e`, `hermes --version` and `hermes gateway --help` passed, and `build-essential`/`uidmap` remained installed. No autoremove was run; `pigz` remains an apt-reported autoremove candidate and was intentionally left untouched. Preserved SSH/admin access, `.docker/config.json`, owner proof, `/etc/containerd/config.toml`, and the installed Hermes CLI. The installer left only upstream blank/example config templates. No persistent installer log exists (success path removed the temporary dependency log). This establishes CLI installation only, not Lumen adoption or cross-machine Runs.
- 2026-09-28 bounded live gateway API probe: The blank/example profile and a process-only `API_SERVER_KEY` did not expose port `:8642`. In a fresh isolated `HERMES_HOME`, explicit `platforms.api_server.enabled: true`, loopback host/port, and a process-only dummy key produced a loopback listener. `/health` returned Hermes 0.21.1 status ok; unauthenticated `/v1/capabilities` returned HTTP 401, while the bearer-authenticated response declared required authentication and Runs submission/status/events/stop/approval plus durable idempotency. No `/v1/runs` call occurred; no provider credential was inspected or intentionally used and no model request was sent. The gateway was stopped and the exact temporary profile/log/state removed. This proves pinned Hermes API readiness/configuration only, not Lumen adoption or Run behavior.
- 2026-09-28 Lumen external adoption check: The actual adoption validator passed against authenticated pinned Hermes through the persistent tunnel (health/capabilities RTT 34/28 ms); no Run was sent. Secure repo-local test paths avoided Lumen's deliberate world-writable-parent rejection. Setup initialized Host state and started its staged macOS agent; doctor returned ready. Setup and rerun still fail at validated publication, with the staged plist loaded but no published plist. Full evidence and cleanup are in `02-02-EVIDENCE.md`; diagnose this activation transition next.
- Owner-requested current Hermes installer attempt on 2026-09-28: stopped and removed only the exact prior temporary Phase 02 gateway/profile, preserving `/home/ubuntu/.hermes`. The official installer changed the source checkout to upstream `37daf85b`, preserved a local-only commit under `refs/hermes-update-backups/...`, and became silent at Node dependency preparation. The SSH session was interrupted; final version/service/process verification is unknown, and a verification reconnect was denied by the local network sandbox. This is not pinned Hermes evidence. Before another VPS gate, inspect once and restore/use pinned `2237be355906fbe6065ce1815711eee52b2d646e` as needed. Details: `02-02-EVIDENCE.md`.
- Latest read-only VPS preflight (2026-09-28 16:18 UTC) confirmed the owner's current Hermes checkout remains `37daf85`, its user gateway service is active, no Lumen CLI is installed, and no Hermes Runs API TCP listener is present. Preserve the active gateway; older pinned Hermes/API/adoption results are historical isolated-profile evidence and do not prove the present service is pinned or Run-capable. Details and exact scope are in `02-02-EVIDENCE.md`.
- Fresh GitHub read-only verification (2026-09-28 16:28 UTC) confirmed draft PR #20 is closed and unmerged, the original branch remains, and `feat/lumen-continuity-preserved` points to the locally verified `017e0fe5cfe91fe3379ebe85b61e61e12cd951a9`. Updated the stale plan 06 summary that had said current GitHub verification was unavailable. No PR or branch mutation occurred; details are in `02-06-EVIDENCE.md`.
- Fresh isolated pinned Hermes install (2026-09-28 16:44 UTC) passed in `/home/ubuntu/lumen-phase02-hermes.abTeR4/home`, with source `2237be355906fbe6065ce1815711eee52b2d646e`, Hermes `v0.21.1`, Python 3.11.16; existing owner gateway remained active and no provider setup occurred. This is CLI-install evidence only; Runs API, Lumen adoption/update and real cross-machine Run remain open. Exact evidence and installer overhead are in `02-02-EVIDENCE.md`.
- Isolated pinned Runs API readiness (2026-09-28 16:50 UTC) passed: health 200, unauthenticated capabilities 401, authenticated capabilities included Runs operations and durable idempotency; response confirmed server-side tool execution and does not prove isolation. Test API was stopped, port 8642 verified closed, owner gateway remained active. No Run or model/provider request occurred. Restrict and verify the isolated API toolsets before Lumen adoption. Evidence: `02-02-EVIDENCE.md`.
- Phase 02 plan 01 local setup regression: confirmed that `Validated` redundantly called `host.VerifyInitialized` while the active Host held the state-store lock, failing before LaunchAgent finalization. Removed only that second read; pre-start initialization verification and post-start supervisor/authenticated readiness remain. Red/green regression and focused setup tests pass; independent scoped review found no issue. Live macOS publication/login/reboot remains unverified. Details: `02-01-EVIDENCE.md`.
- Focused plan 01 recovery tests were rerun on 2026-09-27: partial bootstrap rejects without mutation; Host identity verification is stable; setup journal resumes after an injected interruption and skips completed initialization on rerun. Four tests passed; evidence is in `02-01-EVIDENCE.md`. The public CLI/service-manager interruption remains unverified.
- Physical device, recovery and seven-day soak evidence remain unverified.

- 2026-09-28 Phase 02 E1 recovery fix verified: SSE deadline now falls through to bounded Hermes RunStatus reconciliation instead of prematurely persisting `unknown_outcome`. Targeted recovery tests, the full Darwin synthetic `milestone1-macos-check`, and `phase2-check` passed. Exact Run, logs, and limitations are in `02-01-EVIDENCE.md`. E2 probes on local Docker Desktop/arm64 passed for Host/Hermes mount separation, the exact cross-network sink denial, no-schema tool-call rejection, forged Room-policy denials, and separation between explicit Hermes API sessions (with same-session history positive control). This is limited synthetic-profile evidence, not Linux/amd64 VPS or externally managed Hermes certification. No Lumen-owned memory/context exists yet, so that distinct E2 class remains untestable; unrestricted production egress also remains. Details in `02-05-EVIDENCE.md`.
- On 2026-10-02, the local E2 script exited 0 against pinned Hermes with a synthetic provider on Docker Desktop Linux/arm64. File/mount, controlled-network, and tool/self-grant denials passed. The test-only profile accepts Host UID/GID `0:0` with all Linux capabilities dropped and no-new-privileges; independent cleanup checks passed. This does not close real chat runtime binding or certify ordinary Host deployment/personal data. Lumen-owned memory denial remains unsupported until memory exists. Details in `02-11-EVIDENCE.md`.

## Deferred Items

| Category | Item | Status | Deferred At | Milestone |
|---|---|---|---|---|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-10-02
Stopped at: Phase 02 is complete. The private Hermes default remains NVIDIA NIM `nvidia/nemotron-3.5-lightning-30b-a3b`. The bounded local foundation has one real NIM Host completion, pinned-Hermes synthetic lifecycle evidence, chat-only zero-tool/no-effect evidence, and profile-scoped NIM file/network controls.
Next: begin Phase 03 conversation and web planning. Event-level tool denial remains in Phase 07, memory denial in Phase 04, and VPS/deployment gates in Phase 10.
Resume file: .planning/phases/03-conversation-and-web/03-01-PLAN.md
