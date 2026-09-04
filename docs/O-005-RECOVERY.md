# O-005: Minimal Host recovery decision

Status: accepted as `D-024`; implementation is deferred to the recovery phase.

## Decision

V1 supports an owner-initiated, manual encrypted backup and an explicit Host migration. There is exactly one active Host for a Space. A backup is data, not authority: exporting or restoring it never activates a second Host.

The Host is the authority for the current Space epoch, task state, node membership, capability grants, and revocations. A migration advances the epoch and records the new Host identity. Commands from an older epoch are rejected, including commands produced by a Host that was offline during migration.

## Backup and key handling

- The owner starts export from the active Host and explicitly chooses the destination. The export contains the minimum durable Space state needed for recovery: protocol version, schema version, Space identity, node/grant state, task/event state, and migration metadata.
- Secrets and sensitive state are encrypted before leaving the Host. The export is never written to ordinary logs or included in diagnostics.
- Encryption keys remain under owner-controlled recovery material and platform-protected key storage where available. V1 does not recover a lost key, escrow it remotely, or invent a provider-specific cryptographic scheme.
- Export reports a durable result (`created`, `failed`, or `unknown`) and an integrity check. An `unknown` result is not treated as a successful backup.

## Restore and migration

1. Verify the export's integrity, format version, Space identity, and decryption capability on the candidate device.
2. Restore into an inactive/quarantined installation. Do not accept commands, advertise as Host, or issue grants during validation.
3. Require explicit owner confirmation naming the candidate as the Host. Record a new epoch and candidate identity durably before enabling Host traffic.
4. Mark the previous Host retired. It must stop accepting new work and reject its old epoch. If it later reconnects, it may recover state as a node only through explicit re-pairing.
5. Reconcile task and node status from the durable event state. In-flight work remains `unknown` until the target reports a result; it is not silently repeated.

For migration, the source first enters `draining` and rejects new work while completing or cancelling local mutations. The source and candidate exchange a verified handoff record, then the candidate commits the next epoch. If the source cannot confirm the handoff, the candidate stays inactive and the source remains authoritative. If authority transfer is committed, rollback means restoring the latest valid export as a new explicit migration; it never reactivates the old epoch.

## Failure behavior and non-goals

Corrupt, incompatible, undecryptable, or identity-mismatched exports are rejected without modifying the active installation. A failed restore leaves the candidate quarantined and the source unchanged. Device loss is recoverable only from a valid export and recovery material. V1 does not provide automatic cloud backup, live multi-master replication, transparent failover, background migration, or conflict-free merging of independently active Hosts.

## Acceptance checks

- A valid export restores on a clean candidate only after owner confirmation and produces exactly one active epoch.
- Two Hosts cannot accept commands for the same Space epoch; old-epoch commands are rejected after migration.
- Interrupted migration before transfer leaves the source active; interruption after transfer leaves only the new epoch active.
- Corrupt, incompatible, undecryptable, and wrong-Space exports cause no active-store mutation.
- Lost recovery material produces a clear unrecoverable outcome without weakening encryption.
- Restart/reconnect reconciles durable tasks and reports in-flight work as `unknown` rather than duplicating it.
