# Roadmap: Lumen

## Overview

The personal alpha builds and verifies useful local product behavior before any VPS or installer gate: Host/Hermes foundation, continuous Space conversation, accepted memory, node continuity, approvals/questions, cross-node execution, voice companion, integrated local use, then always-on deployment and alpha acceptance. Phases 11–17 retain their prior order and roadmap-level goals. This is the exact order and objective set in [docs/PLAN.md](../docs/PLAN.md).

The [execution blueprint](EXECUTION-BLUEPRINT.md) summarizes the sequence; 58 file-owned GSD plans for Phases 03–17 hold the current tasks. The [file inventory](FILE-INVENTORY.md) indexes their proposed tree. The [contract catalog](CONTRACT-CATALOG.md), [client branch gate](CLIENT-BRANCHES.md), [Phase 07–10 handoff](PHASE-07-10-HANDOFF.md), [later-phase contracts](PHASE-11-17-CONTRACTS.md), [platform coverage](PLATFORM-COVERAGE.md), and [completeness audit](PLANNING-COMPLETENESS-AUDIT.md) bind shared details and remaining gates. Decision-dependent paths are rewritten in the affected plan before execution.

## Phases

- [x] **Phase 01: Reconcile the repository and establish GSD authority** - Planning authority and implementation claims are traceable to current evidence.
- [x] **Phase 02: Local Host/Hermes product foundation** - A local Host qualifies its real Hermes runtime, durable execution lifecycle, and restricted profile before product use.
- [ ] **Phase 03: Continuous Space conversation** - Host-owned conversations and durable agent runs continue across web reconnects, Host/runtime restarts, and client changes with replayable receipts.
- [ ] **Phase 04: Accepted memory and context controls** - Owner-approved memory changes subsequent answers and deleted context stays absent.
- [ ] **Phase 05: Node identity and continuity** - An enrolled node can continue the same Space conversation through durable outbox, replay, receipts, and revocation.
- [ ] **Phase 06: Attention inbox and human input** - Authorized nodes synchronize action-bound approvals separately from clarification answers.
- [ ] **Phase 07: Cross-node capabilities and useful action** - A request in one node can invoke one bounded capability on another authorized node and return a durable receipt through Host policy.
- [ ] **Phase 08: Voice companion node** - A qualified node uses the shared conversation/action path with local wake and immediate mute/stop.
- [ ] **Phase 09: Integrated local product** - The personal-alpha features work together locally with failure/recovery and measured resource evidence.
- [ ] **Phase 10: Always-on deployment and alpha release** - Supported Host installation, update/recovery, backup/restore, node-offline continuity journey, soak and owner sign-off pass.
- [ ] **Phase 11: One messaging integration** - An explicitly linked messaging thread participates in the canonical Space conversation.
- [ ] **Phase 12: Apple abilities and controlled browsing** - Selected native and browser actions use exact grants and scoped credentials.
- [ ] **Phase 13: Controlled coding and broader device execution** - Remote coding produces reviewable work without changing the canonical project silently.
- [ ] **Phase 14: Continuous agents, automation and delegation** - Approved repeated work and delegated children stay within inspectable authority.
- [ ] **Phase 15: Paid self-hosted readiness** - A new owner can install, operate, update and recover the paid self-hosted Space.
- [ ] **Phase 16: Managed service** - A customer can use a dedicated managed Space with recoverable and isolated data.
- [ ] **Phase 17: Public integration ecosystem and shared Spaces** - Independent integrations can conform without expanding authority; shared membership has explicit rules.

## Phase Details

### Phase 01: Reconcile the repository and establish GSD authority

**Goal**: Planning authority and implementation claims are traceable to current evidence.
**Depends on**: Nothing (first phase)
**Requirements**: No FR requirement; Phase 01 is the documented planning/evidence gate.
**Success Criteria** (what must be TRUE):

  1. Planning files map every FR requirement to one owner and phase; repository claims distinguish merged work from draft code.
  2. Navigation points to active GSD and canonical docs, and removed Superpowers files are no longer active targets.

**Plans**: 01-01 evidence map; 01-02 canonical docs; 01-03 GSD authority and owner sign-off — completed 2026-09-27.

### Phase 02: Local Host/Hermes product foundation

