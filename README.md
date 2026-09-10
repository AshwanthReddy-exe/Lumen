# Lumen

Lumen links a person’s devices into one private **Space**. From any connected device, the user can talk to Lumen, continue shared context, and invoke explicitly permitted capabilities on the same device or another node.

The product north star is a lethal install-and-connect loop: `lumen setup` turns any eligible Mac, Linux/VPS box, old PC, or Android Termux phone into a configured Host without manual Hermes plumbing, then `lumen connect` pairs nodes by QR or short code. Lumen installs or adopts Hermes, generates both configurations, supervises both services across reboot, and reports development, personal-alpha, hardened, degraded, or action-required conditions honestly.

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
Mac:               current Host development harness; future pet surface + coding node
iPhone:            future interaction + personal-capability node
```

The Host is a headless service, not a companion application. The same Host artifact must run under a service manager on Linux/VPS, macOS, and Android Termux. A companion may share the same physical device, but it connects to the Host through the ordinary authenticated boundary and never owns canonical Space state.

## Target repository structure

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

The gate also builds `build/lumen-host/lumen-host-android-arm64` with Go's Android ARM64 target; it is an Android PIE using `/system/bin/linker64` and must be installed as the foreground `lumen-host` binary. A physical Termux run remains required owner evidence.

For the Mac-only development setup, run `scripts/lumen-mac-host start` in one terminal. It rebuilds the native binary, initializes private state once under `~/.local/share/lumen`, keeps the Host in the foreground, and writes redacted output to `~/.local/share/lumen/host.log`. Use `lumen-host doctor` for a local deployment-profile report and `scripts/lumen-mac-host status|stop|logs` for lifecycle operations. This loopback harness is development evidence only; it is not the isolated-mTLS or Termux release proof.

With the Host and the separately supervised local Hermes gateway running, execute `scripts/lumen-mac-test` in another terminal. It checks both services, submits a unique bounded task, resolves the one-time Lumen policy approval, waits up to five minutes, and prints the durable Hermes answer. Pass a custom prompt as one quoted argument. A Hermes runtime tool approval is never silently accepted; the script displays it and stops for an explicit operator decision.

## Current priority

**Finish the Lumen-owned Host and Hermes setup journey before implementing authenticated pairing, nodes, or companion features.** The execution core exists; the active work is the public setup CLI, Hermes lifecycle/configuration, service installation, whole-deployment doctor, and clean-machine release evidence. The active plan and exit gate are in [PLAN.md](./docs/PLAN.md) and [PHASE-2-HOST-HERMES.md](./docs/PHASE-2-HOST-HERMES.md).
