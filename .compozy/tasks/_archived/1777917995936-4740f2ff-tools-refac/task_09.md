---
status: completed
title: Session-Bound Autonomy Tools and Claim-Token Hard Cut
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 09: Session-Bound Autonomy Tools and Claim-Token Hard Cut

## Overview

Convert AGH-owned autonomy execution from raw-`claim_token` public contracts to a session-bound surface keyed by `run_id`, while adding the dedicated autonomy tool family the canonical design requires. This is the most invasive hard cut in `tools-refac` because it changes tool, CLI, HTTP, UDS, contract, codegen, docs, and web-generated consumer assumptions in one move.

<critical>
- ALWAYS READ `_techspec.md`, ADR-003, and ADR-005 before touching autonomy contracts
- REFERENCE TECHSPEC sections "Session-Bound Autonomy Lookup", "Data Models", "Post-Implementation Residual Checks", and "E2E Tests"
- FOCUS ON WHAT: converge every AGH-owned autonomy surface on the same authoritative task writers without exposing raw `claim_token`
- MINIMIZE CODE duplication — do not introduce a second lease credential or alternate writer path
- TESTS REQUIRED — stale lease, foreign run, double lease, and redaction behavior must be covered end to end
</critical>

<requirements>
1. MUST add dedicated autonomy tools for `run_claim_next`, `run_heartbeat`, `run_complete`, `run_fail`, and `run_release`.
2. MUST hard-cut AGH-owned CLI, HTTP, UDS, and contract surfaces away from raw `claim_token` arguments and outputs in favor of session-bound lookup plus `run_id`.
3. MUST keep the existing task-service lease writers as the only authoritative writers.
4. MUST preserve redaction rules so raw `claim_token` never appears in logs, tool payloads, contracts, web-generated types, or docs after the cut.
5. MUST co-ship contract/codegen/web fixture updates in the same implementation track.
</requirements>

## Subtasks

- [x] 9.1 Add the autonomy built-in tool family over the current task lease writers
- [x] 9.2 Implement the session-bound lease lookup and `run_id`-driven public contract
- [x] 9.3 Hard-cut CLI, HTTP, UDS, and OpenAPI contracts away from raw `claim_token`
- [x] 9.4 Regenerate and update downstream web/generated task types and fixtures
- [x] 9.5 Add unit, integration, and transport-parity tests for lease safety and redaction

## Implementation Details

See TechSpec sections "Session-Bound Autonomy Lookup", "Existing MCP Config And Auth", "Post-Implementation Residual Checks", and "Implementation Steps". This task is cross-cutting by design and should land as a single coherent contract cut, not a compatibility bridge.

### Relevant Files

- `internal/task/lease.go` — current lease DTOs and raw-token handling that must stop crossing AGH-owned public surfaces
- `internal/task/lease_manager.go` — authoritative lease writer and lookup behavior
- `internal/api/contract/agents.go` — current raw-token public autonomy contracts
- `internal/api/core/agent_tasks.go` — HTTP/UDS-facing task execution handlers
- `internal/cli/task.go` — current CLI claim/heartbeat/complete/fail/release surface
- `internal/api/spec/spec.go` — OpenAPI contract generation for the hard cut

### Dependent Files

- `web/src/generated/agh-openapi.d.ts` — generated frontend contract that must change in the same workstream
- `web/src/systems/tasks/types.ts` — frontend task-run aliases that currently reflect the old contract shape

### Related ADRs

- [ADR-003: Identity-Bound Task Execution Uses Dedicated Agent Tools](adrs/adr-003-identity-bound-autonomy-tools.md)
- [ADR-005: Autonomy Tool Surfaces Are Session-Bound And Never Expose Raw Claim Tokens](adrs/adr-005-session-bound-autonomy-surface.md)

### Web/Docs Impact

- `web/`: `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, `web/src/systems/tasks/mocks/fixtures.ts`, and related tests/stories because the task-run lease contract changes shape.
- `packages/site`: `packages/site/content/runtime/core/autonomy/task-runs-and-leases.mdx`, `packages/site/content/runtime/core/hooks/event-catalog.mdx`, and CLI reference pages under `runtime/cli-reference/task/`, especially `next`, `heartbeat`, `complete`, `fail`, `release`, and `task/run/*`.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: affects hosted MCP exposure, task-run observability, and any extension or hook contract that references AGH-owned autonomy payloads.
- Agent manageability: directly changes the canonical task-execution path across tools, CLI, HTTP, and UDS.
- Config lifecycle: no new top-level config keys expected, but runtime docs and examples for autonomy/leases must change in lockstep with the contract cut.

## Deliverables

- Canonical `agh__autonomy` tool family
- Session-bound public autonomy contract keyed by `run_id` instead of raw `claim_token`
- Regenerated OpenAPI and downstream web task-run type/fixture updates
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for autonomy hard-cut parity and redaction **(REQUIRED)**

## Tests

- Unit tests:
  - [x] session-bound lease lookup rejects foreign run, expired lease, double lease, and missing active lease cases with deterministic reason codes
  - [x] autonomy tools and public contracts never emit raw `claim_token` while preserving `claim_token_hash` for observability-only use
  - [x] task writer calls still route exclusively through the current authoritative lease manager
- Integration tests:
  - [x] tool, CLI, HTTP, and UDS autonomy flows converge on the same lease writers for claim/heartbeat/complete/fail/release
  - [x] regenerated OpenAPI, web task types, and task mocks match the hard-cut contract in the same change
  - [x] transport-parity tests prove AGH-owned surfaces no longer accept or return raw `claim_token`
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- AGH-owned autonomy surfaces stop exposing raw `claim_token`
- Agents can execute task-run lifecycle through dedicated tools and session-bound contracts that still converge on the existing authoritative writers

## References

- `.compozy/tasks/tools-refac/analysis/competitor-tool-surface-notes.md`
- `docs/_memory/lessons/L-004-manual-equals-peer.md`
- `docs/_memory/lessons/L-005-authoritative-primitive-exclusivity.md`