**Goal**: Local Host-mediated real Hermes work is authorized, durable, recoverable and constrained before product data flows through it.
**Depends on**: Phase 01
**Requirements**: FR-01, FR-06, FR-38, FR-41, FR-69
**Success Criteria** (what must be TRUE):

  1. A real authorized Host-mediated Hermes answer is observed on the named local profile; credentials stay redacted and synthetic evidence stays labeled.
  2. Run mapping, approval, cancellation, stream loss and restart produce one durable truthful outcome with no duplicate dispatch.
  3. The restricted profile denies Host-file/secret and unapproved-network access at the tested runtime boundary. Chat-only tool safety is established by a zero-tool runtime inventory plus an explicit no-effect action canary; event-level denial of a model-originated tool call is UNSUPPORTED until Phase 07 introduces the Host capability broker. If Lumen-owned memory does not yet exist, mark that boundary UNSUPPORTED and retain its negative gate under Phase 04; do not imply it passed.

**Plans**: 02-08 provider/runtime qualification; 02-09 runtime/profile binding and chat-only configuration; 02-10 durable Run lifecycle; 02-11 restricted-profile negative tests; 02-12 independent review and phase exit. Completed 02-06 history remains retained.
**Superseded**: Unfinished 02-01 through 02-05 and 02-07 routes are marked with GSD `status: superseded`; their evidence remains. Local runtime/lifecycle/containment work is replaced by 02-08 through 02-12. Setup, reboot, update/rollback, external Runs and release closeout are owned by Phase 10 plans.
**Gate**: Phase 01 work remains the approved planning baseline; its verification digest is refreshed after checking the revised requirements ownership and 17-phase order. Existing setup/deployment code remains available but is no Phase 02 product exit gate.

### Phase 03: Conversation and web

**Goal**: A Host-owned Space conversation supports continuous agent runs across clients and restarts, with a responsive web surface and durable run state independent of Hermes sessions.
**Depends on**: Phase 02
**Requirements**: FR-04, FR-30, FR-60, FR-75
**Success Criteria** (what must be TRUE):

  1. A real conversation and its user/assistant messages survive separate Host and Hermes restarts.
  2. Duplicate commands return one receipt; SSE reconnect replays committed events from a durable cursor.
  3. Authenticated clients can observe, reconnect to, and stop an in-flight run without owning its lifecycle; run state distinguishes accepted, running, waiting-for-input, completed, failed, cancelled and uncertain.
  4. A continuous run preserves its Space conversation binding, context revision, runtime profile, checkpoints and event cursor across client disconnect and Host/runtime restart; ambiguous dispatch is reconciled without blind resubmission.
  5. Long-running work is bounded by explicit time/resource limits and remains inspectable and stoppable; no background or cross-node action can exceed Host-issued authority.

**Plans**: 03-01 through 03-05 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/03-conversation-and-web/03-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).

### Phase 04: Accepted memory and context controls

**Goal**: The owner controls retained knowledge and every context projection into a turn.
**Depends on**: Phase 03
**Requirements**: FR-33, FR-36, FR-61, FR-63
**Success Criteria** (what must be TRUE):

  1. Accepted records and proposals carry provenance, scope, classification, retention and revision.
  2. Inspect, accept, correct, delete and export controls affect the next runtime projection.
  3. Each run binds a context revision; memory edits have an explicit next-turn/current-run effect, while deletion or scope revocation prevents further projection into active work.
  4. Deleted/disallowed records are absent after runtime restart and session replacement; retrieval claims use independent questions and privacy canaries.

**Plans**: 04-01 through 04-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/04-accepted-memory-and-context-controls/04-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 05: Node identity and continuity

**Goal**: A node securely joins a Space and can continue a conversation while connected to its authorized Host, including reconnect after suspension or temporary network loss.
**Depends on**: Phases 03–04
**Requirements**: FR-02, FR-05, FR-10, FR-15, FR-23, FR-24, FR-66, FR-84
**Success Criteria** (what must be TRUE):

  1. The selected node client passes the bounded platform gate for secure key storage, pairing, replay, approvals and required native capabilities. Platform names describe tested implementations, not Lumen identity types.
  2. Owner-confirmed enrollment, rotation/revocation and scoped node status use versioned signed, expiring messages.
  3. A node outbox, durable receipts and cursor replay survive suspension, lost acknowledgements, clock skew and network changes without duplicate Space messages.
  4. A client can reattach to an existing continuous run using its stable Space/run identity and recover committed events from a fixed watermark.
**Plans**: 05-01 through 05-05 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/05-phone-identity-and-chat/05-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 06: Attention inbox and human input

**Goal**: Phone and web deliver exact action approvals and ordinary clarification inputs with one synchronized accepted resolution.
**Depends on**: Phases 03 and 05; runtime capability confirmation from Phase 02
**Requirements**: FR-31, FR-81, FR-82
**Success Criteria** (what must be TRUE):

  1. Approval binds exact action/arguments/actor/node/expiry and is consumed once.
  2. Clarification answer supplies input but grants no authority.
  3. Concurrent resolutions choose one winner; changed actions, expiry and cancellation cannot reuse a decision.
  4. A run waiting for an answer or approval is durably paused, resumes from the same run after authorized input, and expires/cancels without issuing a stale effect.

