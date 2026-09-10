# Lumen-owned setup and Hermes frontier design

## Status

Approved in conversation on 2026-09-10. This design refines Block 2 installation and reserves the Block 3 connection handoff. It does not claim that either implementation is complete.

## Purpose

An owner should be able to turn an eligible macOS, Linux/VPS, old PC, or Android Termux environment into a working Lumen Host without learning Hermes installation, environment variables, or service managers. Lumen remains the product surface and authority; Hermes remains a replaceable, untrusted execution adapter.

The public first-run journey is:

```text
lumen setup
```

On success, Lumen and Hermes are installed or located, privately configured, initialized, supervised, started, and verified. The command ends with one redacted ready or degraded report and exact owner actions when automation cannot safely finish.

## Command boundary

`lumen` is the public lifecycle and integration CLI. `lumen-host` remains the internal foreground service binary and retains low-level commands used by supervisors and recovery tooling.

- `lumen setup` owns first installation and safe reruns.
- `lumen doctor` reports the whole deployment, including Host, Hermes, supervision, and integration readiness.
- `lumen service start|stop|restart|status` controls the installed services through the platform adapter.
- `lumen integration list|connect|disconnect|status` exposes supported Hermes-backed integrations through Lumen policy.
- `lumen connect` is reserved in Block 2 as the public node-pairing entry point; its authenticated QR/manual-code behavior is implemented and released in Block 3.
- `lumen-host init`, `serve`, and the operator protocol remain internal building blocks rather than the normal owner journey.

## Setup contract

`lumen setup` is a resumable state machine, not an opaque shell pipeline. Each completed step has locally verifiable evidence so rerunning the command continues safely without replacing initialized Space state or silently rotating credentials.

1. Detect the platform, architecture, available supervisor, current installation, and requested deployment profile.
2. Select only a supported, pinned Lumen and Hermes distribution pair. An unsupported combination stops before mutation.
3. Install or update artifacts using checksums and version compatibility metadata.
4. Create private Lumen and Hermes directories and credentials with platform-appropriate ownership and permissions.
5. Generate validated Lumen and Hermes configuration without requiring the owner to edit environment files.
6. Initialize encrypted Host state exactly once through the existing create-once primitive.
7. Install Host and Hermes supervisor definitions, enable boot start where supported, and start services in dependency order.
8. Validate Host readiness, Hermes health and capability compatibility, supervisor state, and the selected isolation profile.
9. Return a redacted structured result: `ready`, `degraded`, or `action_required`, plus safe next actions.

Setup accepts explicit non-interactive inputs for automation, but secrets must come from protected files, file descriptors, platform credential stores, or interactive hidden input. Secrets never appear in command arguments, generated logs, ordinary status, or task prompts.

## Deployment profiles

The setup planner chooses behavior from explicit profiles. Detection may recommend a profile but cannot silently weaken one.

### Development

Host and Hermes may use authenticated numeric loopback with synthetic state. This profile is convenient compatibility evidence and is never reported as hardened.

### Personal alpha

Host and Hermes may be co-located when the platform can keep credentials private enough for the owner's stated risk. The report names the remaining isolation limitations. On Android Termux, same-UID Hermes is compatibility-only and cannot satisfy hardened evidence.

### Hardened

Host and Hermes run under enforceably separate principals or containers and communicate through authenticated TLS with a pinned server identity and separately scoped bearer credential. If the current device cannot provide that boundary, setup configures an explicitly supplied isolated Hermes endpoint or returns `action_required`; it never downgrades automatically.

## Hermes installation boundary

Lumen owns acquisition, compatibility, configuration, supervision, health checks, and upgrades for the Hermes version it supports. Platform adapters may use native package tools, containers, or a verified upstream installer, but all produce the same setup evidence and normalized status.

Hermes configuration is generated from a typed Lumen setup model. Environment files are an adapter output, not the user-facing source of truth. Existing external Hermes installations may be adopted only after version, endpoint identity, authentication, capability, and ownership checks pass.

An update stages new artifacts, validates them before service replacement, and preserves the last working configuration for rollback. An uncertain or partially completed update is reported honestly and does not reinitialize the Space.

