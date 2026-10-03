# API Coverage — pinned Hermes Runs surface

> Full coverage by default. Decisions apply to the pinned `v2026.9.7` source commit `2237be355906fbe6065ce1815711eee52b2d646e` and the recorded gateway `0.21.1`; capability advertisements and newer documentation do not certify this pin. Phase 02 probes may change a row only with evidence. The Host retains policy and durable task authority.

| capability | decision | reason |
| --- | --- | --- |
| GET /v1/capabilities authenticated discovery | INTEGRATE | |
| GET /health liveness | INTEGRATE | |
| GET /health/detailed verbose runtime diagnostics | OPT-OUT | Existing Host doctor derives readiness from bounded health, authenticated discovery, artifact and supervisor evidence; verbose runtime diagnostics are not needed for this gate and may expose details. |
| POST /v1/runs idempotent creation | INTEGRATE | |
| GET /v1/runs/{id} status reconciliation | INTEGRATE | |
| GET /v1/runs/{id}/events bounded SSE | INTEGRATE | |
| POST /v1/runs/{id}/approval exact once or deny | INTEGRATE | |
| POST /v1/runs/{id}/stop | INTEGRATE | |
| POST /v1/runs/{id}/steer optional transport method | INTEGRATE | |
| Ordinary clarification answer through steer | OPT-OUT | Queued steer is not an answer. Keep clarification disabled unless pinned E1 certifies a distinct path; Phase 07 owns an unsupported extension. |
| Session-wide or permanent runtime approval | OPT-OUT | D-02 and FR-41 allow only Host-authorized one-time action-bound approval; broader runtime grants would expand authority. |
| Runtime session or previous-response continuity as Space history | OPT-OUT | Host-owned canonical conversation is assigned to Phase 04; runtime session state cannot become Space authority. |
| Upstream run listing as the task authority | OPT-OUT | The Host's durable task-to-run mapping and status reconciliation are authoritative; an upstream listing cannot replace them. |
| Hermes built-in tools, MCP, browser, voice or delegation through this Run | OPT-OUT | These require separate typed Lumen capability, context and credential grants in their assigned later phases; FR-41 grants only agent.run/execute. |

The matrix enumerates the Runs interface used by `internal/hermes/client.go` and `internal/hermes/events.go`, plus deliberate product-level opt-outs. `INTEGRATE` states planned contract coverage, not a live-pass assertion. `02-03-EVIDENCE.md` must record the pinned result for each enabled operation, including the exact clarification finding. Development loopback and fake-server tests retain their own evidence class.
