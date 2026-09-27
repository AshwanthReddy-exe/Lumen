# Phase 01 registers

These are indexes, not duplicate sources of truth. Each item remains open until its destination phase records evidence and a decision.

## Decisions

Accepted and superseded decisions: [docs/DECISIONS.md](../../docs/DECISIONS.md). The seven-document ingest resolved one informational precedence item (D-059 explicit alpha memory) in [INGEST-CONFLICTS.md](../INGEST-CONFLICTS.md). New choices need a dated decision record, not an edit to a prior decision's meaning.

## Experiments

The full setup, measures and pass/fail gates for E1–E10 live in [the master plan](../../docs/PLAN.md#required-experiments).

| Gate | Decision it protects | Destination |
|---|---|---|
| E1–E2 | Hermes interaction contract and runtime containment | Phase 02 |
| E3–E6 | Mobile strategy, storage, connectivity and wake foundation | Phase 03 |
| E7 | Speech model and device resource choice | Phase 09 |
| E8 | Memory retrieval and privacy usefulness | Phase 05 |
| E9 | Side-effect uncertainty and node retry policy | Phase 08 |
| E10 | Migration fencing and old-Host retirement | Phase 10 |

## Risks

| Risk | Current status | Mitigation/evidence gate |
|---|---|---|
| Hermes approval/clarification API may not expose required interaction | Unverified | E1; disable unsupported path until supported extension exists |
| A Hermes profile digest may be mistaken for containment | Open | E2 process/tool/network enforcement and negative tests |
| Whole-state snapshots may not scale with conversations | Open | E4 atomicity, encryption, crash and latency comparison |
| Shared mobile stack may miss native audio/offline requirements | Open | E3 physical-device proof and agreed fallback |
| Old Host may retain authority after migration | Open | E10 explicit retirement/fail-closed process |
| Draft PR #20 work may be reported as current product | Controlled in docs, not merged | [Dated evidence ledger](BASELINE-EVIDENCE.md) and owner review |

Operational and security failure behavior remains in [the master plan](../../docs/PLAN.md#failure-behavior) and [threat model](../../docs/THREAT_MODEL.md).
