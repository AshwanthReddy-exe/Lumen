# Phase 02 plan 01 evidence — in progress

Observed on 2026-09-27, macOS arm64, Go 1.27.1, Docker daemon 29.7.2. Lumen base revision `81b60e33ef006e33f0e63c882052c2a17efcacea`; pinned Hermes source tag `v2026.9.7`, commit `2237be355906fbe6065ce1815711eee52b2d646e`, archive SHA-256 `907c2a72db1c5dd637ea8eeae97f4cb5b32cef615c17258f6b190924ec5bf688`.

| Check | Result | Evidence and limit |
| --- | --- | --- |
| Go baseline | Pass | `rtk go test ./...` outside restricted socket sandbox: 463 tests in nine packages. In-sandbox bind failures were environmental. |
| Native Host ↔ pinned Hermes gateway | Pass for synthetic-provider profile | `rtk mise run milestone1-macos-check` exited 0. Existing journey built the pinned gateway, started a native Host, completed an `agent.run/execute` task with a synthetic marker, rejected a denied approval with durable `approval_denied`, and durably recorded cancellation. This does **not** prove a real provider reply or runtime containment. |
| Full automated Phase 02 baseline | Pass | `rtk mise run phase2-check` exited 0 after correcting Compose syntax validation to avoid requiring deployment secrets. Includes Go tests/race, cross-builds and Android unit/build checks. This is not a physical Android acceptance test. |
| Interrupted setup and create-once identity after live restart | Unverified | The native journey above does not perform the exact interrupted-stage rerun and before/after identity digest comparison required by plan 01 task 2. |
| Real provider through Lumen | Unverified | The provider in this journey is synthetic. PR #20 records an HTTP 401 in its separate later-scope real-chat path. |
| Same-machine external compatibility journey on this Mac | Not applicable | `milestone1-external-check` is Linux-only. It previously failed on macOS at `chmod --` before reaching its OS guard; the guard is now first and returns the explicit Linux-only error. This is a probe usability fix, not evidence of external Runs or a two-machine setup. |

No Space keys, bearer tokens, prompt content or private runtime state are included here. The successful checks do not close Phase 02 by themselves.
