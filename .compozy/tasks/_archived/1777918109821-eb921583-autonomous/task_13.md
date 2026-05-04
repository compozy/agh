---
status: completed
title: Safe Spawn API CLI And Reaper
type: backend
complexity: critical
dependencies:
  - task_12
---

# Task 13: Safe Spawn API CLI And Reaper

## Overview
Add the safe autonomous spawn surface. Agents may create bounded child sessions through first-class API/CLI commands, while the daemon enforces lineage, TTL, max-child caps, permission narrowing, and cleanup through a reaper.

<critical>
- ALWAYS READ `_techspec.md`, ADR-006, ADR-009, ADR-010, and ADR-011 before adding spawn behavior
- CHILD PERMISSIONS MUST BE A SUBSET OF THE PARENT - unknown permission atoms must be rejected
- COORDINATORS CANNOT SPAWN COORDINATORS IN MVP
- TTL AND REAPER BEHAVIOR MUST RELEASE ACTIVE LEASES STRUCTURALLY
- TESTS REQUIRED - permission comparator, caps, TTL, parent stop, reaper, and lease release must be covered
- NO WORKAROUNDS - do not rely on prompt instructions alone for permission safety
</critical>

<requirements>
- MUST add daemon/session spawn APIs using typed `SpawnOpts` and lineage metadata from task_12.
- MUST add agent-facing UDS endpoint and `agh spawn` CLI command with stable JSON output.
- MUST enforce max children per parent, max depth, mandatory TTL, workspace bounds, and permission narrowing.
- MUST implement a context-owned reaper that stops expired/orphaned children and releases their active task leases with structured reasons.
- MUST emit typed `spawn.*` hooks through task_03 where externally meaningful.
- MUST preserve manual operator-created sessions and existing session creation paths.
</requirements>

## Subtasks
- [x] 13.1 Add `SpawnOpts`, permission atom model, and subset comparator.
- [x] 13.2 Implement daemon/session manager spawn path with lineage, TTL, cap, and workspace validation.
- [x] 13.3 Add `/agent/spawn` UDS endpoint and `agh spawn` CLI command.
- [x] 13.4 Add reaper lifecycle for TTL expiry, parent stop, and orphan cleanup.
- [x] 13.5 Release active task leases through task service APIs when spawned sessions are stopped by the reaper.
- [x] 13.6 Add tests for spawn success, denial, reaper cleanup, lease release, hooks, and operator regression.

## Implementation Details
Spawn is a safety boundary, not a convenience wrapper around `Manager.Create`. The permission comparator should be a real typed model with known atoms. Unknown atoms must fail closed.

The reaper must be owned by daemon lifecycle with explicit context cancellation and wait-group shutdown. It should stop children and release leases through existing service methods, not direct database updates.

### Relevant Files
- `internal/session/manager.go` - spawn creation and child session tracking.
- `internal/session/session.go` - session roles and lineage from task_12.
- `internal/daemon/daemon.go` - reaper composition root and shutdown.
- `internal/api/udsapi/routes.go` - spawn route registration.
- `internal/api/core/interfaces.go` - session/spawn interface additions.
- `internal/cli/session.go` or `internal/cli/root.go` - command tree precedent.
- `internal/cli/client.go` - UDS spawn client method.
- `internal/task/manager.go` - active lease release when spawned sessions stop.
- `internal/hooks/payloads.go` - `spawn.*` hook payloads.
- `.resources/hermes/agent/auxiliary_client.py` - reference for delegated auxiliary agents.
- `.resources/hermes/mini_swe_runner.py` - reference for bounded subprocess/agent execution.
- `.resources/paperclip/doc/plans/2026-02-19-ceo-agent-creation-and-hiring.md` - reference for agent creation policy boundaries.

### Dependent Files
- `internal/daemon/*coordinator*` - task_14 uses safe spawn to delegate work.
- `packages/site/content/runtime/cli-reference/` - task_16 documents `agh spawn`.
- `.compozy/tasks/autonomous/qa/test-cases/` - task_17 plans spawn safety QA.

### Related ADRs
- [ADR-006: Safe Spawn With Lineage And Permission Narrowing](adrs/adr-006.md) - spawn safety contract.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - `spawn.*` hooks.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - manual session creation remains.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - CLI/docs/contract discipline.

## Deliverables
- Safe spawn API and CLI.
- Permission narrowing comparator and policy validation.
- TTL/parent-stop reaper that releases active leases safely.
- Unit tests with 80%+ coverage for spawn validation and permission comparator **(REQUIRED)**.
- Integration tests for real child session lifecycle and lease cleanup **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Permission comparator accepts exact/subset permissions and rejects superset or unknown atoms.
  - [x] Spawn validation rejects coordinator-from-coordinator, missing TTL, over-depth, over-child-cap, and cross-workspace requests.
  - [x] CLI/handler validation returns structured errors without leaking internal policy state.
  - [x] Reaper classifies TTL expiry, parent stop, and orphan cleanup with structured reasons.
  - [x] `spawn.*` payload conversion includes parent/root/depth without raw secrets.
- Integration tests:
  - [x] A parent session spawns a child with narrowed permissions and durable lineage.
  - [x] Stopping the parent stops children and releases any active task lease through the task service.
  - [x] TTL expiry stops child sessions without fire-and-forget goroutines.
  - [x] Manual operator session creation remains available and is not subject to child-only caps.
  - [x] Generated contracts/web checks pass if spawn DTOs are public.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Agents can delegate through bounded child sessions instead of ungoverned process creation.
- Spawn safety is enforced by daemon code, not prompts.
