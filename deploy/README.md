# Host provisioning

Supervisor definitions are deliberately non-bootstrapping templates. Provision the private state and secrets once, then start the foreground service; never run `init` from a restart hook and never auto-reinitialize an existing state directory.

```sh
install -d -m 700 "$LUMEN_DATA_DIR"
lumen-host init
chmod 600 "$LUMEN_OPERATOR_CREDENTIAL_FILE" "$LUMEN_HERMES_BEARER_FILE" "$LUMEN_HERMES_CA_FILE" "$LUMEN_HERMES_CLIENT_CERT_FILE" "$LUMEN_HERMES_CLIENT_KEY_FILE"
```

For hardened Hermes, set `LUMEN_HERMES_PROFILE=hardened`, the HTTPS base URL, CA/client certificate/key files, bearer file, and pinned server certificate before starting `lumen-host serve`. Docker Compose mounts these files read-only and expects the state volume to have been initialized by a one-time `docker compose run --rm lumen-host init` using the same environment and secret mounts.
