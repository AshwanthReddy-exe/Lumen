# O-004 Android Host decision

## Decision

The first Host is a native Android application with an app-private encrypted store and one visible foreground service. It is intended for a charged, local-network old phone; it is not an always-on server guarantee.

The service owns Host recovery, the authenticated node channel, and health reporting. It starts only through Android-supported paths, restores durable state before accepting work, and reports `offline` or `degraded` rather than pretending that queued or interrupted work completed. The companion UI binds to that service; closing the UI does not change Space authority.

Device-key and encrypted-store access remain Android adapters. Portable Space, policy, protocol, and task rules do not depend on Android APIs.

## Boundaries

- The Host requires a visible foreground notification while actively coordinating the Space.
- Boot recovery is best effort: it restores state and reports availability, but does not bypass OS restrictions or claim uninterrupted service.
- A low battery, storage pressure, thermal limit, lost network, stopped service, or unavailable key moves the Host to a visible degraded state and blocks new cross-node work when safety requires it.
- Local node work may continue only under an unexpired cached grant; it queues its permitted delta until the Host returns.
- No microphone, camera, location, overlay, accessibility, or battery-optimization exemption is required for the initial Host.

## Non-goals

- Unattended server-grade uptime, automatic Host failover, remote wake-up, and OEM-specific battery-management workarounds.
- Running Hermes or arbitrary tools on the Android Host. Hermes remains optional on capable nodes.

## Acceptance checks

1. On the intended old phone, create a Space, background the companion, and confirm the foreground notification, health state, and authenticated local-node channel remain honest.
2. Restart the process and device during an active task; confirm durable tasks recover or end as `unknown_outcome`, never `completed` without evidence.
3. Simulate low storage, unavailable encryption key, battery/thermal warning, and network loss; confirm new cross-node dispatch is blocked or queued with an explained state.
4. Stop the foreground service explicitly; confirm nodes report the Host unavailable and no cross-node action is accepted.
