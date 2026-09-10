# Host provisioning

Supervisor definitions are deliberately non-bootstrapping templates. Provision the private state and secrets once, then start the foreground service; never run `init` from a restart hook and never auto-reinitialize an existing state directory.

```sh
install -d -m 700 "$LUMEN_DATA_DIR"
lumen-host init
lumen-host doctor
chmod 600 "$LUMEN_OPERATOR_CREDENTIAL_FILE" "$LUMEN_HERMES_BEARER_FILE" "$LUMEN_HERMES_CA_FILE" "$LUMEN_HERMES_CLIENT_CERT_FILE" "$LUMEN_HERMES_CLIENT_KEY_FILE"
```

For hardened Hermes, set `LUMEN_HERMES_PROFILE=hardened`, the HTTPS base URL, CA/client certificate/key files, bearer file, and pinned server certificate before starting `lumen-host serve`. Docker Compose mounts these files read-only and expects the state volume to have been initialized by a one-time `docker compose run --rm lumen-host init` using the same environment and secret mounts.

## Android Termux provisioning

With an authorized Android device connected over ADB, run `scripts/push-termux-host` from the repository root. It runs the Phase 2 gate, configures USB-only `adb reverse` access from device-local port 8642 to the Mac Hermes gateway, and copies the verified ARM64 PIE binary plus installer bundle to `/sdcard/Download/lumen`. For an explicit development-only run, it also stages the existing Mac Hermes bearer token and a development `host.env`; the installer immediately moves both into Termux-private storage and deletes the shared copies. It never copies Host state. Do not use this development handoff for hardened or release evidence.

In Termux, run `termux-setup-storage && sh ~/storage/downloads/lumen/install-lumen-host`. The installer creates private Host and runit directories plus `~/.termux/boot/lumen-host`, which starts `runsvdir -P "$HOME/service"` when the Termux:Boot add-on is installed and enabled. Without Termux:Boot, start it manually with `runsvdir -P "$HOME/service"`. After filling `~/.config/lumen/host.env`, run `. ~/.config/lumen/host.env && lumen-host init && lumen-host doctor` before starting the service. The installer refuses to overwrite initialized Host state. If a prior installer attempt left an uninitialized binary or configuration, rerun it with `LUMEN_REINSTALL=1`; that retry still refuses initialized state. The development bundle provides its own USB-only configuration. For hardened deployment, fill `~/.config/lumen/host.env` and place the referenced Hermes bearer, CA, client certificate, and client-key files in Termux-private storage with mode `0600`; do not copy those hardened credentials through shared storage or ADB.
