# Roadmap: Lumen

## Overview

The personal alpha follows the existing plan's Phases 01–10: reconcile evidence, close the Host/Hermes foundation, qualify platform choices, then deliver canonical conversation, memory, phone continuity, approvals, Mac action, desk voice and recovery. Phases 11–17 extend messaging, abilities, automation and commercial delivery. This is the exact order and objective set in [docs/PLAN.md](../docs/PLAN.md).

## Phases

- [x] **Phase 01: Reconcile the repository and establish GSD authority** - Planning authority and implementation claims are traceable to current evidence.
- [ ] **Phase 02: Close the existing Host/Hermes foundation** - The existing Host and pinned Hermes runtime behave safely through setup, interruption and recovery.
- [ ] **Phase 03: Validate mobile, storage, network and wake foundations** - The alpha platform choices are supported by reproducible physical and operational evidence.
- [ ] **Phase 04: Canonical conversations and product API** - A Host-owned conversation survives runtime and Host changes and is available on the web.
- [ ] **Phase 05: Accepted memory and context controls** - The owner controls what Lumen remembers and what each destination receives.
- [ ] **Phase 06: Pairing, client identity and reliable synchronization** - A paired phone can continue the same conversation while the laptop is off.
- [ ] **Phase 07: Attention inbox and cross-device human input** - The owner can answer exact approvals and ordinary questions from any authorized surface.
- [ ] **Phase 08: Mac execution node and useful cross-device action** - A phone request can safely produce one bounded, verifiable Mac action.
- [ ] **Phase 09: Speech and Android desk companion** - A local desk voice surface can use the same conversation and action path.
- [ ] **Phase 10: Personal alpha acceptance and recovery** - The full personal-alpha journey remains trustworthy through sustained use and recovery.
- [ ] **Phase 11: One messaging integration** - An explicitly linked messaging thread participates in the canonical Space conversation.
- [ ] **Phase 12: Apple abilities and controlled browsing** - Selected native and browser actions use exact grants and scoped credentials.
- [ ] **Phase 13: Controlled coding and broader device execution** - Remote coding produces reviewable work without changing the canonical project silently.
- [ ] **Phase 14: Durable automation and delegation** - Approved repeated work and delegated children stay within inspectable authority.
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

### Phase 02: Close the existing Host/Hermes foundation

**Goal**: The existing Host and pinned Hermes runtime behave safely through setup, interruption and recovery.
**Depends on**: Phase 01
**Requirements**: FR-01, FR-06, FR-08, FR-09, FR-38, FR-41
**Success Criteria** (what must be TRUE):

  1. Owner can install or adopt the pinned Hermes runtime and see honest readiness on a supported machine.
  2. Denied approval, cancellation, ambiguous dispatch and restart leave one durable, truthful outcome.

**Plans**: TBD

- [ ] 02-01-PLAN.md
- [ ] 02-02-PLAN.md
- [ ] 02-03-PLAN.md
- [ ] 02-04-PLAN.md
- [ ] 02-05-PLAN.md
- [x] 02-06-PLAN.md
- [ ] 02-07-PLAN.md

**Gate**: Phase 01 artifact set was approved and PR #21 merged into `main` as `81b60e3` on 2026-09-27. Phase 02 starts from that updated base.

### Phase 03: Validate mobile, storage, network and wake foundations

**Goal**: The alpha platform choices are supported by reproducible physical and operational evidence.
**Depends on**: Phase 01; runtime tests depend on Phase 02
**Requirements**: FR-69, FR-84
**Success Criteria** (what must be TRUE):

  1. Owner can review pass or fail evidence for mobile, storage, network and local wake experiments.
  2. A documented platform choice and fallback follow each experiment, including physical phone evidence.

**Plans**: TBD

### Phase 04: Canonical conversations and product API

**Goal**: A Host-owned conversation survives runtime and Host changes and is available on the web.
**Depends on**: Phases 02–03
**Requirements**: FR-04, FR-30, FR-60, FR-75
**Success Criteria** (what must be TRUE):

  1. Owner can send a web message and see it in the same canonical conversation after Host or Hermes restart.
  2. Retried commands produce one receipt, and a client can replay missed events without losing accepted messages.

