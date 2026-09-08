# Lumen

Lumen links a person’s devices into one private **Space**. From any connected device, the user can talk to Lumen, continue shared context, and invoke explicitly permitted capabilities on the same device or another node.

## Core model

- **Space:** the shared identity, context, policy, task history, and device registry.
- **Host:** the one active headless service that stores canonical Space state and routes work. A device may run both Host and node processes, but their identities and state remain separate.
- **Node:** a paired phone, computer, server, or future device that provides interaction and execution capabilities.
- **Capability:** a bounded action such as coding, reminders, notifications, files, or scheduled tasks. Each capability can be denied, allowed, or set to ask.
- **Companion:** an optional friendly interface, such as an old phone on a desk or a Mac pet. It is not required to use Lumen.

Same-device work takes a local fast path and later synchronizes relevant context with the Host. Cross-node work is authenticated, authorized, and routed through the Host.

## First reference setup

```text
Terminal service:  active Host + canonical Space state
Hermes:            Host-local execution runtime adapter
Old Android phone: desk companion + capability node
Mac:               future pet surface + coding node
iPhone:            future interaction + personal-capability node
```

The Host is a headless service, not a companion application. The same Host artifact must run under a service manager on Linux/VPS, macOS, and Android Termux. A companion may share the same physical device, but it connects to the Host through the ordinary authenticated boundary and never owns canonical Space state.

## Planned repository structure

```text
cmd/                native Go Host executable
internal/           Go Space core, Host composition, storage, control, Hermes adapter
protocol/           versioned schemas and cross-language fixtures
apps/               android-companion, ios-node, macos-node
test/               contract, integration, security, recovery
deploy/             service-manager and container configuration
```

Phase 0's Kotlin Multiplatform core remains migration evidence while Block 2 replaces production authority with Go. Android remains Kotlin and Apple applications remain Swift. Production directories are created only as each delivery phase implements them. The canonical documents are in [docs/](./docs/); contribution rules are in [AGENTS.md](./AGENTS.md).

Run the Phase 0 contract baseline with `mise run phase0-check`. It uses the pinned Java and Gradle environment and the system-managed Swift toolchain.

Phase 0 evidence lives under [`spikes/`](./spikes/) and freezes the stack, context, transport, Host, recovery, and capability decisions before Phase 1 begins.

Run the Space-core contract and simulated restart scenario with `mise run phase1-check`. Phase 1 is complete; the [core contract](./docs/PHASE-1-CONTRACT.md) defines the Phase 2 platform-store boundary.

## Current priority

**Finish the headless Host and its Hermes execution loop before implementing pairing, nodes, or companion features.** The active plan and exit gate are in [PHASE-2-HOST-HERMES.md](./docs/PHASE-2-HOST-HERMES.md). The abandoned Android foreground-Host direction is recorded only in Git history and the changelog; it is not the architecture to extend.
