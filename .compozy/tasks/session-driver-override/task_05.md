---
status: pending
title: Workspace Provider Catalog and Automatic Creator Defaults
type: backend
complexity: medium
dependencies:
  - task_02
---

# Task 05: Workspace Provider Catalog and Automatic Creator Defaults

## Overview

Publish the provider options that are actually visible in a resolved workspace and make all automatic internal session creators explicitly opt into agent defaults by passing an empty provider. This task gives the web dialog the provider picker data it needs while keeping non-interactive runtime-created sessions deterministic and unchanged in this first cut.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and task_02 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Core Interfaces", "API Endpoints", "Testing Approach", and "Technical Considerations"
- PROVIDER DISCOVERY MUST BE WORKSPACE-SCOPED - do not invent a daemon-global `/api/providers` surface
- AUTOMATIC CREATORS MUST BE EXPLICIT - every non-interactive session creator should pass empty provider intentionally, not by omission or hidden defaults
- KEEP THE WEB CONTRACT SIMPLE - `WorkspaceDetailPayload` should provide sorted provider options directly for the dialog
- GREENFIELD: nao aceitar inferencia de provider no frontend quando o backend ja conhece o workspace resolvido
</critical>

<requirements>
- MUST add `SessionProviderOptionPayload` and expose `providers []SessionProviderOptionPayload` on `contract.WorkspaceDetailPayload`
- MUST build the workspace-visible provider option list from the resolved workspace config used by session creation
- MUST return provider options in a stable, deterministic order suitable for UI rendering
- MUST keep HTTP/UDS workspace detail handlers and conversions in sync with the new field
- MUST update all non-interactive internal session creators to pass `Provider: ""` explicitly for agent-default runtime selection
- MUST cover at least `internal/automation/dispatch.go`, `internal/daemon/task_runtime.go`, `internal/memory/consolidation/runtime.go`, `internal/api/core/network_details.go`, and `internal/extension/host_api_bridges.go`
- SHOULD ensure no automatic creator accidentally inherits stale provider state from copied `CreateOpts`
</requirements>

## Subtasks
- [ ] 5.1 Add workspace provider option payload types and workspace detail conversion support
- [ ] 5.2 Build sorted provider-option assembly from workspace-merged config
- [ ] 5.3 Extend workspace detail handlers/tests to expose provider options
- [ ] 5.4 Update automatic internal session creators to pass empty provider explicitly
- [ ] 5.5 Add coverage for provider-option ordering and creator-default behavior

## Implementation Details

See TechSpec "API Endpoints", "Testing Approach", and ADR-004. The backend should remain the source of truth for which providers are valid in a workspace; the web client should receive a ready-to-render option list instead of re-deriving it from scattered config assumptions.

### Relevant Files
- `internal/api/contract/contract.go` - workspace detail payload definitions
- `internal/api/core/workspaces.go` - workspace detail assembly and handler logic
- `internal/api/core/conversions_parsers_test.go` - natural place for sorted payload conversion coverage
- `internal/api/core/session_workspace_internal_test.go` - workspace/session handler edge cases
- `internal/automation/dispatch.go` - automatic task-dispatch session creation
- `internal/daemon/task_runtime.go` - runtime-created sessions that should stay on agent defaults
- `internal/memory/consolidation/runtime.go` - non-interactive session creation path for memory consolidation
- `internal/api/core/network_details.go` - internal network/session creator path
- `internal/extension/host_api_bridges.go` - bridge helpers that may construct sessions internally
- `internal/memory/consolidation/runtime_test.go` - creator call-shape coverage
- `internal/daemon/task_runtime_test.go` - automatic creator coverage
- `internal/daemon/daemon_integration_test.go` - runtime session creation integration coverage

### Dependent Files
- `web/src/systems/workspace/types.ts` - task_06 consumes the workspace provider options
- `web/src/systems/workspace/adapters/workspace-api.ts` - task_06 maps the new payload field
- `web/src/systems/workspace/hooks/use-workspaces.ts` - task_06 uses workspace detail data for the dialog
- `.compozy/tasks/session-driver-override/task_06.md` - depends on this provider catalog surface

### Related ADRs
- [ADR-001: Model Session Driver Selection As A Provider Override](adrs/adr-001.md) - keeps provider selection within the existing runtime model
- [ADR-004: Use Explicit Session Creation Surfaces For Provider Selection](adrs/adr-004.md) - provider choice UI depends on workspace-scoped options, not a separate endpoint

## Deliverables
- Workspace detail payload extended with sorted provider options
- Backend conversion logic that exposes workspace-visible providers directly
- Automatic internal session creators updated to pass empty provider explicitly
- Coverage proving provider-option ordering and non-interactive creator defaults **(REQUIRED)**
- Stable backend contract that the web dialog can consume without extra provider-discovery calls **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Workspace detail payload conversion emits sorted provider options
  - [ ] Empty or single-provider workspaces yield deterministic option lists
  - [ ] Automatic creator call sites pass empty provider explicitly instead of carrying stale state
  - [ ] Workspace detail handlers keep backward-compatible behavior for callers that ignore the new field
- Integration tests:
  - [ ] HTTP/UDS workspace detail responses expose provider options for a resolved workspace
  - [ ] Automatic task/runtime/consolidation creator flows still create sessions with the agent default provider
  - [ ] Provider-option assembly uses workspace-merged config rather than daemon-global provider inference
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The backend publishes workspace-scoped provider options directly to the web client
- Automatic internal session creation remains deterministic on agent defaults in this feature cut
