# Lumen

Lumen is a proprietary, paid personal AI **Space** inspired by the continuity, presence, and practical agency of Jarvis. A person talks to one Lumen from the web, messaging, voice, or a paired device; Lumen preserves the relationship and safely coordinates permitted work across their devices and services.

Lumen is not a device assistant or a collection of disconnected apps. The Space owns identity, conversations, memory, policy, approvals, task truth, and portability. Hermes supplies the agent runtime and mature intelligence capabilities behind Lumen-owned contracts.

## Product model

- **Space:** the user's continuous AI identity, canonical conversations, accepted memory, policy, grants, tasks, automations, audit, and node registry.
- **Host:** the one active V1 authority. It stores canonical Space state, selects eligible nodes, authorizes cross-node work, dispatches invocations, and records honest outcomes.
- **Node:** a paired computer, phone, server, or service endpoint that advertises narrowly typed capabilities and revalidates authorization locally.
- **Capability:** a separately permissioned action such as `files.search`, `files.read`, `notification.deliver`, or `shell.execute`. Scope may restrict action, resource, folder, application, context, data class, cost, target, duration, or approval behavior.
- **Surface:** an interface to the same Space. The first product surfaces are a responsive web application, one Hermes-backed messaging gateway, and one lightweight capability node.
- **Companion:** an optional expressive surface, such as a desk display or desktop presence. A companion never owns canonical authority.

The Host can execute work on nodes only through granted capabilities. Access to one folder never implies access to the device, other folders, credentials, or unrelated tools. Discovery reports what a node could do; it never grants permission.

## Product boundary

Lumen owns the differentiated product layer: Space authority, durable conversations and memory, node coordination, permissions and approvals, task lifecycle, receipts, deployment, migration, and user experience.

Hermes is the preferred capability platform for reasoning, model routing, tools, skills, MCP, browser use, messaging, voice orchestration, vision, delegation, automation, and remote execution. Lumen adopts or extends Hermes before building a parallel subsystem. Every imported capability remains behind Lumen policy, credentials, lifecycle, audit, and rollback controls; Hermes is never the Space authority.

Lumen's product is proprietary and commercial. Managed hosting and paid self-hosting share the same Space contracts and encrypted export format. Public interoperability consists of versioned node and capability protocols, integration contracts, an SDK, fixtures, and compatibility rules—not the proprietary product implementation.

## Deployment profiles

- **Combined:** Host, encrypted Space storage, Hermes, gateways, and workers are separately supervised on one machine or VPS.
- **Separated:** Host and canonical storage stay together while Hermes or heavy workers run elsewhere over pinned authenticated transport.
- **Managed:** Lumen provisions a dedicated isolated data plane for each customer, with a shared content-minimizing control plane.
- **Hybrid:** authorized nodes perform suitable work locally and synchronize only permitted outcomes or context.

Topology does not determine assurance. Development, personal, and hardened profiles state their isolation and evidence explicitly.

## Repository and current status

```text
cmd/                native Go Host and CLI executables
internal/           Space core, Host composition, storage, setup, control, Hermes adapter
protocol/           versioned schemas and cross-language fixtures
apps/               optional native surfaces and capability nodes
test/               contract, integration, security, recovery
deploy/             service-manager and container configuration
```

The production Space authority and Host are native Go. The repository already contains substantive encrypted persistence, authority and task lifecycle, local operator control, a constrained Hermes Runs adapter, setup planning/artifact/configuration components, and supervision definitions. The Android application is a disabled companion shell.

The complete product does **not** exist yet. In particular, the public setup/doctor/service orchestration, canonical conversation and memory experience, web application, authenticated node transport, restricted file node, messaging continuity, Jarvis-like voice presence, managed service, and public SDK remain planned work. Platform examples such as macOS, Linux/VPS, Android Termux, and iPhone are evidence targets, not the roadmap's organizing principle.

Run the current contract baseline with `mise run phase0-check`. See the canonical [product requirements](./docs/PRD.md), [architecture](./docs/ARCHITECTURE.md), [decisions](./docs/DECISIONS.md), [delivery plan](./docs/PLAN.md), and [design principles](./docs/DESIGN-PRINCIPLES.md).
