# Plan 02-10 evidence — durable Run lifecycle

Date: 2026-10-02

**Outcome: PASS for the current pinned synthetic macOS lifecycle profile; BLOCKED for the declared real-provider local profile.** On 2026-10-02, `rtk mise run milestone1-macos-check` passed against the current working tree at base revision `ff12f4c1b0644f73dd2488f014150d1b232ee241` plus its preserved uncommitted changes. The full Go suite passed all nine packages before the live journey. Focused lifecycle checks also passed: `rtk go test ./internal/host ./internal/hermes -run 'Test.*(Run|Dispatch|Recover|Restart|Stream|Idempot|Approval|Cancel|Terminal|First|Binding)' -count=1` (42 tests). The synthetic profile used pinned Hermes source commit `2237be355906fbe6065ce1815711eee52b2d646e` and archive SHA-256 `907c2a72db1c5dd637ea8eeae97f4cb5b32cef615c17258f6b190924ec5bf688`; it made no real provider request and used temporary synthetic state.

Observed checkpoints: exact denial persisted as `approval_denied`; in-flight cancellation reached `cancelled`; gateway restart reconciled the same Run with one create; Host restart after a cut SSE stream reconciled Run `run_715027c8e58641099598395997771834` from `running` to `completed` with the same durable mapping and one create; Space/owner/Host identity and initialized/operator state remained preserved; injected SSE frames `2,1,2,3` with a conflicting duplicate did not create duplicate authority transitions; exact in-flight stop reached cancelled on Host and Hermes while inference remained held. The disposable Compose project was `lumen-macos-check-90201`; independent Docker inspection found no matching containers, networks, or volumes after the run. Full redacted transcript: `/Users/ashwanthreddyboddireddy/Library/Application Support/rtk/tee/1790936051_mise_run_f350c7.log`.

The live synthetic run verifies pinned Hermes lifecycle behavior on this macOS test topology, not a named real-provider profile. A separate unmanaged Hermes gateway listener observed during this turn was not adopted as Lumen's profile: `hermes gateway status` reported its launchd service not loaded and a detached process only, the Lumen Host socket was absent, and an HTTP health request refused connection. Do not infer provider identity or credential validity from that listener. Therefore real-provider-backed lifecycle and actual local runtime binding remain open even though synthetic restart, mapping, denial, cancellation, and stop behavior are refreshed here.

## Addendum — real-provider failure receipt (2026-10-02)

The owner-approved one-shot request described in `02-08-EVIDENCE.md` emitted accepted and terminal receipts for the same task and Host; `task show` confirmed terminal `failed`. This is evidence that the current Host path durably reports the observed provider authentication failure. It does not establish successful real-provider lifecycle, answer persistence, restart recovery after a provider response, or general runtime binding. No second request was sent.

The focused Host/Hermes lifecycle test selection was rerun on the current working tree on 2026-10-02: `rtk go test ./internal/host ./internal/hermes -run 'Test.*(Run|Dispatch|Recover|Restart|Stream|Idempot|Approval|Cancel|Terminal|First|Binding)' -count=1` passed 42 tests. The first sandboxed invocation could not bind its ephemeral `127.0.0.1` test servers; the successful retry ran with local-loopback permission and made no external/provider request. This refreshes synthetic/contract evidence only; it does not upgrade the real-provider outcome.

## Verification refresh — 2026-10-02

The same focused lifecycle contract command was rerun against the current active working tree and passed all 42 selected tests in `internal/host` and `internal/hermes`. The tests use local contract/fault-injection servers; no NVIDIA request was made. Result remains synthetic/contract evidence and does not close NIM-profile restart, stream-loss, cancellation, or approval acceptance.

## In-flight NIM cancellation probe — 2026-10-02, unverified

A disposable pinned Hermes 0.21.1 container was configured with the owner-selected NIM model, an empty API tool inventory, and an isolated temporary Host. The runner exited 0, but its captured output was only the text `header`; it did not retain a task ID, Host ID, accepted receipt, or terminal status. The attempt may have dispatched the provider request, so it is not retried and is not counted as a lifecycle pass. Post-run inspection found no container bearing the probe label, no matching named volume, no listener on the probe port, and no matching scratch directory. This attempt does not close NIM cancellation or any other lifecycle gate.