## Hermes capability frontier

Lumen maximizes Hermes reuse without becoming an unrestricted proxy. Hermes may implement model routing, tools, skills, MCP, browser automation, post-activation voice, delegation, remote execution, Telegram, WhatsApp, and future integrations. A discovered Hermes feature is unavailable to users until Lumen has a typed capability contract for it.

Every exposed integration defines:

- a stable Lumen capability and action set;
- target, account, recipient, resource, context, cost, and expiry scopes as applicable;
- credential ownership and the minimum credential handle visible to Hermes;
- `deny`, `ask`, or `allow` policy, with one-time approval for authority expansion;
- idempotency, timeout, cancellation, retry, receipt, and uncertain-outcome behavior;
- redacted status and audit events;
- a pinned Hermes feature/version contract and an off switch.

`lumen integration connect telegram` and similar commands may launch Hermes-provided authentication mechanics, including a provider QR flow, but Lumen owns the lifecycle and records only redacted connection state. Setup never silently signs into an external account. Provider authentication QR codes are distinct from Lumen node-pairing QR codes.

Block 2 installs Hermes, imports and reports its capability inventory, and exposes only the already-approved `agent.run/execute` contract. Later blocks enable messaging, browser, voice, coding, delegation, and account integrations incrementally after each Lumen capability contract and negative test suite is frozen.

## Node connection boundary

Block 2 reserves the `lumen connect` command family and the setup output needed to discover it. It does not ship an unauthenticated shortcut around Block 3.

Block 3 implements a Host-generated, single-use, short-lived pairing offer represented as both a QR payload and a manual code. The joining node generates its own identity keys, the Host validates the offer and current epoch, and the owner explicitly confirms the device identity before membership is committed. Pairing then establishes the authenticated encrypted node channel, replay protection, reconnect behavior, revocation, and device management.

The device surface reports identity, role, health, capabilities, grants, revocation state, and last synchronization without exposing keys or pairing secrets.

## Failure behavior

- Unsupported platform or architecture: stop before mutation and identify the unsupported component.
- Artifact verification failure: retain the prior installation and report failure.
- Existing incompatible configuration: preserve it, show the conflicting fields redacted, and require an explicit migration action.
- Interrupted setup: rerun from durable evidence; never regenerate initialized Host identity or duplicate services.
- Hermes unavailable or incompatible: keep Host authority available, report `degraded`, and reject intelligent work honestly.
- Supervisor unavailable: install and validate artifacts, report `action_required`, and provide the exact foreground fallback without claiming boot readiness.
- External account authentication required: report `action_required` and launch it only with owner intent.
- Hardened isolation unavailable: fail that profile without silently selecting development or personal alpha.

## Validation

Block 2 setup is complete only when automated clean-environment tests and recorded owner evidence prove:

- one public setup command works on supported macOS, Linux/VPS, and Android Termux profiles;
- a rerun is idempotent and an interrupted run resumes without identity or state replacement;
- Lumen and Hermes configuration contains no manual placeholder edits after setup completes;
- Host and Hermes start after reboot under the installed supervisor;
- `lumen doctor` distinguishes ready, degraded, and action-required states without leaking secrets;
- a real bounded `agent.run/execute` task completes through the configured Hermes runtime;
- incompatible artifacts, insecure permissions, missing isolation, failed health, and partial installation produce honest failure evidence;
- development, personal-alpha, and hardened claims are tested separately and never inferred from a weaker profile.

Block 3 separately proves QR and manual-code pairing, expiry, one-time use, wrong-owner and wrong-Space rejection, replay resistance, explicit confirmation, reconnect, revocation, and device-management accuracy.

## Documentation impact

After this design is accepted, update `README.md`, `docs/PRD.md`, `docs/ARCHITECTURE.md`, `docs/DECISIONS.md`, `docs/PLAN.md`, `docs/PHASE-2-HOST-HERMES.md`, and `docs/CHANGELOG.md` together. The implementation plan must begin with the setup state-machine contract and one supported development profile, then add hardened desktop/VPS and Termux adapters without weakening the shared outcome contract.
