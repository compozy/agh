---
status: completed
title: Registry Indexing, Toolsets, Policy, and Projections
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
---

# Task 03: Registry Indexing, Toolsets, Policy, and Projections

## Overview

Build the executable registry index and effective policy engine used by every backend and surface. This task turns providers into operator-visible and session-callable projections while enforcing canonical IDs, collision rules, toolsets, ACP ceilings, source policy, and child-session lineage.

<critical>
- ALWAYS READ `_techspec.md`, ADR-003, ADR-005, ADR-006, and ADR-007 before changing registry policy
- DO NOT let `approve-all` bypass explicit denies, unavailable backends, source grants, lineage, conflicts, or hooks
- DO NOT hide unavailable or unauthorized tools from operator projections
- TESTS REQUIRED: policy tests must prove deny, source, lineage, and projection behavior independently
</critical>

<requirements>
1. MUST aggregate providers into a registry index with deterministic ordering and collision detection.
2. MUST fail closed on canonical ID collisions, sanitized external-name collisions, and `id_too_long`.
3. MUST expand toolsets into concrete ToolID atoms with cycle detection and deterministic errors.
4. MUST evaluate ACP permission mode as a ceiling above registry grants.
5. MUST produce separate operator-visible and session-callable projections with reason codes.
6. MUST enforce child-session subset validation using resolved concrete ToolID atoms.
</requirements>

## Subtasks
- [x] 3.1 Implement provider registration, indexing, sorting, and collision detection
- [x] 3.2 Implement toolset expansion, pattern validation, and cycle detection
- [x] 3.3 Implement effective policy evaluation from config, agent, session, source, and ACP ceiling
- [x] 3.4 Implement operator and session projections with availability/authorization reason codes
- [x] 3.5 Enforce session lineage and child-session subset constraints with concrete ToolID atoms
- [x] 3.6 Add focused tests for collisions, source grants, deny precedence, ACP ceiling, and projection differences

## Implementation Details

Use TechSpec "Integration Points", "Agent Manageability", "Safety Invariants", and ADR-006 for projection rules. Keep this task focused on decisions and projections; actual invocation happens in task_04 and backend providers arrive in later tasks.

### Relevant Files
- `internal/tools/registry*.go` - new registry index and provider aggregation
- `internal/tools/policy*.go` - effective policy evaluator
- `internal/tools/projection*.go` - operator/session projections
- `internal/store/session_lineage.go` - child-session tool subset validation
- `internal/acp/permission.go` - ACP permission mode ceiling inputs

### Dependent Files
- `internal/daemon/*` - later composition root wires providers into the registry
- `internal/api/core/handlers.go` - task_11 injects registry interfaces into handlers
- `internal/api/contract/` - task_11 exposes projection DTOs
- `web/src/systems/tools/**` - task_13 consumes projection reason codes

### Related ADRs
- [ADR-003: Runtime Registry Package Boundary](adrs/adr-003-runtime-registry-package-boundary.md) - defines ownership boundaries
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - defines approval ceiling and policy layering
- [ADR-006: Tool Visibility by Surface](adrs/adr-006-tool-visibility-by-surface.md) - defines operator vs session visibility
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - defines collision-safe identity

### Web/Docs Impact
- `web/`: task_13 must render projection states and reason codes through `web/src/systems/tools/**` and generated API types.
- `packages/site`: task_14 must document projection semantics, policy precedence, toolsets, `deny_tools`, and child-session lineage.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: centralizes policy for native, extension-host, MCP, hooks, skills, network, and task tools.
- Agent manageability: establishes structured state that later CLI/HTTP/UDS surfaces expose to agents.
- Config lifecycle: consumes `tools.policy`, agent `tools`, agent `toolsets`, agent `deny_tools`, and `trusted_sources` from task_02.

## Deliverables
- Registry indexing and projection package behavior
- Effective policy evaluator with ACP ceiling and source grants
- Toolset expansion and child-session lineage enforcement
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for realistic provider aggregation and session projection **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Duplicate canonical IDs and sanitized MCP-name collisions fail closed
  - [x] Toolset cycles, unknown members, and invalid patterns return deterministic errors
  - [x] Explicit denies override allows, toolsets, trusted sources, and `approve-all`
  - [x] `approve-reads` does not approve untrusted external read-only tools without source or tool grants
  - [x] Child-session permissions cannot exceed parent concrete ToolID atoms after toolset expansion
- Integration tests:
  - [x] Operator projection includes unavailable, unauthorized, and conflicted tools with reason codes
  - [x] Session projection exposes only callable tools for the effective session
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Every backend can rely on one registry policy engine instead of implementing local gates
- Operator and session projections intentionally differ and are both deterministic
