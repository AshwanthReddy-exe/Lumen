---
last_mapped_commit: ebe42cda0332c603644874b73ff920964ba853f1
last_mapped_at: 2026-09-26
---
# Technology Stack

**Analysis Date:** 2026-09-26

## Languages

**Primary:**

- Go 1.27.1 - Production Space authority, Host service, operator CLI, storage, setup, and Hermes adapter (`go.mod:1-3`, `README.md:37-48`).

**Secondary:**

- Kotlin - Android companion shell (`apps/android-host/`); the README says it is disabled and does not own Space state (`README.md:46-48`).
- Swift and Kotlin - Historical Phase 0 schema/protocol spikes only (`spikes/o-001/native-schema/Package.swift`, `spikes/o-001/kmp/build.gradle.kts`, `docs/O-001-STACK-SPIKE.md:3-12`).
- Python 3.11 - Pinned Hermes runtime image, not Lumen Host implementation (`deploy/docker/Dockerfile.hermes:1-21`).

## Runtime

**Environment:**

- Go toolchain 1.27.1 for Host and CLI (`mise.toml:1-5`).
- Java 25.0.2 and Gradle 9.5.0 for Android checks (`mise.toml:1-5`).
- Python 3.11 in the Hermes container (`deploy/docker/Dockerfile.hermes:1,17`).

**Package Manager:**

- Go modules; `go.mod` contains only the module declaration and Go version, with no third-party Go requirements (`go.mod:1-3`).
- Gradle for Android and the historical Kotlin spike (`apps/android-host/build.gradle.kts`, `spikes/o-001/kmp/build.gradle.kts`).
- `uv` with committed Hermes `uv.lock` and frozen synchronization (`deploy/hermes-source.lock.json:8-11`, `deploy/docker/Dockerfile.hermes:2,15`).
- Lockfiles: `go.sum` is absent; `deploy/hermes-source.lock.json` pins Hermes source and its archive digest; Hermes `uv.lock` is inside the pinned source archive.

## Frameworks

**Core:**

- Standard-library Go service and CLI; packages are divided under `internal/space`, `internal/host`, `internal/store`, `internal/control`, `internal/hermes`, and `internal/setup` (`README.md:37-44`).
- Android application uses Gradle/Kotlin (`apps/android-host/build.gradle.kts`).
- No production Go web framework or database ORM is declared (`go.mod:1-3`).

**Testing:**

- Go `testing` package for unit, integration, contract, recovery, and security checks (`mise.toml:14-20,38-71`; test files under `internal/` and `test/contract/`).
- Gradle Android unit/instrumentation tests (`apps/android-host/src/test/`, `apps/android-host/src/androidTest/`, `mise.toml:34-35`).
- Swift Package Manager and Gradle also run historical cross-language fixture spikes (`mise.toml:7-12`).

**Build/Dev:**

- `mise` pins tool versions and defines named checks (`mise.toml:1-5,7-72`).
- Docker multi-stage Go build produces a non-root distroless Host image (`deploy/docker/Dockerfile:1-23`).
- Hermes is built from a commit- and archive-digest-pinned source bundle (`deploy/docker/Dockerfile.hermes:4-15`, `deploy/hermes-source.lock.json:1-12`).

## Key Dependencies

**Critical:**

- No third-party Go runtime modules are declared; the Host uses Go standard-library packages and in-repository modules (`go.mod:1-3`, imports in `internal/hermes/client.go:5-21`).
- Hermes Agent `v2026.9.7`, commit `2237be355906fbe6065ce1815711eee52b2d646e` - External agent/runtime API integration, separately built and supervised (`deploy/hermes-source.lock.json:1-7`).

**Infrastructure:**

- Docker Compose - Combined Host/Hermes topology and Host-only external-Hermes topology (`deploy/docker/compose.yaml:1-77`, `deploy/docker/compose.external.yaml:1-33`).
- Native supervisors - launchd, systemd, and Termux/runit definitions (`deploy/launchd/`, `deploy/systemd/`, `deploy/termux/`).
- Android SDK (`ANDROID_HOME`) is required by the Android gate (`mise.toml:34-35`).

## Configuration

**Environment:**

- Host configuration is environment-driven: data directory, local socket, operator credential file, Hermes profile, endpoint, and bearer credential path (`internal/host/service.go:38-59`).
- Hardened Hermes configuration additionally requires CA roots, client certificate/key, HTTPS endpoint, and pinned server certificate (`internal/hermes/client.go:53-79,217-270`; `deploy/docker/compose.external.yaml:8-23`).
- Secrets are mounted/read from files; Compose mounts credentials read-only (`deploy/docker/compose.yaml:53-67`, `deploy/docker/compose.external.yaml:8-23`).
- `.env` or credential file contents are not part of this map; configuration names and examples only are recorded.

**Build:**

- Go module: `go.mod`; toolchain and commands: `mise.toml`.
- Docker build definitions: `deploy/docker/Dockerfile`, `deploy/docker/Dockerfile.hermes`.
- Android build definition: `apps/android-host/build.gradle.kts`.
- Hermes source pin: `deploy/hermes-source.lock.json`.

## Platform Requirements

**Development:**

- Go 1.27.1 and `mise` for the Host gates; Docker is required for live container topology checks (`mise.toml:1-5,38-72`).
- Java 25.0.2, Gradle 9.5.0, and Android SDK for Android application tests/builds (`mise.toml:1-5,34-35`).
- macOS system tools are required by macOS-specific checks; `plutil` validates launchd configuration (`mise.toml:30-35,53-60`).

**Production:**

- Documented Host targets are Linux amd64/arm64, macOS amd64/arm64, and Android arm64 under Termux (`deploy/README.md:9-11`).
- Combined Compose is Linux-oriented; macOS runs the Host natively and can use a separate Hermes gateway (`deploy/README.md:39-45`).
- Windows Host and the planned iOS/Android product clients are not current production targets (`deploy/README.md:9-11`).

## Evidence State

- **Implemented on `main` (`ebe42cd`):** Go Host/CLI, encrypted local state, operator control, Hermes Runs adapter, setup/release components, and Android shell (`README.md:37-48`; `internal/hermes/client.go:181-199`).
- **Automated checks configured:** Go tests, vet, race, cross-builds, artifact reproducibility, Compose/script checks, and Android tests (`mise.toml:14-72`). These task definitions are not themselves proof that every task ran in this checkout.
- **Live-proven per deployment record:** Azure Linux combined Compose and same-VPS external Hermes lifecycle/reboot journey; the record explicitly limits what those runs establish (`deploy/README.md:33-37`).
- **PR #20 branch only:** `origin/feat/lumen-continuity` adds canonical conversations/memory, a development chat path, and a bounded signed `node.status/read` path. Those files are absent from `main`; branch-local synthetic evidence is recorded at `docs/evidence/HERMES-CHAT-PREFLIGHT-2026-09-25.md:12-14` and branch `docs/PLAN.md:112-124`. PR #20 remains a distinct draft branch and its own evidence must not be attributed to this main-based stack.

---

*Stack analysis: 2026-09-26*
