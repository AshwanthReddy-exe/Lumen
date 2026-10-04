---
phase: 02-close-the-existing-host-hermes-foundation
plan: "08"
subsystem: host-hermes-provider
tags: [hermes, nvidia-nim, provider, evidence]
requires:
  - phase: 02-close-the-existing-host-hermes-foundation
    provides: Host task submission and pinned Hermes adapter
provides:
  - Redacted evidence for one real NVIDIA NIM Host-mediated response
  - Honest separation of AgentRouter authentication failure and unknown task outcome
affects: [phase-02-runtime-qualification]
actuals:
  tasks: 2
  commits: 0
tech-stack:
  added: []
  patterns: [record provider outcomes from durable Host receipts without storing prompts, responses, or credentials]
key-files:
  created: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-08-EVIDENCE.md]
  modified: [.planning/phases/02-close-the-existing-host-hermes-foundation/02-08-EVIDENCE.md]
key-decisions:
  - "Keep NVIDIA NIM nvidia/nemotron-3.5-lightning-30b-a3b as the local Hermes default; do not fall back to AgentRouter."
  - "Treat the NIM success, AgentRouter HTTP 401, and AgentRouter task_unknown as separate outcomes."
requirements-completed: []
coverage:
  - id: P02-08-NIM-HOST-RESULT
    description: A Host-mediated task on the pinned NIM profile reached durable completion with non-empty output.
    requirement: FR-38
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-08-EVIDENCE.md; pinned Hermes 0.21.1 receipt, one-time approval, and durable completed status
        status: pass
    human_judgment: true
    rationale: The response body was deliberately discarded; this establishes provider-backed completion and persistence, not answer quality.
  - id: P02-08-HONEST-FAILURE
    description: The prior AgentRouter authentication failure remains a durable failed Host task and the later unrecoverable attempt remains unknown.
    requirement: FR-41
    verification:
      - kind: other
        ref: .planning/phases/02-close-the-existing-host-hermes-foundation/02-08-EVIDENCE.md; separate AgentRouter receipts and task_unknown record
        status: pass
    human_judgment: true
    rationale: Historical outcomes remain distinct; neither is relabeled as success or used to overwrite the NIM result.
status: complete
completed: 2026-10-02
---

# Phase 02 Plan 08: Qualify the NVIDIA NIM Host result

One harmless request through the pinned Hermes 0.21.1 profile using NVIDIA NIM model `nvidia/nemotron-3.5-lightning-30b-a3b` reached durable Host completion with non-empty output. The response content was discarded and quality was not assessed. The earlier AgentRouter HTTP 401 remains an honest failed task; the later AgentRouter `task_unknown` attempt remains unrecoverable and was not retried. Credential values and fingerprints were not retained.

This closes only Plan 02-08's provider-result scope. It does not certify the current unpinned Hermes 0.20.6 service, NIM lifecycle recovery, tool/file containment, provider egress restrictions, or Phase 02 as a whole.
