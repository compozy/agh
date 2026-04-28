---
status: pending
title: Web Operator Tool Diagnostics Surface
type: frontend
complexity: high
dependencies:
  - task_11
  - task_12
---

# Task 13: Web Operator Tool Diagnostics Surface

## Overview

Add a truthful web diagnostics surface for the Tool Registry using generated contracts from the daemon. This task lets operators inspect tool descriptors, backend/source state, policy/availability reasons, MCP auth diagnostics, and session-callable projections without inventing controls that the backend does not support.

<critical>
- ALWAYS READ `_techspec.md`, `web/CLAUDE.md`, `DESIGN.md`, and task_11 contracts before changing web code
- DO NOT create login, approval, invoke, or remote-management controls unless backed by real HTTP/UDS contracts
- DO NOT hand-write DTOs that should come from `web/src/generated/agh-openapi.d.ts`
- TESTS REQUIRED: adapters, query hooks, MSW fixtures, stories/routes, and generated types must compile together
</critical>

<requirements>
1. MUST build web UI from generated Tool Registry contracts rather than mirrored manual DTOs.
2. MUST add or update `web/src/systems/tools/**` for adapters, query keys/options, hooks, MSW fixtures, and components where appropriate.
3. MUST integrate MCP auth diagnostics with existing settings patterns without duplicating `agh mcp auth` behavior.
4. MUST display operator-visible unavailable, unauthorized, conflicted, and auth-required reason codes truthfully.
5. MUST show session-callable projections only where backed by task_11 endpoints.
6. MUST preserve `DESIGN.md` visual grammar and web-specific skill requirements.
</requirements>

## Subtasks
- [ ] 13.1 Add tools system adapters, query keys/options, hooks, and MSW fixtures from generated contracts
- [ ] 13.2 Add operator diagnostics components for descriptors, backend/source, availability, policy, and auth reasons
- [ ] 13.3 Integrate settings/session views only where daemon-backed state exists
- [ ] 13.4 Add route or settings placement consistent with existing web architecture
- [ ] 13.5 Add Storybook or route stories/fixtures for native, extension, MCP, conflicted, unavailable, and auth-required states
- [ ] 13.6 Add web tests, typecheck, lint, and build coverage

## Implementation Details

Use TechSpec "Impact Analysis", "Agent Manageability", and task_11 generated contracts. Follow `web/CLAUDE.md`, `DESIGN.md`, and existing `web/src/systems/settings`, `web/src/systems/skill`, `web/src/systems/network`, and `web/src/systems/session` patterns.

### Relevant Files
- `web/src/generated/agh-openapi.d.ts` - generated Tool Registry contract types
- `web/src/systems/tools/**` - new tool diagnostics system
- `web/src/systems/settings/**` - MCP auth diagnostics integration if reused
- `web/src/systems/session/**` - session-callable projection display if backed by endpoints
- `web/src/test/**` - MSW/test utilities if existing patterns require updates
- `DESIGN.md` - design-system tokens and visual grammar

### Dependent Files
- `internal/api/contract/tools.go` - generated source for web types
- `packages/site/content/runtime/core/tools.mdx` - task_14 may include screenshots or UI descriptions only after this surface exists
- `web/src/systems/tools/**/*.stories.*` - story fixtures for visual and state coverage

### Related ADRs
- [ADR-006: Tool Visibility by Surface](adrs/adr-006-tool-visibility-by-surface.md) - web is operator-visible and must show diagnostic states truthfully
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - UI must display canonical ToolID
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - MCP auth diagnostics remain redacted

### Web/Docs Impact
- `web/`: creates or updates `web/src/systems/tools/**`, generated type consumers, MSW fixtures, route/settings integration, and story/test coverage.
- `packages/site`: task_14 may document the operator diagnostics surface after it exists; no site docs are authored in this task.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: displays native, extension-host, and MCP tool source/availability states without changing extension APIs.
- Agent manageability: mirrors daemon-backed CLI/API state for operators; does not create new agent-operable backend verbs.
- Config lifecycle: displays config-derived policy/availability only if exposed by task_11; does not add config keys.

## Deliverables
- Tool diagnostics web system using generated contracts
- MSW fixtures and state coverage for native, extension, MCP, denied, conflicted, unavailable, and auth-required tools
- Route/settings/session integration where backed by daemon contracts
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration/component tests for web diagnostics **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Adapters parse generated tool DTOs without manual duplicated DTO definitions
  - [ ] Query hooks handle loading, error, empty, unavailable, conflicted, and auth-required states
  - [ ] Components display canonical `tool_id`, backend kind, source, and reason codes without secrets
  - [ ] Settings integration does not render unsupported OAuth or approval controls
- Integration tests:
  - [ ] MSW-backed route or component test renders native, extension-host, and MCP tool diagnostics
  - [ ] Session projection display matches task_11 endpoint semantics
  - [ ] `make bun-lint`, `make bun-typecheck`, `make bun-test`, and `make web-build` pass
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Web shows truthful Tool Registry diagnostics backed by daemon contracts
- No plausible-but-unsupported UI controls are introduced
