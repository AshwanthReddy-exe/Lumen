# Future OS upstream delta — 2026-10-04

## Revisions inspected

- Lumen's checked-in reference snapshot: `98f7f3a3385e12d38ee7fc75bdca2cc3856cf987` (`2026-09-24`).
- Earlier broad Lumen analysis: `907f38b046b32ed3ac795c07b641e681d8e52101` (`2026-10-02` report metadata).
- Upstream `main` fetched read-only for this focused comparison: `52328e8009817c5eca66e4461ee4cf55e23fd6c9` (`2026-10-04`). `git ls-remote` and fetched `FETCH_HEAD` agreed. Only the files named below were compared in depth; this is not a full repository review or test/build certification.

## Newer patterns relevant to Lumen

| Future OS source at `52328e8` | Mechanism observed | Lumen use and boundary |
| --- | --- | --- |
| `mobile/src/remote/syncEngine.ts` | A serial lane owns both timeline cache and run cursors as one logical commit. A pushed stream is only a wake-up signal; durable replay/reconcile is the source for rebuilding. Reconnect, gap, incomplete prefix and truncation converge on reconciliation. | Strengthen Phase 05 sync plans with an atomic `(projection, cursor)` commit invariant, a single-writer per Space/run lane, replay watermark, and convergence tests from the same stale/corrupt starting states. Push notifications must not advance a cursor by themselves. Host remains the canonical record. |
| `mobile/src/remote/replay.ts` | Paged replay preserves a fixed replay boundary, checks cursor/index coverage, and distinguishes deliberately trimmed events from missing events. | Add explicit Phase 05/06 test vectors for empty page, `hasMore`, duplicate index, gap, trim metadata, stale watermark and payload bound. Never silently infer completeness from returned event count. |
| `agent/src/rpc/session_prompt.rs` | Runtime run queue/lease sequencing serializes accepted runs; completion worker owns releasing the active slot. Optional coalescing retains request identities and combines execution settings from the most recent request, while documenting an acceptance-loss edge if activation fails. | Phase 14 may borrow explicit queue/lease ownership and stale-worker fencing. Do **not** copy coalescing semantics: Lumen must not acknowledge multiple user intents and then discard them on dispatch failure. Each accepted intent needs its own durable receipt/outcome and policy binding. |
| `packages/rpc/src/command_policy.rs` | Each command has typed execution kind, timeout and retry policy; retryable mutation must reuse the exact request/idempotency identity. | Use as a contract-catalog checklist for Phase 07/11–14 RPCs: state mutation vs read, deadline, cancellation, retry class and idempotency key. Lumen chooses the values and Host authorization independently. |
| `packages/remote-crypto/src/lib.rs` | Versioned encrypted record framing has explicit size limits, replay window, monotonic sequence and nonce exhaustion behavior. | Phase 05 can use these as review prompts for transport framing and replay tests. Do not copy cryptographic code without a pinned-source/license review, protocol review, independent cryptography review, and Go/Swift fixture parity. |
| Current `SECURITY.md` | Tool permission defaults and platform guarantees remain configuration-specific; local-first storage does not mean prompts/tool results never leave the machine. | Keep Lumen default-deny and disclose provider egress at each projection. Future OS permission labels/sandbox defaults never become Lumen grants or security evidence. |

## Plan changes required from this delta

1. **Mapped to Phase 05-01 Task 2, 05-02 Task 3, 05-04 Task 2 and 05-05 Task 1**: freeze an explicit single-writer projection/cursor atomic-commit rule and replay integrity matrix before selected-client implementation, then bind its paths/checks to the D-056 decision. The reference cannot decide client stack or platform support.
2. Phase 14 plans may specify Host-owned lease epoch comparison at every transition and effect dispatch, but must retain every accepted schedule/child intent and receipts through crash, pause, revocation and ambiguous execution.
3. Every transport plan should define max record bytes, sequence exhaustion/rekey, replay-window behavior, duplicate handling, timeout and unknown outcome in Lumen's protocol fixture. These are review criteria, not a decision to import Future OS crypto.
4. No Future OS source code is copied by this analysis. If a future task copies code, freeze the exact upstream SHA, source path/lines, license and notices in that task; test Lumen authority and denial behavior independently.

## Pinned source links

- [Future OS sync engine at `52328e8`](https://github.com/futuregene/future-os/blob/52328e8009817c5eca66e4461ee4cf55e23fd6c9/mobile/src/remote/syncEngine.ts)
- [Future OS replay at `52328e8`](https://github.com/futuregene/future-os/blob/52328e8009817c5eca66e4461ee4cf55e23fd6c9/mobile/src/remote/replay.ts)
- [Future OS session prompt at `52328e8`](https://github.com/futuregene/future-os/blob/52328e8009817c5eca66e4461ee4cf55e23fd6c9/agent/src/rpc/session_prompt.rs)
- [Future OS command policy at `52328e8`](https://github.com/futuregene/future-os/blob/52328e8009817c5eca66e4461ee4cf55e23fd6c9/packages/rpc/src/command_policy.rs)
- [Future OS record crypto at `52328e8`](https://github.com/futuregene/future-os/blob/52328e8009817c5eca66e4461ee4cf55e23fd6c9/packages/remote-crypto/src/lib.rs)
- [Future OS security policy at `52328e8`](https://github.com/futuregene/future-os/blob/52328e8009817c5eca66e4461ee4cf55e23fd6c9/SECURITY.md)
