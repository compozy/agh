---
status: completed
title: API Contracts, HTTP/UDS Routes, and Codegen
type: backend
complexity: critical
dependencies:
  - task_05
  - task_07
  - task_09
  - task_10
---

# Task 11: API Contracts, HTTP/UDS Routes, and Codegen

## Overview

Expose the Tool Registry through stable public daemon contracts after all executable backends exist. This task adds contract DTOs, core handlers, HTTP and UDS routes, OpenAPI generation, generated TypeScript types, structured errors, and session-specific projection/invoke endpoints.

<critical>
- ALWAYS READ `_techspec.md`, ADR-006, and the contract/codegen rules before editing API contracts
- DO NOT expose an API that can only list descriptors; invoke routes must call executable native, extension, and MCP backends
- DO NOT update OpenAPI without regenerating downstream TypeScript and web contract artifacts
- TESTS REQUIRED: HTTP and UDS must stay behaviorally aligned for the same daemon state
</critical>

<requirements>
1. MUST add public DTOs for tool descriptors, projections, availability, policy decisions, toolsets, call inputs, call results, and structured errors.
2. MUST add core handler interfaces that depend on registry abstractions rather than concrete backend packages.
3. MUST add HTTP routes for operator registry list/search/info/toolsets and invoke operations where allowed by TechSpec.
4. MUST add UDS route parity for agent-operable local management and invocation paths.
5. MUST add session projection endpoints so hosted MCP/web/agents can compare visible callable tools.
6. MUST run codegen and co-ship OpenAPI plus generated web TypeScript contracts.
</requirements>

## Subtasks
- [x] 11.1 Add tool registry DTOs and contract tests
- [x] 11.2 Inject registry interfaces into core handlers without package-boundary violations
- [x] 11.3 Add HTTP list/search/info/invoke/toolset/session routes and status-code/body tests
- [x] 11.4 Add UDS parity routes and UDS client-compatible error payloads
- [x] 11.5 Regenerate OpenAPI and web generated TypeScript contracts
- [x] 11.6 Add contract/codegen drift tests and handler integration tests

## Implementation Details

Use TechSpec "API Endpoints", "Agent Manageability", "Data Models", and "Implementation Steps" 13-14. Codegen co-ship is mandatory because web and docs consume these contracts.

### Relevant Files
- `internal/api/contract/` - DTOs for tool registry surfaces
- `internal/api/core/handlers.go` - handler dependency injection
- `internal/api/core/tools.go` - new core tool handlers
- `internal/api/httpapi/routes.go` - HTTP route registration
- `internal/api/udsapi/routes.go` - UDS route registration
- `openapi/agh.json` - regenerated OpenAPI artifact
- `web/src/generated/agh-openapi.d.ts` - regenerated TypeScript contract artifact

### Dependent Files
- `internal/cli/client.go` - task_12 consumes UDS/HTTP client behavior
- `web/src/systems/tools/**` - task_13 consumes generated tool types
- `packages/site/content/runtime/api-reference/index.mdx` - task_14 references generated API docs
- `internal/api/*_test.go` - HTTP/UDS parity coverage

### Related ADRs
- [ADR-006: Tool Visibility by Surface](adrs/adr-006-tool-visibility-by-surface.md) - defines operator vs session response behavior
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - public DTOs use canonical ToolID only
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - API must not leak MCP auth material

### Web/Docs Impact
- `web/`: regenerate `web/src/generated/agh-openapi.d.ts`; task_13 must build tools adapters, query options, MSW fixtures, and UI from these generated types.
- `packages/site`: task_14 must update API reference and registry endpoint docs after OpenAPI changes.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: exposes native, extension-host, and MCP tool metadata through stable contracts without leaking backend implementation details.
- Agent manageability: adds HTTP and UDS paths for list, search, info, invoke, toolsets, and session projections with structured output/errors.
- Config lifecycle: no new keys; responses reflect config/policy from task_02 and task_03.

## Deliverables
- Tool Registry contract DTOs and handler interfaces
- HTTP and UDS route parity for registry operations
- Regenerated `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for HTTP/UDS parity and codegen drift **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Contract DTOs serialize canonical `tool_id`, backend kind, source ref, availability, and structured errors without secrets
  - [x] Core handlers map registry errors to deterministic HTTP/UDS status and body payloads
  - [x] Session endpoints return callable projections only while operator endpoints include unavailable/denied tools
- Integration tests:
  - [x] `GET /api/tools`, search, info, invoke, and toolsets routes return status-code plus body assertions
  - [x] Matching UDS routes return behaviorally equivalent payloads for the same state
  - [x] `make codegen` and `make codegen-check` pass with no generated drift
  - [x] `make bun-typecheck` and `make bun-test` pass against regenerated web types
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- HTTP, UDS, OpenAPI, and generated TypeScript contracts describe the same registry behavior
- Public contracts expose executable backends without leaking tokens, nonces, or approval secrets

## Verification Evidence

- `make codegen` passed and regenerated `openapi/agh.json` plus `web/src/generated/agh-openapi.d.ts`.
- `make codegen-check` passed with no generated drift.
- `make bun-typecheck` passed against regenerated web contracts.
- `make bun-test` passed: 257 files, 1838 tests.
- Focused package tests passed: `go test ./internal/tools ./internal/api/contract ./internal/api/spec ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon -count=1`.
- Full `make verify` passed after final tracking updates: codegen, web checks, `golangci-lint` with 0 issues, Go test suite with 6900 tests, build, and package boundaries.