**Plans**: TBD
**UI hint**: yes

### Phase 05: Accepted memory and context controls

**Goal**: The owner controls what Lumen remembers and what each destination receives.
**Depends on**: Phase 04
**Requirements**: FR-33, FR-36, FR-61, FR-63
**Success Criteria** (what must be TRUE):

  1. Owner can inspect, accept, correct and delete a memory with its provenance and scope.
  2. Subsequent turns reflect corrections and exclude deleted or disallowed context even after runtime replacement.

**Plans**: TBD
**UI hint**: yes

### Phase 06: Pairing, client identity and reliable synchronization

**Goal**: A paired phone can continue the same conversation while the laptop is off.
**Depends on**: Phase 04; Phase 03 mobile/network decision
**Requirements**: FR-02, FR-05, FR-10, FR-15, FR-23, FR-24, FR-66
**Success Criteria** (what must be TRUE):

  1. Owner can pair and revoke a phone and inspect its identity, health and synchronized state.
  2. Phone sends and replays the same conversation without duplicate messages; revocation blocks new Host access.

**Plans**: TBD
**UI hint**: yes

### Phase 07: Attention inbox and cross-device human input

**Goal**: The owner can answer exact approvals and ordinary questions from any authorized surface.
**Depends on**: Phases 02, 04 and 06
**Requirements**: FR-31, FR-81, FR-82
**Success Criteria** (what must be TRUE):

  1. An approval or clarification appears on phone and web within its conversation and attention inbox.
  2. One resolution wins across concurrent surfaces; expiry and changed actions cannot reuse authority.

**Plans**: TBD
**UI hint**: yes

### Phase 08: Mac execution node and useful cross-device action

**Goal**: A phone request can safely produce one bounded, verifiable Mac action.
**Depends on**: Phases 06–07
**Requirements**: FR-11, FR-12, FR-13, FR-14, FR-20, FR-21, FR-22, FR-35, FR-64, FR-65
**Success Criteria** (what must be TRUE):

  1. Owner can select a Mac capability, approve it on the phone and see a bounded result in the same conversation.
  2. The Mac enforces local permissions and resource scope; offline or uncertain execution is shown honestly.

**Plans**: TBD
**UI hint**: yes

### Phase 09: Speech and Android desk companion

**Goal**: A local desk voice surface can use the same conversation and action path.
**Depends on**: Phase 03 wake experiment; Phases 04, 06–08
**Requirements**: FR-07, FR-40, FR-71, FR-72, FR-73, FR-83
**Success Criteria** (what must be TRUE):

  1. Owner sees clear listening and mute states, with no audio leaving before activation.
  2. After activation, speech can initiate the phone-to-Mac flow and interruption promptly stops playback.

**Plans**: TBD
**UI hint**: yes

### Phase 10: Personal alpha acceptance and recovery

**Goal**: The full personal-alpha journey remains trustworthy through sustained use and recovery.
**Depends on**: Phases 02–09
**Requirements**: FR-03, FR-34, FR-62
**Success Criteria** (what must be TRUE):

  1. Owner starts on laptop, continues on phone with laptop off, then approves a phone-to-Mac action and sees its receipt in one conversation.
  2. Backup, restore and failure drills preserve canonical truth; declared latency and seven-day soak results are measured.

**Plans**: TBD

### Phase 11: One messaging integration

**Goal**: An explicitly linked messaging thread participates in the canonical Space conversation.
**Depends on**: Phase 10
**Requirements**: FR-70
**Success Criteria** (what must be TRUE):

  1. Owner can continue one selected conversation through the app and messaging provider.
  2. Duplicate or failed provider delivery yields an honest receipt without merging unrelated chats.

**Plans**: TBD
**UI hint**: yes

### Phase 12: Apple abilities and controlled browsing

