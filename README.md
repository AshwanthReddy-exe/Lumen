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

The production Host and Space authority are native Go; the superseded Kotlin Multiplatform authority and scenario runner were retired after the Go gate passed. Android remains a standalone Kotlin companion and Apple applications remain Swift. Production directories are created only as each delivery phase implements them. The canonical documents are in [docs/](./docs/); contribution rules are in [AGENTS.md](./AGENTS.md).

Run the Phase 0 contract baseline with `mise run phase0-check`. It uses the pinned Java and Gradle environment and the system-managed Swift toolchain.

Phase 0 evidence lives under [`spikes/`](./spikes/) and freezes the stack, context, transport, Host, recovery, and capability decisions before Phase 1 begins.

Phase 1’s Kotlin Space-core contract and simulated restart scenario are retained as historical documentation in [PHASE-1-CONTRACT.md](./docs/PHASE-1-CONTRACT.md); production authority is now Go.

Run the native Block 2 gate with `ANDROID_HOME=/Users/ashwanthreddyboddireddy/Library/Android/sdk mise run phase2-check`. It verifies Go formatting, vet, unit/contract tests, race tests, reproducible amd64 and linux/arm64 builds, Android companion unit tests, and debug APK assembly. Android verification is required; the gate fails when `ANDROID_HOME` is unavailable.

Start the Mac development Host with `scripts/lumen-mac-start`. It rebuilds the native binary from the current source, initializes private state once under `~/.local/share/lumen`, starts the foreground service, and appends redacted Host output to `~/.local/share/lumen/host.log`. In another terminal use `scripts/lumen-mac-status`, `scripts/lumen-mac-logs`, or `scripts/lumen-mac-stop`. Use `scripts/lumen-mac-task task submit ...` for authenticated task commands so the shared environment is loaded. The default development profile expects Hermes on loopback at `http://127.0.0.1:8642`; hardened deployments use the supervisor and secret configuration in `deploy/README.md`.

## Current priority

**Finish the headless Host and its Hermes execution loop before implementing pairing, nodes, or companion features.** The active plan and exit gate are in [PHASE-2-HOST-HERMES.md](./docs/PHASE-2-HOST-HERMES.md). The abandoned Android foreground-Host direction is recorded only in Git history and the changelog; it is not the architecture to extend.
