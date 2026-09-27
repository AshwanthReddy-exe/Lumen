---
last_mapped_commit: ebe42cda0332c603644874b73ff920964ba853f1
last_mapped_at: 2026-09-26
---
# External Integrations

**Analysis Date:** 2026-09-26

## APIs & External Services

**Agent runtime:**

- Hermes Agent API - Host probes capabilities and health, submits/observes/stops runs, consumes SSE events, and resolves supported approvals (`internal/hermes/client.go:23-34,181-199`).
  - SDK/Client: in-repository Go adapter using `net/http`; no third-party Go SDK (`internal/hermes/client.go:5-21`, `go.mod:1-3`).
  - Auth: bearer credential loaded through configured file/source; Compose mounts it read-only (`internal/host/service.go:38-59`, `deploy/docker/compose.yaml:53-67`).
  - Trust: development permits loopback HTTP only; hardened mode requires HTTPS, explicit CA roots, client identity, and certificate pinning (`internal/hermes/client.go:217-270`; `deploy/docker/compose.external.yaml:12-23`).
  - Boundary: adapter reports runtime evidence and cannot authorize or mutate Space state (`internal/hermes/client.go:1-3,181-192`).

**Artifact distribution:**

- GitHub Releases - Installer resolves a pinned manifest and verifies artifact size and SHA-256 before installing the Host (`deploy/README.md:3-20`).
  - SDK/Client: shell installer and Go release tooling (`scripts/install.sh`, `scripts/lumen-release`, `cmd/lumen/release.go`).
  - Auth: public artifact download; publication requires a GitHub token with package-write access for Hermes image publishing (`deploy/README.md:13-20`).
  - State: scripted release publication exists; repository code and docs alone do not prove a particular current release is published.

## Data Storage

**Databases:**

- No external database is configured. Host canonical state is stored locally in an encrypted file snapshot with atomic replacement and process locking (`internal/store/store.go`, `internal/store/envelope.go`; `docs/ARCHITECTURE.md`, section “Persistence decision”).
  - Connection: local `LUMEN_DATA_DIR` (`internal/host/service.go:38-59`).
  - Client: in-repository Go store; AES-256-GCM envelope is implemented in `internal/store/envelope.go`.
- Hermes maintains its own separate runtime state; it is not the canonical Space store (`docs/ARCHITECTURE.md`, sections “Hermes integration and enforcement” and “Persistence decision”).

**File Storage:**

- Local filesystem only on `main`; encrypted Host state and separate Hermes volume in combined Compose (`deploy/docker/compose.yaml:53-77`).
- No object-storage service is integrated.

**Caching:**

- No external cache service is integrated.

## Authentication & Identity

**Auth Provider:**

- Custom Host-local owner authentication through an operator credential and Unix-domain control socket (`internal/host/service.go:38-59`, `docs/ARCHITECTURE.md`, section “Host process boundary”).
- Hermes authentication uses bearer credentials; hardened remote transport additionally verifies server identity and presents a client certificate (`internal/hermes/client.go:53-79,231-270`).
- User/node pairing identity and authenticated node transport are planned on `main`, not implemented as a production integration (`README.md:46-48`, `docs/PLAN.md`, “Secure node fabric”).

## Monitoring & Observability

**Error Tracking:**

- No hosted error-tracking service is integrated.

**Logs:**

- Go service logging is local; setup and doctor surfaces report local deployment health (Host composition under `internal/host/`, `cmd/lumen-host/`; `docs/ARCHITECTURE.md`, section “Host process boundary”).
- No external metrics or log aggregation backend is configured.

## CI/CD & Deployment

**Hosting:**

- Deployment definitions support combined Docker Compose Host+Hermes and external-Hermes Host-only Compose (`deploy/docker/compose.yaml:1-77`, `deploy/docker/compose.external.yaml:1-33`).
- Native Host service definitions also exist for launchd, systemd, and Termux/runit (`deploy/launchd/`, `deploy/systemd/`, `deploy/termux/`).
- Live evidence recorded: Azure Ubuntu combined deployment and same-VPS external endpoint with pinned mutual TLS, including reboot persistence (`deploy/README.md:33-37`). The record does not establish second-machine deployment or live-update proof.

**CI Pipeline:**

- No `.github/workflows/` directory is present in this checkout. `mise.toml` defines local checks; it does not establish a hosted CI pipeline (`mise.toml:7-72`).

## Environment Configuration

**Required env vars:**

- Host: `LUMEN_DATA_DIR`, optional `LUMEN_SOCKET_PATH`, `LUMEN_OPERATOR_CREDENTIAL_FILE`, `LUMEN_HERMES_PROFILE`, `LUMEN_HERMES_BASE_URL`, `LUMEN_HERMES_BEARER_FILE` (`internal/host/service.go:38-59`).
- Hardened Hermes: CA, client certificate/key, and server certificate pin paths/configuration (`deploy/docker/compose.external.yaml:8-23`).
- Combined Compose additionally expects data and secret source paths and Hermes container settings (`deploy/docker/compose.yaml:53-67`).

**Secrets location:**

- File-backed under private Host state or service secret mounts; Compose mounts Hermes bearer, CA, client cert and key files read-only (`deploy/README.md:22-31`, `deploy/docker/compose.yaml:62-67`, `deploy/docker/compose.external.yaml:18-23`).
- Never pass credentials as command-line arguments; this is stated in the Dockerfile and deployment guidance (`deploy/docker/Dockerfile:19-24`, `deploy/README.md:22-31`).

## Webhooks & Callbacks

**Incoming:**

- None implemented on `main`; there is no public conversation or node API server in the current Host surface (`README.md:46-48`).

**Outgoing:**

- Hermes API requests and SSE observation are the current runtime integration (`internal/hermes/client.go:181-199`). No messaging-provider webhook integration is implemented on `main` (`README.md:48`).

## PR #20 Branch Boundary

- `origin/feat/lumen-continuity` is 26 commits beyond main and changes Hermes packaging plus adds `integrations/hermes/`, `internal/conversation/`, `internal/node/`, and a local chat journey. The branch retains the same Go module declaration (`git diff --stat main...origin/feat/lumen-continuity`, branch `go.mod:1-3`).
- Branch evidence reports local arm64 synthetic Hermes/container probes and a native macOS Host chat journey, while explicitly leaving production process-to-request binding, real-provider qualification, and manual owner acceptance open (`docs/evidence/HERMES-CHAT-PREFLIGHT-2026-09-25.md:7-14` on that branch).
- Its signed `node.status/read` route has automated coverage only; branch plan states device pairing identity, owner confirmation, durable key binding, reconnect, and general capability transport remain open (`docs/PLAN.md:120-124` on that branch).
- Do not describe these branch-only integrations as implemented on `main` or as production-ready.

---

*Integration audit: 2026-09-26*
