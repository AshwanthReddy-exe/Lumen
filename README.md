# Lumen

Lumen links a person’s devices into one private **Space**. From any connected device, the user can talk to Lumen, continue shared context, and invoke explicitly permitted capabilities on the same device or another node.

## Core model

- **Space:** the shared identity, context, policy, task history, and device registry.
- **Host:** the one active coordinator that stores canonical Space state and routes cross-node work. Any eligible node may be configured as Host.
- **Node:** a paired phone, computer, server, or future device that provides interaction and execution capabilities.
- **Capability:** a bounded action such as coding, reminders, notifications, files, or scheduled tasks. Each capability can be denied, allowed, or set to ask.
- **Companion:** an optional friendly interface, such as an old phone on a desk or a Mac pet. It is not required to use Lumen.

Same-device work takes a local fast path and later synchronizes relevant context with the Host. Cross-node work is authenticated, authorized, and routed through the Host.

## First reference setup

```text
Old Android phone: active Host + desk companion
Mac:               pet surface + coding/execution node
iPhone:            interaction + personal capability node
```

The topology is configurable: a Mac, Linux machine, old PC, home server, or VPS can become the Host when it meets the Host contract.

## Planned repository structure

```text
apps/               android-host, ios-node, macos-node
core/               space, context, policy, tasks, scheduler
packages/           protocol and shared capability contracts
adapters/           Hermes and platform capability adapters
services/           optional relay and notification transport
tests/              contract, integration, security, recovery
infra/              development and deployment configuration
```

Phase 0 selected a Kotlin Multiplatform core with native Android and Apple applications. Production directories are created only as each delivery phase implements them. The canonical documents are in [docs/](./docs/); contribution rules are in [AGENTS.md](./AGENTS.md).

Run the Phase 0 contract baseline with `mise run phase0-check`. It uses the pinned Java and Gradle environment and the system-managed Swift toolchain.

Phase 0 evidence lives under [`spikes/`](./spikes/) and freezes the stack, context, transport, Host, recovery, and capability decisions before Phase 1 begins.
