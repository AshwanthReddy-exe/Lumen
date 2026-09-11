# Milestone 1 dual topology foundation design

## Status

Approved by the owner on 2026-09-11. This design refines Milestone 1 in [PLAN.md](../../PLAN.md) so Lumen supports both a combined deployment and an independently managed external Hermes deployment. It preserves the approved milestone lifecycle in [the milestone-driven development design](./2026-09-11-milestone-driven-development-design.md).

## Purpose

Milestone 1 must establish a reliable Lumen Space foundation without making Hermes deployment ownership part of the Space contract. A user may install Lumen and Hermes together or connect a Lumen Host to an existing Hermes service on a VPS. Both paths expose the same Host-owned task behavior and differ only in artifact ownership, supervision, and connection assurance.

Milestone 0 remains complete at its intended Space authority and encrypted-store boundary. Its implementation is preserved. The only Milestone 0 correction in this work is to replace stale historical validation instructions with the current Go authority checks.

## Goals

1. Provide one public setup journey with explicit `combined` and `external` topology choices.
2. Install, update, roll back, and supervise Hermes only in `combined` topology.
3. Connect to an existing independently managed Hermes without modifying or supervising it in `external` topology.
4. Allow Hermes to perform authorized Space work through a narrow Host-mediated task boundary.
5. Produce truthful setup, doctor, service, recovery, and compatibility outcomes for both topologies.
6. Qualify both topologies before Milestone 1 closes.

## Non goals

- Lumen does not SSH into, provision, upgrade, restart, or inspect the filesystem of an external Hermes machine.
- Hermes never reads the encrypted Space store or the owner-restricted Host control socket.
- Milestone 1 does not add canonical conversations, memory projection, node transport, messaging, voice, or general-purpose Space queries. Those remain in later milestones.
- Setup does not introduce a second runner, installer framework, state store, runtime adapter, or supervisor abstraction.

## Governing invariants

- One active Host owns canonical Space identity, grants, approvals, task lifecycle, durable outcomes, and audit.
- Hermes is untrusted execution infrastructure behind the existing versioned Runs adapter.
- A topology choice changes deployment ownership, not authority or product behavior.
- Assurance profile and topology are independent. `development`, `personal-alpha`, and `hardened` cannot silently imply `combined` or `external`.
- Setup reruns preserve initialized Space identity and credentials. A changed topology, profile, endpoint identity, or plan digest fails before mutation.
- `ready`, `degraded`, and `action_required` are the only public setup and doctor outcomes.
- No secret, raw Hermes response, private prompt, or unrestricted Space context appears in ordinary output, logs, journals, or evidence.

## Deployment topologies

### Combined

Lumen owns the selected Lumen and Hermes release artifacts on one supported Linux or VPS deployment. Setup verifies both artifacts, creates distinct credentials and configuration, initializes the Host exactly once, installs both supervisor definitions, starts both services, and performs authenticated compatibility and Runs probes. Service commands control both services in dependency order. Failed replacement restores the last verified artifacts and canonical state.

### External

Lumen owns only the local Host artifact and service. The user supplies an existing Hermes endpoint and credential-file references. Setup validates endpoint syntax, origin, transport identity, authentication, required capabilities, and the supported Runs behavior before recording adoption. It never writes a Hermes configuration, installs a Hermes artifact, or controls the remote service. Service commands control only the Host. A later remote outage yields `degraded` when the initialized Host remains safe and usable; identity, credential, compatibility, or incomplete-setup failures yield `action_required`.

## Setup contracts

### Topology model

Replace the ambiguous `HermesAdopted` boolean with an explicit `Topology` value containing only `combined` or `external`. Persist topology in the plan result, setup request, public report, generated Lumen configuration, and setup-journal binding.

The journal binding covers profile, platform, architecture, topology, selected artifact identities, endpoint origin, endpoint identity digest, and plan digest. It stores digests or paths to credential sources, never credential contents. A rerun with a different binding returns `action_required` before any stage executes.

### Inputs

Combined setup requires a release manifest containing immutable Lumen and Hermes artifacts for the selected platform, architecture, profile, and compatibility contract.

External setup requires an HTTPS endpoint outside the development profile, a bearer credential file, and the identity material required by the selected assurance profile. Hardened external setup requires normal CA, hostname, and validity verification, an exact pinned leaf identity, and client authentication. Plain loopback HTTP remains development-only and cannot satisfy either release gate.

### Stage ownership

One `SetupRunner` owns the stage sequence and journal writes. Stage callbacks perform effects or observations but do not independently claim journal completion. The sequence is:

`detected -> artifacts_ready -> directories_ready -> credentials_ready -> configuration_ready -> host_initialized -> services_installed -> services_started -> validated`

For external topology, Hermes artifact and supervisor effects are explicit no-ops backed by recorded external-adoption evidence; they are never simulated as local installation.

## Host and Hermes access boundary

The Host initiates authenticated Hermes Runs. Each dispatch is bound to a durable Host task and contains only:

- the stable task and idempotency identities;
- the authorized capability and target;
- a runtime-profile digest;
- a deadline and cancellation lineage;
- the minimum bounded context required for that task.

Hermes returns untrusted progress, tool requests, approval requests, artifacts, usage, terminal evidence, and results. The Host validates every state transition and persists canonical outcomes before acknowledgement. Hermes may request a capability only through the Host's typed policy path. It cannot create grants, select an unauthorized target, approve itself, query unrelated Space data, or write canonical state.