**Plans**: 06-01 through 06-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/06-attention-inbox-and-human-input/06-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 07: Cross-node capabilities and useful action

**Goal**: A request from one authorized node can cause one approved, bounded capability on a different enrolled node, with Host-owned routing and a verifiable Space receipt.
**Depends on**: Phases 05–06
**Requirements**: FR-11, FR-12, FR-13, FR-14, FR-20, FR-21, FR-22, FR-35, FR-64, FR-65
**Success Criteria** (what must be TRUE):

  1. A target node is enrolled and advertises a versioned capability manifest; each capability rechecks local OS permission and selected-resource scope at execution.
  2. Host validates actor, target node, exact capability/arguments, grant and approval, then persists and dispatches one typed invocation; Hermes cannot select targets, grant authority, or bypass node checks.
  3. Source and target nodes observe the same durable invocation state and final receipt; disconnect before/after an effect yields a recoverable result or explicit uncertainty without blind duplicate effect.
  4. Revocation, stale manifests, replayed commands, duplicate dispatch, node restart, and target substitution are denied or reconciled safely.

**Plans**: 07-01 through 07-03 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/07-mac-node-and-useful-action/07-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 08: Voice companion node

**Goal**: A qualified voice-capable node uses the same Space conversation, approvals and cross-node capability path with local wake and speech controls.
**Depends on**: Phases 03–07
**Requirements**: FR-07, FR-40, FR-71, FR-72, FR-73, FR-83
**Success Criteria** (what must be TRUE):

  1. Physical Android wake detection meets E6's intended-detection/false-activation thresholds or supported conditions are narrowed.
  2. Visible mute/listening/stop controls prevent pre-activation audio egress and promptly interrupt local playback.
  3. Local recognition and speech are reported honestly; unsupported Hermes reply paths remain disabled.

**Plans**: 08-01 through 08-02 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/08-desk-voice/08-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 09: Integrated local product

**Goal**: All personal-alpha capabilities work together locally with honest failures, accessibility states, and reproducible resource/latency evidence.
**Depends on**: Phases 02–08
**Requirements**: Cross-phase integration gate; adds no new FR owner. It exercises Phases 03–08 locally and excludes primary-node-offline and restore claims.
**Success Criteria** (what must be TRUE):

  1. Real Hermes answer, accepted memory correction, source-node message, approval and target-node receipt complete in one local Space; an interrupted continuous run can be reattached and reconciled.
  2. Provider failure, Host/runtime restart, client suspension, node loss, storage-full and accessibility states are visible and recover safely.
  3. Latency/resource results name revision, hardware, model and workload; no VPS/primary-node-offline claim is made.

**Plans**: 09-01 through 09-02 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/09-integrated-local-product/09-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 10: Always-on deployment and alpha release

**Goal**: A declared always-on Host configuration installs, updates, recovers and supports the complete primary-node-offline alpha journey with measured operational evidence.
**Depends on**: Phases 02–09
**Requirements**: FR-03, FR-08, FR-09, FR-34, FR-62
**Success Criteria** (what must be TRUE):

  1. Supported VPS setup, anonymous pinned-image pull, supervisor reboot, truthful doctor, executable update/rollback and interrupted replacement recovery pass on named machines.
  2. Any supported macOS auto-start profile proves canonical LaunchAgent publication, recoverable interruption, and real login/reboot behavior; otherwise macOS auto-start is explicitly unsupported.
  3. External Hermes Runs across two machines and the declared containment profile pass, or unsupported profile limits are explicit.
  4. Encrypted export/restore and old-Host retirement pass; one enrolled node continues the same conversation/action journey while the source node is offline.
  5. Measured latency and seven-day soak meet declared limits and owner sign-off records the exact supported configuration.

**Plans**: 10-01 through 10-05 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/10-always-on-deployment-and-alpha-release/10-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).

### Phase 11: One messaging integration

**Goal**: An explicitly linked messaging thread participates in the canonical Space conversation.
**Depends on**: Phase 10
**Requirements**: FR-70
**Success Criteria** (what must be TRUE):

  1. Owner can continue one selected conversation through the app and messaging provider.
  2. Duplicate or failed provider delivery yields an honest receipt without merging unrelated chats.

**Plans**: 11-01 through 11-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/11-messaging-integration/11-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 12: Apple abilities and controlled browsing

**Goal**: Selected native and browser actions use exact grants and scoped credentials.
**Depends on**: Phases 08 and 10
**Requirements**: FR-37, FR-42
**Success Criteria** (what must be TRUE):

  1. Owner can create an approved reminder and inspect its external item reference.
  2. Browser work cannot use personal credentials or unrelated device resources without consent.