**Goal**: Selected native and browser actions use exact grants and scoped credentials.
**Depends on**: Phases 08 and 10
**Requirements**: FR-37, FR-42
**Success Criteria** (what must be TRUE):

  1. Owner can create an approved reminder and inspect its external item reference.
  2. Browser work cannot use personal credentials or unrelated device resources without consent.

**Plans**: TBD
**UI hint**: yes

### Phase 13: Controlled coding and broader device execution

**Goal**: Remote coding produces reviewable work without changing the canonical project silently.
**Depends on**: Phases 08 and 10; tool qualification where used
**Requirements**: FR-39
**Success Criteria** (what must be TRUE):

  1. Owner can inspect a patch, test evidence and proposed application before accepting changes.
  2. Failed or cancelled coding leaves the canonical project unmodified and reports its outcome.

**Plans**: TBD

### Phase 14: Durable automation and delegation

**Goal**: Approved repeated work and delegated children stay within inspectable authority.
**Depends on**: Phase 10 and a stable effectful capability
**Requirements**: FR-32, FR-67, FR-68
**Success Criteria** (what must be TRUE):

  1. Owner can preview, approve, pause and revoke a narrow recurring workflow.
  2. A restart or duplicate trigger does not broaden grants or hide a child failure.

**Plans**: TBD

### Phase 15: Paid self-hosted readiness

**Goal**: A new owner can install, operate, update and recover the paid self-hosted Space.
**Depends on**: Phase 10 and selected beta cohorts
**Requirements**: FR-74, FR-77, FR-79, FR-80
**Success Criteria** (what must be TRUE):

  1. A new owner can complete installation, update or rollback and recover from an encrypted export.
  2. Entitlement expiry leaves export usable; support diagnostics disclose no secrets.

**Plans**: TBD
**UI hint**: yes

### Phase 16: Managed service

**Goal**: A customer can use a dedicated managed Space with recoverable and isolated data.
**Depends on**: Phase 15
**Requirements**: FR-76
**Success Criteria** (what must be TRUE):

  1. A managed customer can use a dedicated data plane and request deletion or restore.
  2. Cross-customer isolation and recovery are demonstrated, with provider trust disclosed.

**Plans**: TBD

### Phase 17: Public integration ecosystem and shared Spaces

**Goal**: Independent integrations can conform without expanding authority; shared membership has explicit rules.
**Depends on**: Stable beta contracts and operational evidence
**Requirements**: FR-78
**Success Criteria** (what must be TRUE):

  1. An independent node passes public conformance fixtures without proprietary source.
  2. An extension cannot gain undeclared authority, and shared members see only their permitted Space data.

**Plans**: TBD

## Progress

| Phase | Plans Complete | Status | Completed |
|---|---|---|---|
| 01. Reconcile the repository and establish GSD authority | 3/3 | Complete | 2026-09-27 |
| 02. Close the existing Host/Hermes foundation | 1/7 | In Progress | - |
| 03. Validate mobile, storage, network and wake foundations | 0/TBD | Not started | - |
| 04. Canonical conversations and product API | 0/TBD | Not started | - |
| 05. Accepted memory and context controls | 0/TBD | Not started | - |
| 06. Pairing, client identity and reliable synchronization | 0/TBD | Not started | - |
| 07. Attention inbox and cross-device human input | 0/TBD | Not started | - |
| 08. Mac execution node and useful cross-device action | 0/TBD | Not started | - |
| 09. Speech and Android desk companion | 0/TBD | Not started | - |
| 10. Personal alpha acceptance and recovery | 0/TBD | Not started | - |
| 11. One messaging integration | 0/TBD | Not started | - |
| 12. Apple abilities and controlled browsing | 0/TBD | Not started | - |
| 13. Controlled coding and broader device execution | 0/TBD | Not started | - |
| 14. Durable automation and delegation | 0/TBD | Not started | - |
| 15. Paid self-hosted readiness | 0/TBD | Not started | - |
| 16. Managed service | 0/TBD | Not started | - |
| 17. Public integration ecosystem and shared Spaces | 0/TBD | Not started | - |
