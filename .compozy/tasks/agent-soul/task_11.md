---
status: completed
title: Expose HTTP and UDS Routes Through Shared Core Handlers
type: backend
complexity: high
dependencies:
  - task_10
---

# Task 11: Expose HTTP and UDS Routes Through Shared Core Handlers

## Overview

Expose Soul, Heartbeat, session health, and wake status through HTTP and UDS by adapting shared core handlers to the services and contracts created earlier. This task makes the feature agent-operable without duplicating behavior across transports.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, every ADR, and current API routing patterns before editing.
- REFERENCE TECHSPEC for route names, request bodies, response shapes, route parity, errors, and redaction.
- FOCUS ON WHAT must be exposed: shared core handlers, HTTP routes, UDS routes, parity tests, and deterministic errors.
- MINIMIZE CODE in task notes; do not add CLI formatting here.
- TESTS REQUIRED for HTTP/UDS parity, CAS conflicts, validation errors, redaction, and session health reads.
- NO WORKAROUNDS: no HTTP-only features, no UDS-only features, and no transport-specific business logic.
</critical>

<requirements>
- MUST activate `agh-code-guidelines` and `golang-pro` before editing production Go.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST implement shared `internal/api/core` handlers for Soul read/validate/write/delete/history/rollback/refresh as specified.
- MUST implement shared `internal/api/core` handlers for Heartbeat read/validate/write/delete/history/rollback/status and session health/status/inspect.
- MUST register HTTP and UDS routes using the same core handler logic and contract DTOs.
- MUST return deterministic status codes/errors for not found, invalid content, stale `expected_digest`, disabled config, and ineligible sessions.
- MUST preserve route redaction and avoid exposing raw prompt-only content where the spec forbids it.
</requirements>

## Subtasks
- [x] 11.1 Add shared core handlers that adapt Soul services to contract DTOs.
- [x] 11.2 Add shared core handlers that adapt Heartbeat, wake status, and session health services to contract DTOs.
- [x] 11.3 Register HTTP routes and route metadata.
- [x] 11.4 Register UDS routes with parity to HTTP.
- [x] 11.5 Add handler, route, and transport parity tests.
- [x] 11.6 Verify generated OpenAPI still matches implemented HTTP routes.

## Implementation Details

Business behavior should stay in the Soul, Heartbeat, and session services. Core handlers should handle auth/context, request validation, conversion, and error mapping; transports should only bind routes.

### Relevant Files
- `internal/api/core/` - shared handler implementation.
- `internal/api/httpapi/routes.go` - HTTP route registration.
- `internal/api/udsapi/routes.go` - UDS route registration.
- `internal/api/core/conversions.go` - DTO conversion helpers.
- `internal/api/core/agent_channels.go` - existing `/agent/context` and agent-surface precedent.
- `internal/soul/` - service dependencies.
- `internal/heartbeat/` - service dependencies.
- `internal/session/` - session health dependencies.

### Dependent Files
- `internal/api/core/*_test.go` - handler behavior and error mapping.
- `internal/api/httpapi/*_test.go` - HTTP route registration and response tests.
- `internal/api/udsapi/*_test.go` - UDS route registration and parity tests.
- `openapi/agh.json` - must remain consistent with HTTP routes.
- `.compozy/tasks/agent-soul/task_12.md` - CLI commands consume these routes.
- `.compozy/tasks/agent-soul/task_13.md` - Host API/tools/resources can reuse core behavior or services.

### Related ADRs
- [ADR-002: Soul Prompt and Read Model Exposure](adrs/adr-002.md) - requires context/read model exposure.
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - requires managed authoring endpoints.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - requires CLI/HTTP/UDS parity.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: routes establish stable public behavior that Host API, tools, resources, and SDKs must mirror later.
- Agent manageability: complete HTTP/UDS surface for agents to inspect and manage Soul, Heartbeat, and session health.
- Config lifecycle: expose disabled/config-bound errors deterministically; no new config keys.

### Web/Docs Impact
- Web impact: generated web client types already exist from task_10; task_14 updates consumers after routes stabilize.
- Docs impact: task_15 must document HTTP and UDS route behavior, request bodies, response payloads, and redaction.

## Deliverables
- Shared core handlers for Soul, Heartbeat, session health, and wake status.
- HTTP routes with contract-aligned payloads.
- UDS routes with behavior parity to HTTP.
- Handler and transport tests for success and failure paths.
- Completion evidence that there is no duplicated transport-specific business logic.

## Tests
- Unit tests:
  - [x] Core handlers map valid service responses to contract DTOs.
  - [x] Core handlers map validation, conflict, not-found, disabled, and ineligible errors deterministically.
  - [x] Redaction tests prove forbidden raw content is absent.
  - [x] Session health handlers return closed state and reason values.
- Integration tests:
  - [x] HTTP and UDS routes return equivalent payloads for the same Soul read/status request.
  - [x] HTTP and UDS routes return equivalent errors for stale `expected_digest`.
  - [x] `GET /agent/context` or equivalent route includes compact Soul projection when expected.
  - [x] OpenAPI route definitions match implemented HTTP routes.
- Test coverage target: >=80%.
- All tests must pass.

## Completion Evidence
- Shared authored-context route behavior lives in `internal/api/core` and is bound by thin HTTP/UDS route registrations.
- HTTP/UDS parity is covered from `internal/daemon`, the composition root allowed to exercise both transports.
- Verification passed with `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon ./internal/session -count=1`.
- Verification passed with `make verify`, including codegen/OpenAPI drift checks and package-boundary checks.

## References
- `_techspec.md` - route parity and shared core boundary.
- `_techspec_soul.md` - Soul endpoint behavior.
- `_techspec_heartbeat.md` - Heartbeat/session health endpoint behavior.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - gateway route/protocol precedent.
- `.resources/openclaw/docs/gateway/protocol.md:313-438` - protocol route precedent.
- `.resources/paperclip/server/src/routes/agents.ts:2694-3045` - agent route/status precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Agents can use HTTP and UDS to inspect and manage authored context consistently.
- Route behavior is shared through core handlers and matches generated contracts.
