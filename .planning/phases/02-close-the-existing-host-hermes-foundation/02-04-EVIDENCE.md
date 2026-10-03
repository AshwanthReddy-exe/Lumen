# Phase 02 plan 04 evidence — external Hermes across machines

## Automated adoption and denial coverage

Observed 2026-09-28 on the current worktree at `ff12f4c1b0644f73dd2488f014150d1b232ee241` (worktree had uncommitted changes). The plan's focused command passed outside the sandbox:

```text
rtk go test ./internal/setup ./test/contract -run 'TestExternal|TestAdopt' -count=1
Go test: 38 passed in 2 packages
```

The initial sandboxed attempt failed before assertions because the test fixtures create temporary directories under the account home. This is control-flow, certificate/pin, capability/version, and adoption-state regression evidence from tests; it is not evidence of a real Hermes Run or a cross-machine connection.

## Live A/B gate

**Not run; unverified.** The available `scripts/lumen-external-check` creates a Python compatibility endpoint and is explicitly synthetic. It cannot satisfy this plan's live gate.

The live gate still needs an independently reachable pinned Hermes B, its independent supervisor, safe mutual-TLS/client identity and scoped bearer material, a configured provider for one harmless real task, and operator-observable B identity/run evidence. The VPS SSH connection currently times out during banner exchange; the last successful remote observation found its owner gateway on an unrelated upstream revision and no Runs API listener. No Lumen Run, model request, remote restart, or remote file/secret access was performed in this session.

Acceptance remains open until a real Host A adoption completes one Host-authorized Hermes Run with durable Host outcome, the independently supervised B identity is unchanged across an A-only restart, and each planned pin/auth/version/binding negative case is rejected before Run creation.

## Cross-machine probe implementation

Updated 2026-10-02. Added executable `scripts/lumen-cross-machine-check` with separate `snapshot-b` and `run-a` modes. B snapshots capture a hashed machine identity, expected Hermes source commit/version, config digest, systemd invocation/PID/state, and pinned endpoint certificate digest. A builds the current Host locally, adopts Hermes through the existing external Compose path, records one real task/Run and durable Host output, restarts only its own Host, then compares operator-copied B snapshots. Temporary Compose project/data are uniquely scoped and removed on exit. Secrets remain file inputs and are not printed.

Local verification:

```text
sh -n scripts/lumen-cross-machine-check                  # passed
GOCACHE=/private/tmp/lumen-phase02-gocache TMPDIR=/private/tmp \
  rtk go test ./test/contract -run '^TestCrossMachineProbeContract$' -count=1
Go test: 1 passed in 1 packages
graphify update .
Code graph updated; 2290 nodes, 5289 edges, 142 communities
```

An independent read-only security review found and we fixed TLS leaf-certificate extraction/IPv6 parsing, snapshot field validation, and expected commit/pin validation. The final review found no remaining blocker in the probe's scope. These checks validate shell syntax and the script's static contract only. No B snapshot, Lumen setup, Run, Host restart, or live negative swap was performed. The A/B acceptance gate remains open. The probe's machine/supervisor snapshots are operator-observed assertions, not remote attestation.
