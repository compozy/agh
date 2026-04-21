---
status: pending
title: Provider-Aware Session Resolution and Validation
type: backend
complexity: medium
dependencies: []
---

# Task 01: Provider-Aware Session Resolution and Validation

## Overview

Implement the config-layer helper that resolves an agent for one session with an optional provider override. This task establishes the runtime semantics for the whole feature: the selected session provider becomes canonical, provider-owned runtime fields are re-resolved coherently, and validation stays scoped to the resolved workspace config instead of daemon-global assumptions.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Core Interfaces", "Data Models", "Testing Approach", and "Technical Considerations"
- KEEP SESSION RUNTIME RESOLUTION INSIDE `internal/config` - do not spread provider override rules across `internal/session`, `internal/api`, and callers
- `ResolvedAgent.Provider` MUST BECOME THE CANONICAL FIELD - downstream code must not recompute the effective provider ad hoc
- TESTS REQUIRED - provider override semantics are subtle and must land with precise unit coverage before lifecycle wiring begins
- GREENFIELD: nao aceitar misturas parciais de provider novo com `command` ou `model` herdados do provider antigo
</critical>

<requirements>
- MUST add `Config.ResolveSessionAgent(agent AgentDef, providerOverride string) (ResolvedAgent, error)` as the single session-aware resolution entrypoint
- MUST preserve current `ResolveAgent` behavior when `providerOverride == ""`
- MUST clone the input `AgentDef`, overwrite `Provider`, clear explicit `Command`, and clear explicit `Model` before reusing the shared provider resolution path when `providerOverride != ""`
- MUST ensure the selected provider owns `provider`, `command`, `default_model`, and the provider-owned MCP layer while preserving global and agent-local layers
- MUST return the effective provider in `ResolvedAgent.Provider` for all successful resolutions, including the no-override path
- SHOULD keep provider validation compatible with workspace-merged config resolution so later session lifecycle code can validate against `spec.workspace.Config`
</requirements>

## Subtasks
- [ ] 1.1 Add the session-aware resolution helper under `internal/config`
- [ ] 1.2 Implement override semantics for `provider`, `command`, `model`, and provider-owned MCP layers
- [ ] 1.3 Make `ResolvedAgent.Provider` explicit and canonical across the resolution result
- [ ] 1.4 Add focused unit coverage for no-override, override, and invalid-provider cases
- [ ] 1.5 Document any helper-level invariants needed by later session and API tasks

## Implementation Details

See TechSpec "Core Interfaces", "Testing Approach", and ADR-001 / ADR-002. The key design constraint is coherence: once a session chooses a different provider, the runtime must resolve as if that provider had been selected originally, not as a hybrid of two providers.

### Relevant Files
- `internal/config/agent.go` - current agent resolution path and natural home for the new helper
- `internal/config/provider.go` - built-in provider registry and provider-owned defaults
- `internal/config/agent_test.go` - existing agent resolution tests to extend with session-aware cases
- `internal/config/provider_test.go` - natural place for provider-owned runtime invariants and layer-merge coverage

### Dependent Files
- `internal/session/manager_start.go` - later tasks must call the new helper during create/resume runtime preparation
- `internal/session/session.go` - later tasks will persist the canonical provider returned here
- `internal/api/core/workspaces.go` - later tasks will surface workspace-visible provider options derived from the same config model
- `.compozy/tasks/session-driver-override/task_02.md` - depends on this task for lifecycle plumbing semantics

### Related ADRs
- [ADR-001: Model Session Driver Selection As A Provider Override](adrs/adr-001.md) - defines the override boundary
- [ADR-002: Re-Resolve Provider-Owned Runtime Fields On Session Override](adrs/adr-002.md) - defines how provider-owned fields must be re-resolved

## Deliverables
- `ResolveSessionAgent` implemented under `internal/config`
- Canonical `ResolvedAgent.Provider` semantics for both default and override resolution paths
- Provider override behavior that re-resolves `command`, `model`, and provider-owned MCP state coherently
- Focused unit coverage for override and validation semantics **(REQUIRED)**
- Regression protection against mixed-runtime resolution bugs **(REQUIRED)**
- Test coverage >=80% for the touched `internal/config` package(s) **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] No override preserves current `ResolveAgent` behavior exactly
  - [ ] Override swaps provider and re-resolves `command` plus default model from the selected provider
  - [ ] Override clears explicit agent `Command` and `Model` influence before provider resolution
  - [ ] Provider-owned MCP layers are replaced while global and agent-local layers remain intact
  - [ ] Unknown or unavailable provider override returns a descriptive validation error
- Integration tests:
  - [ ] Session-oriented callers can reuse the helper without needing separate provider recomputation logic
  - [ ] Workspace-merged config remains the effective input to provider resolution rather than daemon-global config
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- AGH has one canonical session-aware provider resolution path
- A session-level provider override produces a coherent runtime instead of a mixed provider state
