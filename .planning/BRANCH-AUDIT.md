# Branch and PR audit — 2026-10-03

The three review units created or updated for this planning effort are separate and stacked: [#22](https://github.com/AshwanthReddy-exe/Lumen/pull/22) is the Phase 02 foundation draft against `main`; [#23](https://github.com/AshwanthReddy-exe/Lumen/pull/23) is the 17-phase plan draft against #22; [#24](https://github.com/AshwanthReddy-exe/Lumen/pull/24) is the focused encrypted store backup draft against #23. They are drafts because the listed independent/live gates remain open. Merge order is #22, #23, then #24 after review and rebase if GitHub changes the base relationship.

| Branch/worktree | Distinct work and audit result | PR action |
| --- | --- | --- |
| `feat/host-hermes-foundation-closure` | Local setup, pinned Hermes, NIM, Docker and macOS paths with bounded evidence; `phase2-check` and `phase0-check` passed. Real deployment/login/reboot/security gates remain. | Updated draft #22, independent review required. |
| `docs/lumen-gsd-master-plan` | All 17 phases, 58 execute plans and explicit conditional handoffs; syntax/ownership checks pass. Cross-AI convergence unavailable; physical/client/provider gates remain. | Draft #23, stacked on #22. |
| `fix/store-durable-backup` | Extracted only encrypted state hard-link, pre-replacement directory sync and focused tests from dirty Phase 03 worktree. Full Go suite passes. Physical power-loss and recovery-artifact retention remain open. | Draft #24, stacked on #23. |
| `feat/lumen-continuity-preserved` | 25 distinct commits; Go and Phase 0 suites pass, but historical live provider route has an HTTP 401 gate. Integration candidate on `feat/lumen-continuity-integration` resolves seven conflicts and passes those suites, yet its Hermes zero-tool patch fails to compose with the existing room-grant patch. Automatic approval review rejected security-sensitive patch edits. | No PR until both patch contracts apply against pinned source and are independently reviewed. Preserve both branches/worktrees. |
| `feat/phase03-foundations` | No unique commit; mixed uncommitted store, runtime inspect, old Phase 03 plans and mobile spike. E3/E5/E6 physical routes and E4 representative/power-loss evidence remain blocked or unverified. | No whole-branch PR; store subset extracted as #24. Preserve remaining work. |
| `docs/block-execution-plan`; `fix/termux-pie-build` | First has no unique commits; Termux fix is patch-equivalent to upstream. | No duplicate PR. |
| `feat/m2-conversation-nucleus`; `feat/lumen-continuity` | Strict predecessors of preserved continuity branch. | No duplicate PR. |
| `feat/m1-closure-finish`; `feat/host-execution-contract` | Old forks with unique work but destructive conflicts with current Go Host/planning tree. | No branch-wide PR. Extract only a demonstrably missing, tested behavior in its relevant phase. |

Do not stage `.gsd/`, `.planning/milestone.lock`, `.planning/debug/`, ignored `node_modules`/`dist`, or the unrelated dirty `AGENTS.md`/`.gitignore` as part of these PRs. PR creation and test results do not satisfy physical-device, provider, power-loss, seven-day soak or owner-signoff gates.
