# O-004 Headless Host service decision

## Status

Accepted and current for the Space Authority process. The original platform list and Host-first sequence are historical delivery context: combined Linux/VPS is the first reference profile, while macOS and Termux qualify independently. This decision does not require every profile to finish before node-fabric or Jarvis-experience work begins.

## Decision

The Host is a headless native Go process with encrypted durable state and an authenticated local operator boundary. It runs in the foreground and delegates restart, boot, log capture, resource limits, and background supervision to systemd, launchd, Docker, or Termux/runit.

The same public Host contract and compatible distribution format must support Linux/VPS, macOS, and Android Termux when those profiles ship. Deployment adapters may provide service definitions and secret sources, but they cannot change Space authority, lifecycle semantics, task outcomes, or the Hermes adapter contract. A native Android companion is a client even when Termux runs the Host on the same phone.

The service owns Host recovery, health, canonical state, local operator commands, and later the authenticated node channel. It restores and commits recovery before accepting work. It reports `offline`, `degraded`, or `unknown_outcome` rather than inferring success from process or adapter state.

## Boundaries

- The executable never daemonizes itself. A service manager owns its background lifecycle and restart policy.
- The Host exposes its local control endpoint through an owner-restricted Unix-domain socket, verifies endpoint identity, and requires a generated credential stored outside ordinary logs and command history.
- Canonical state stays unavailable until its key and last committed state are verified and restart recovery is durably recorded.
- A storage error, unavailable key, incompatible state version, Hermes incompatibility, event-stream loss, or exhausted deadline moves affected work to an explicit unavailable, failed, or unknown state.
- Hermes is a separately supervised untrusted process. Hardened Linux, macOS, and VPS deployments isolate it under another OS principal or container and connect through a protected endpoint or proxy with pinned mutual TLS and a separately scoped bearer credential.
- Co-located plaintext loopback is an explicit development profile for synthetic state only. Because one Termux installation shares an Android UID, a co-located Termux Hermes proves compatibility but not isolation; a security-valid Termux deployment uses an isolated remote Hermes endpoint with pinned mutual TLS.
- Local node work may continue only under an unexpired cached grant; it queues its permitted delta until the Host returns.
- No microphone, camera, location, display, accessibility, or mobile foreground-service permission belongs to the Host.

## Non-goals

- Automatic failover, multi-master state, remote public administration, and embedding a model runtime inside the Host authority process.
- Pairing, node transport, companion UI, and device capabilities inside the foundation-closure slice. They are separate milestones and reuse this Host boundary. A remote isolated Hermes endpoint is permitted where the platform cannot enforce local process isolation, including hardened Termux use.

## Acceptance checks

1. Run the Host distribution from a terminal on the combined Linux/VPS reference profile; confirm initialization, readiness, status, shutdown, and restart semantics.
2. Start it under a server supervisor; kill the process during an active task and confirm recovery or `unknown_outcome`, never unsupported success.
3. Connect through the hardened authenticated Hermes boundary, validate `/v1/capabilities` and readiness, submit one run, consume bounded SSE events, cancel one run, and complete one exact one-time approval flow.
4. Simulate missing keys, partial persistence, incompatible Hermes capabilities, authentication failure, event-stream loss, duplicate events, and timeout; confirm the Host fails closed and preserves its last committed authority record.
5. Repeat the applicable lifecycle contract separately before claiming macOS or Termux support; Termux/runit and same-UID constraints remain profile-specific evidence.
