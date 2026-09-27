# Context

## Engineering principles
- source: docs/DESIGN-PRINCIPLES.md
- note: Lumen is a private, multi-device Space. A device is only a node; web, messaging, voice, native apps, and companions are only surfaces. The active Host owns canonical conversations, accepted memory, policy, routing, task truth, and audit history.

## Failure and operations principles
- source: docs/DESIGN-PRINCIPLES.md
- note: Process death, restart, duplicated or reordered messages, stale grants, clock changes, offline nodes, partial writes, unavailable Hermes, network loss, and dependency upgrades are normal states.

## Presence and interoperability principles
- source: docs/DESIGN-PRINCIPLES.md
- note: Lumen's proprietary product may provide paid managed and self-hosted experiences while publishing the protocols and SDK needed to build compatible nodes and integrations.

## Host and Hermes historical closure
- source: docs/PHASE-2-HOST-HERMES.md
- note: The following list records how the original Host-first implementation was sliced. It is historical branch evidence, not the active product roadmap; [PLAN.md](./PLAN.md) owns current sequencing.

## Current main and draft PR boundary
- source: docs/PLAN.md
- note: [PR #20](https://github.com/AshwanthReddy-exe/Lumen/pull/20) is draft on `feat/lumen-continuity`, 26 committed changes ahead of `main` at `0b67f81` when inspected. It includes conversation/memory and bounded node-status work but retains open live gates; its separate worktree also has uncommitted edits. The older `feat/m2-conversation-nucleus` work is incorporated there and is not an independent completion claim.
