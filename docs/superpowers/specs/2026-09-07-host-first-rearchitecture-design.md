# Host-first rearchitecture design

## Decision summary

Lumen will build and validate its authority process before building network clients. The Host is a headless terminal service. Hermes is its first execution adapter. Android is a companion/node app, while Android Termux is one supported environment for the same Host service artifact.

This design supersedes the assumption that installing the Android companion makes that APK the Host. Published commits remain as historical evidence; corrective commits remove the wrong runtime ownership without rewriting shared history.

## Invariants

1. Exactly one active Host owns canonical Space state, policy, tasks, approvals, context, schedules, and audit history.
2. The Host runs without a graphical application and does not depend on Android lifecycle APIs.
3. Linux/VPS, macOS, and Android Termux execute the same Host contract and distribution baseline.
4. A supervisor may restart the process but cannot create authority, infer completion, or choose retry semantics.
5. Hermes is a separately supervised untrusted adapter. It cannot grant capability access, consume Lumen approvals, write canonical context, or finalize a task by itself.
6. Companion applications remain optional clients even when co-located with the Host.

## Architecture

The Kotlin Multiplatform Space core remains inward-facing. Its first Phase 2 increment adds the minimum Host-local execution lifecycle that the daemon needs: distinct owner and Host-executor principals, durable dispatch/run evidence, running and cancellation states, terminal compare-and-set rules, and evidence-bound reconciliation after uncertainty. A Kotlin/JVM 21 Host service composes that core with encrypted filesystem persistence, a loopback operator endpoint, lifecycle and health reporting, and a narrow Hermes HTTP/SSE adapter. Service managers run the foreground process in the background appropriate to each operating environment.

The initial control plane is local only. This allows the authority, persistence, and runtime boundaries to be tested without prematurely designing node identity or network trust. Node pairing and transport start only after the Host/Hermes exit gate passes.

## Host and Hermes data flow

The recoverable owner and active Host service are distinct principals. The active Host identity is also the initial local execution target and advertises the Hermes-backed capability. An authenticated operator command maps to the owner principal; the operator credential does not become a node, Host, storage, or Hermes key.

The Host durably accepts a typed, idempotent task before calling Hermes. It provides only approved task input, scoped context, deadline, and cancellation identity. It records the Hermes run identifier and consumes one bounded, schema-checked event stream. Because Hermes SSE is ephemeral evidence rather than a durable ordered log, the Host assigns its own ingestion records, makes normalized transitions idempotent, and reconciles terminal state through bounded status queries after a disconnect. Tool requests and runtime approvals are proposals. The Host evaluates its own grant and exact approval record before forwarding any decision.

Completion requires evidence that satisfies the Host transition contract. Loss of the event stream triggers bounded status reconciliation. If the Host cannot prove a terminal outcome, it records `unknown_outcome`; it never converts process exit, HTTP success, or an adapter claim into completion.

## Deployment model

`lumen-host serve` remains in the foreground. systemd, launchd, a Docker supervisor, or Termux/runit provides background execution and restart policy. Configuration identifies a private data directory, local operator socket, Hermes endpoint, endpoint identity pin, and secret references. Host-state encryption, operator control, and Hermes authentication use separate generated credentials. Secrets are read from restricted files or supervisor credentials and are redacted from diagnostics.

Hardened Linux, macOS, and VPS deployments run Host and Hermes under separate OS principals or container boundaries. The operator uses an owner-restricted Unix-domain socket and verifies the Host endpoint identity. Hermes sits behind a protected TLS endpoint or proxy with pinned server identity and client authentication; redirects and URL userinfo are rejected. Plain loopback HTTP is allowed only in an explicit development profile with synthetic state and cannot satisfy the security exit gate.

Termux is a first-class compatibility target, not a separate Android Host implementation. The native Android app does not share the Host data directory or Android Keystore state. When both are eventually paired on one phone, they communicate through the same authenticated client boundary as separate processes.

Termux/runit and Hermes processes inside one Termux installation share the same Android UID; runit or proot is not an isolation boundary. A same-UID Hermes run therefore proves only protocol compatibility with synthetic, non-sensitive state. A security-valid Termux deployment connects the Host to Hermes isolated on another machine or container through pinned mutual TLS. Compromise of the Termux UID remains compromise of the Host and is stated during setup.

## Android correction

The existing Android Host service, Host lifecycle UI, and canonical Android store are removed from the future companion module. The current installed Space is development data from a superseded topology and must not silently become node state. The rewire will use a deliberate development reset or explicit quarantine step before installing the corrected companion.

Before any new production Space is initialized, the old Android foreground service must be stopped and its test Space explicitly discarded or archived as non-authoritative evidence. The action and its recoverability are recorded. The new Host never imports that state automatically.

The companion later owns only node identity, a minimum encrypted local cache/journal, Host connection status, approval/task presentation, and separately permissioned local capabilities such as text-to-speech, microphone, and camera.

## Sequencing

The immediate work is the headless service, durable Host storage, Hermes adapter, and Host-local execution loop. Pairing, local-network transport, Android UI expansion, Mac/iPhone nodes, and hardware companion capabilities are downstream and cannot begin merely because they can be mocked.

The canonical breakdown and acceptance evidence are in `docs/PHASE-2-HOST-HERMES.md`. That document is the required starting point for the next session.
