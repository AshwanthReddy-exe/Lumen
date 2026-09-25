# Host provisioning

## Installing a release

```sh
curl -fsSL https://github.com/AshwanthReddy-exe/Lumen/releases/latest/download/install.sh | sh
```

The installer detects the host platform, resolves the matching artifact from the pinned release manifest, verifies its size and SHA-256 before writing anything, and installs `lumen-host`. It never initializes a Space and never touches Host state; set `LUMEN_DATA_DIR` to a private directory and run `lumen setup` as a separate, explicit step. `LUMEN_VERSION` requires a specific version, `LUMEN_MANIFEST` points at a specific manifest, and `LUMEN_ARTIFACT_DIR` installs from a local mirror instead of the network.

Supported Host targets are `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `android/arm64` (Termux). Windows and the planned iOS/Android application surfaces are not Host targets.

Maintainers build and pin a release with the reproducible builder, which builds every target twice and refuses to publish unless both builds are byte-identical:

```sh
scripts/lumen-release --version 0.1.0-m1 \
  --hermes-image-ref ghcr.io/<owner>/lumen-hermes@sha256:<digest> --publish
```

It never writes a placeholder identity, so a released manifest always resolves to real, digest-pinned bytes. Publishing the pinned Hermes image requires a token with `write:packages`.

Supervisor definitions are deliberately non-bootstrapping templates. Provision the private state and secrets once, then start the foreground service; never run `init` from a restart hook and never auto-reinitialize an existing state directory.

```sh
install -d -m 700 "$LUMEN_DATA_DIR"
lumen-host init
lumen-host doctor
chmod 600 "$LUMEN_OPERATOR_CREDENTIAL_FILE" "$LUMEN_HERMES_BEARER_FILE" "$LUMEN_HERMES_CA_FILE" "$LUMEN_HERMES_CLIENT_CERT_FILE" "$LUMEN_HERMES_CLIENT_KEY_FILE"
```

For hardened Hermes, set `LUMEN_HERMES_PROFILE=hardened`, the HTTPS base URL, CA/client certificate/key files, bearer file, and pinned server certificate before starting `lumen-host serve`. Docker Compose mounts these files read-only and expects the state volume to have been initialized by a one-time `docker compose run --rm lumen-host init` using the same environment and secret mounts.

## Verified Azure Docker reference

On 2026-09-14 IST / 2026-09-13 UTC, the combined gate passed twice clean on Azure Ubuntu 22.04 amd64 with Docker `29.1.3`, Compose `2.40.3`, and Hermes `v2026.9.7` (`2237be355906fbe6065ce1815711eee52b2d646e`, archive SHA-256 `907c2a72db1c5dd637ea8eeae97f4cb5b32cef615c17258f6b190924ec5bf688`). A real reboot preserved both services, canonical state, and a completed marker task; doctor returned `ready`.

The external proof used a separate Host-only Compose project and independently supervised Hermes `0.21.1` behind pinned mutual TLS. Host lifecycle commands did not change Hermes or its TLS gateway. A real reboot preserved all service identities and canonical state, and doctor returned `ready`. `mise run milestone1-external-check` supplies a repeatable isolated-endpoint lifecycle check. This same-VPS evidence proves ownership separation, not a second-machine deployment or the still-open distributable/live-update gate.

## macOS provisioning

Run the Host directly on darwin rather than in a container. The control socket is the operator channel and must hold an enforceable `0600` mode, but a host-shared bind mount on Docker Desktop for macOS fails `chmod` on a socket with `EINVAL`, so the containerized Host refuses to start instead of weakening that invariant. The combined container topology is therefore a Linux deployment; on macOS keep the pinned Hermes gateway containerized and the Host native.

`mise run milestone1-macos-check` runs that journey end to end against the pinned Hermes gateway: a configured synthetic task completes with durable output, a denied approval fails closed as `failed` with `approval_denied` and no dispatched run, and an explicit cancellation stops an in-flight run to a terminal `cancelled`. The synthetic provider honors `LUMEN_PROVIDER_DELAY_MS`; the gate defaults it to 3000 ms so the cancellation race is deterministic instead of accidental.

For manual development the native Host lifecycle is `scripts/lumen-mac-host start|status|stop|logs`, with `scripts/lumen-mac-test` for a bounded policy-approved Host-to-Hermes round trip.

## Android Termux provisioning

With an authorized Android device connected over ADB, run `scripts/push-termux-host` from the repository root. It runs the Phase 2 gate, configures USB-only `adb reverse` access from device-local port 8642 to the Mac Hermes gateway, and copies the verified ARM64 PIE binary plus installer bundle to `/sdcard/Download/lumen`. For an explicit development-only run, it also stages the existing Mac Hermes bearer token and a development `host.env`; the installer immediately moves both into Termux-private storage and deletes the shared copies. It never copies Host state. Do not use this development handoff for hardened or release evidence.

In Termux, run `termux-setup-storage && sh ~/storage/downloads/lumen/install-lumen-host`. The installer creates private Host and runit directories plus `~/.termux/boot/lumen-host`, which starts `runsvdir -P "$HOME/service"` when the Termux:Boot add-on is installed and enabled. Without Termux:Boot, start it manually with `runsvdir -P "$HOME/service"`. After filling `~/.config/lumen/host.env`, run `. ~/.config/lumen/host.env && lumen-host init && lumen-host doctor` before starting the service. The installer refuses to overwrite initialized Host state. If a prior installer attempt left an uninitialized binary or configuration, rerun it with `LUMEN_REINSTALL=1`; that retry still refuses initialized state. The development bundle provides its own USB-only configuration. For hardened deployment, fill `~/.config/lumen/host.env` and place the referenced Hermes bearer, CA, client certificate, and client-key files in Termux-private storage with mode `0600`; do not copy those hardened credentials through shared storage or ADB.