**Plans**: 12-01 through 12-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/12-apple-and-browsing/12-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 13: Controlled coding and broader device execution

**Goal**: Remote coding produces reviewable work without changing the canonical project silently.
**Depends on**: Phases 08 and 10; tool qualification where used
**Requirements**: FR-39
**Success Criteria** (what must be TRUE):

  1. A bounded coding run can continue across client disconnect, checkpoint its work, and target only an explicitly selected node/workspace.
  2. Owner can inspect a patch, test evidence and proposed application before accepting changes.
  3. Failed, cancelled, expired or uncertain coding leaves the canonical project unmodified and reports its outcome.

**Plans**: 13-01 through 13-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/13-coding-and-device-execution/13-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).

### Phase 14: Continuous agents, automation and delegation

**Goal**: Approved repeated work and delegated children stay within inspectable authority.
**Depends on**: Phase 10 and a stable effectful capability
**Requirements**: FR-32, FR-67, FR-68
**Success Criteria** (what must be TRUE):

  1. Owner can preview, approve, pause and revoke a narrow recurring workflow or continuous agent goal.
  2. Runs persist checkpoints, lease/heartbeat state, budgets, deadlines and child-run links across restart and client changes; inactive or orphaned leases cannot execute work.
  3. Cross-node child work gets only an explicitly delegated, bounded grant and records its target node, approval, evidence and receipt.
  4. A duplicate trigger, restart, cancellation, revocation or child failure does not broaden grants, duplicate effects or disappear from the parent run.

**Plans**: 14-01 through 14-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/14-automation-and-delegation/14-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).

### Phase 15: Paid self-hosted readiness

**Goal**: A new owner can install, operate, update and recover the paid self-hosted Space.
**Depends on**: Phase 10 and selected beta cohorts
**Requirements**: FR-74, FR-77, FR-79, FR-80
**Success Criteria** (what must be TRUE):

  1. A new owner can complete installation, update or rollback and recover from an encrypted export.
  2. Entitlement expiry leaves export usable; support diagnostics disclose no secrets.

**Plans**: 15-01 through 15-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/15-paid-self-hosted/15-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).
**UI hint**: yes

### Phase 16: Managed service

**Goal**: A customer can use a dedicated managed Space with recoverable and isolated data.
**Depends on**: Phase 15
**Requirements**: FR-76
**Success Criteria** (what must be TRUE):

  1. A managed customer can use a dedicated data plane and request deletion or restore.
  2. Cross-customer isolation and recovery are demonstrated, with provider trust disclosed.

**Plans**: 16-01 through 16-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/16-managed-service/16-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).

### Phase 17: Public integration ecosystem and shared Spaces

**Goal**: Independent integrations can conform without expanding authority; shared membership has explicit rules.
**Depends on**: Stable beta contracts and operational evidence
**Requirements**: FR-78
**Success Criteria** (what must be TRUE):

  1. An independent node passes public conformance fixtures without proprietary source.
  2. An extension cannot gain undeclared authority, and shared members see only their permitted Space data.

**Plans**: 17-01 through 17-04 are file-owned GSD plans with contract, implementation, failure and evidence gates. See [the first plan](phases/17-public-integrations-and-shared-spaces/17-01-PLAN.md) and [the contract catalog](CONTRACT-CATALOG.md).

## Progress

| Phase | Plans Complete | Status | Completed |
|---|---|---|---|
| 01. Reconcile the repository and establish GSD authority | 3/3 | Complete | 2026-09-27 |
| 02. Local Host/Hermes product foundation | 6/6 (02-06 retained plus 02-08–12) | Complete | 2026-10-02 |
| 03. Continuous Space conversation | 0/5 | Not started | - |
| 04. Accepted memory and context controls | 0/4 | Not started | - |
| 05. Node identity and continuity | 0/5 | Not started | - |
| 06. Attention inbox and human input | 0/4 | Not started | - |
| 07. Cross-node capabilities and useful action | 0/3 | Not started | - |
| 08. Voice companion node | 0/2 | Not started | - |
| 09. Integrated local product | 0/2 | Not started | - |
| 10. Always-on deployment and alpha release | 0/5 | Not started | - |
| 11. One messaging integration | 0/4 | Not started | - |
| 12. Apple abilities and controlled browsing | 0/4 | Not started | - |
| 13. Controlled coding and broader device execution | 0/4 | Not started | - |
| 14. Durable automation and delegation | 0/4 | Not started | - |
| 15. Paid self-hosted readiness | 0/4 | Not started | - |
| 16. Managed service | 0/4 | Not started | - |
| 17. Public integration ecosystem and shared Spaces | 0/4 | Not started | - |