## Addendum — NVIDIA NIM profile boundary (2026-10-02)

The owner-selected NVIDIA NIM profile completed one Host-mediated task and a separate explicit terminal-canary task without the canary side effect; the latter ran with empty Hermes API toolsets. These observations confirm a completed task mapping on the named profile and no observed effect for that request. They do not exercise restart, stream loss, cancellation, duplicate dispatch, or approval races on NIM. Those lifecycle cases remain demonstrated only by the pinned synthetic-provider profile above, so 02-10 remains BLOCKED for the declared real-provider local profile.

## Verification refresh — provider-free lifecycle contracts (2026-10-02)

On the current active working tree, `rtk proxy go test ./internal/host ./internal/hermes -run 'Test.*(Run|Dispatch|Recover|Restart|Stream|Idempot|Approval|Cancel|Terminal|First|Binding)' -count=1` passed both packages with local-loopback permission. The sandboxed first invocation failed because tests could not bind ephemeral `127.0.0.1` servers; the permitted retry ran only in-process/local contract tests and sent no provider request. This confirms implementation contracts only. It does not add NIM lifecycle evidence or recover the earlier uncorrelated cancellation attempt; 02-10 remains BLOCKED for the declared NIM profile.

## Current-tree pinned synthetic lifecycle journey — 2026-10-02

`rtk proxy mise run milestone1-macos-check` passed on the current active working tree. The command first passed `go test ./... -count=1` across all nine packages, then built the native macOS Host and pinned Hermes `0.21.1` from the verified archive and ran the isolated synthetic-provider journey under Compose project `lumen-macos-check-72635`. It observed: a durable completed task; denial persisted as `approval_denied` with no Run mapping; exact in-flight stop while provider remained held, with Host and Hermes both `cancelled`; gateway restart reconciliation of the same Run with one create; an SSE cut followed by abrupt Host restart, same Run reconciled to `completed` with one create and preserved Space/owner/Host identity, initialization marker, operator credential, and unchanged state at exit; duplicate/reordered SSE frames `2,1,2,3` including a conflicting duplicate ignored without duplicate authority transitions; and stop after the held provider barrier with one create. All native journey checkpoints passed.

Independent post-run Docker inspection found no container, network, or volume for `lumen-macos-check-72635`. The pre-existing `lumen-macos-check-85217` synthetic Hermes/provider pair remained running and untouched. No NIM request was made. This is strong current-tree pinned synthetic-runtime lifecycle evidence; it does not close the declared NIM profile's live restart/stream-loss/cancellation/approval matrix. The earlier NIM cancellation attempt remains unverified because it has no accepted receipt or task identity.

## Current full-suite refresh — 2026-10-02

After the Phase 02 Compose and NIM harness changes, `rtk proxy go test ./... -count=1` passed across all nine packages on the active tree. This refreshes the provider-free Go contract result. It does not supply a NIM held-run, restart, stream-loss, cancellation, exact-approval, or terminal-race observation; those profile-specific lifecycle gates remain BLOCKED.


## Lifecycle scope reconciliation — 2026-10-02

The approved Phase 02 lifecycle contract requires durable mapping, cancellation, stream loss, restart and exact approval behavior on the pinned Hermes runtime; it does not require every fault-injection case to use a live commercial provider. The current pinned synthetic-provider journey exercises those lifecycle and failure paths, while 02-08 separately records one successful Host-mediated NIM result and honest provider-failure receipts. These independent claims are not combined: NIM-specific restart/cancellation/approval behavior remains unqualified, but is not a Phase 02 exit criterion. No additional provider request was made.


## Current scoped disposition — 2026-10-02

The earlier BLOCKED-for-NIM headings are historical status snapshots and are superseded for Phase 02 exit by the approved provider-independent lifecycle scope in `02-10-PLAN.md`. The pinned Hermes synthetic-provider journey passed the required mapping, approval, denial, cancellation, stream-loss and restart cases. The selected NIM profile separately has one successful durable Host-backed result and truthful provider-failure evidence in 02-08. No NIM-specific restart/cancellation/approval behavior is claimed or required by the approved Phase 02 contract.