Milestone 1 exposes no general Space-reading API to Hermes. Conversation and memory projections will extend this same task envelope in Milestone 2 with provenance, retention, deletion, provider, locality, and data-class controls.

## Public behavior

### Setup

The CLI selects one explicit topology. Combined setup acquires and manages both artifacts. External setup adopts an already running Hermes endpoint and manages only the Host. Interactive presentation and non-interactive flags must resolve into the same typed request; environment variables may supply test or operator inputs but are never readiness evidence.

### Doctor

Doctor derives its outcome from live observations and durable configuration:

- `ready`: Host state is valid; locally owned services have correct supervisor state; Hermes identity, authentication, capabilities, and required behavior pass.
- `degraded`: initialized Host state remains valid, but a previously validated Hermes service is temporarily unavailable or a non-authoritative operational dependency is impaired.
- `action_required`: setup is incomplete, canonical state or keys are invalid, the topology binding changed, a required local service is not boot-enabled, Hermes identity or credentials do not match, or compatibility is unsupported.

Doctor never treats a file, socket, environment marker, process name, or HTTP success alone as proof of readiness.

### Service lifecycle

Combined service commands control Hermes and Host through the selected local supervisor. External service commands control only the Host and report external Hermes status as an observation. No command crosses the network to manage external Hermes.

## Distribution and compatibility

The combined path pins the official Hermes repository, immutable tag and peeled commit, supported runtime version, lockfile, build recipe, artifact size, SHA-256 digest, and Lumen compatibility contract. Two clean builds must produce identical release identities.

External Hermes is not required to use Lumen's artifact, but it must report a version and capability surface certified by the same behavioral contract. Source or version metadata alone does not establish compatibility. Certification covers authenticated health, capability discovery, idempotent run creation, the sole Host-owned event stream, status reconciliation, exact `once` and `deny` approval, cancellation, bounded output, and secret-safe errors.

## Failure and recovery

| Condition | Required behavior |
| --- | --- |
| Setup interrupted | Resume from durable verified evidence; do not repeat a proven effect. |
| Topology or identity changes on rerun | Return `action_required` before mutation. |
| Artifact replacement fails | Restore the last verified artifact and preserve Space state. |
| External Hermes is unavailable after validation | Report `degraded`; never claim a task completed. |
| Endpoint identity, pin, or credential mismatches | Fail closed as `action_required`; do not send task context. |
| Hermes is incompatible before dispatch | Record bounded rejection evidence; do not create a run. |
| Run creation is uncertain | Reconcile only through durable idempotency evidence; otherwise record `unknown_outcome`. |
| Events duplicate, reorder, or disconnect | Apply Host-owned transition keys and bounded status reconciliation. |
| Host or Hermes restarts | Resume only durably mapped work; never redispatch an uncertain effect. |
| Cancellation races completion | Persist the first valid terminal transition. |

## Verification

### Milestone 0 reconciliation

- Run focused Go Space and encrypted-store tests, including race tests.
- Update stale historical command and removed-path references without changing the frozen historical evidence.
- Run link and cross-reference validation.

### Automated Milestone 1 checks

- Unit tests cover topology validation, immutable journal binding, configuration, credential separation, stage ownership, doctor classification, artifact replacement, and supervisor selection.
- Contract tests cover both public setup journeys, safe rerun, interruption, approval, cancellation, duplicate and reordered events, lost streams, Host and Hermes restart, incompatible capabilities, authentication failure, endpoint substitution, redirect rejection, pin mismatch, and secret redaction.
- `milestone1-combined-check` uses clean isolated volumes to prove installation, rerun, restart, configured task execution, failed update, rollback, and state preservation.
- `milestone1-external-check` uses separately supervised Host and Hermes environments to prove adoption without remote mutation, authenticated task execution, outage and recovery, credential and identity rejection, and Host-only service control.
- `milestone1-check` runs both topology gates plus formatting, vet, unit, race, reproducible build, deployment validation, and secret scanning.

### Owner evidence

Combined evidence uses a clean Ubuntu 24.04 LTS amd64 environment and records installation, boot restart, real pinned Hermes behavior, failed update, rollback, and preserved canonical state.

External evidence uses independently managed Host and Hermes machines. It records both machine identities and versions, TLS and credential preconditions without secrets, Host setup, authenticated Runs behavior, independent reboots, disconnection and reconnection, and proof that Lumen neither installed nor controlled the remote Hermes service.

Unrun external checks remain open. A local process restart, fake server, fixture manifest, or environment marker cannot substitute for owner evidence.

## Milestone exit

Milestone 1 closes only when both topologies pass their automated gates, real-runtime behavior is certified, required reboot and rollback evidence is recorded, no unresolved Critical or Important review finding remains, canonical documentation is current, and Graphify is updated.

The exit unlocks Milestone 2's canonical conversation and memory nucleus. It does not claim node transport, remote device capabilities, messaging, voice, managed hosting, or earned autonomy.

## Implementation direction

Keep the completed baseline-reconciliation commit. Replace the remaining Milestone 1 plan tasks with dependency-ordered slices for documentation reconciliation, topology contract, combined distribution, external adoption, public orchestration, Runs certification, automated journeys, owner evidence, and closure. Each behavior change follows test-first development and receives an independent task review before the next dependent slice begins.
