---
status: completed
title: Expose automation over core API, HTTP/UDS routes, and OpenAPI
type: backend
complexity: high
dependencies:
  - task_06
---

# Task 07: Expose automation over core API, HTTP/UDS routes, and OpenAPI

## Overview

Expose the built-in automation manager through the shared API layer used by both HTTP and UDS transports. This task turns the runtime into a supported daemon surface with typed request and response contracts, webhook routes, automation-aware health data, and an updated OpenAPI document for downstream consumers.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add typed contract models and a shared `AutomationManager` surface in `internal/api/core` rather than implementing automation separately per transport.
2. MUST expose the TechSpec automation endpoints for jobs, triggers, runs, and webhooks across the appropriate HTTP and UDS route groups.
3. MUST enforce the hardened ownership rules in transport handlers, including config-backed mutation limits and webhook authentication failures before dispatch.
4. MUST update the canonical OpenAPI generator and output so generated clients and web types include the automation surfaces.
</requirements>

## Subtasks
- [x] 7.1 Add automation DTOs and manager interfaces to the shared API layer
- [x] 7.2 Implement shared handler methods for jobs, triggers, runs, webhook delivery, and automation health
- [x] 7.3 Register automation and webhook routes in HTTP and UDS transports with the correct exposure rules
- [x] 7.4 Update the OpenAPI operation registry and generated spec output
- [x] 7.5 Add transport tests for CRUD, history, webhook rejection, and health reporting

## Implementation Details

Follow the TechSpec sections "API Endpoints", "Webhook HTTP Integration", "Health Integration", and "Impact Analysis". Keep request decoding, status mapping, and ownership enforcement in `internal/api/core`; HTTP and UDS route files should stay as thin registration layers.

### Relevant Files
- `internal/api/core/interfaces.go` — Shared transport-facing interfaces belong here, including the new automation manager surface
- `internal/api/core/handlers.go` — Existing handler patterns for validation and error mapping should be extended for automation
- `internal/api/httpapi/routes.go` — HTTP route registration will add `/api/automation/*` and `/api/webhooks/*`
- `internal/api/udsapi/routes.go` — UDS route registration should expose the same non-webhook automation management surface
- `internal/api/contract/` — Additive request and response DTOs belong here
- `internal/api/spec/spec.go` — The canonical OpenAPI registry must document the new automation operations

### Dependent Files
- `internal/cli/client.go` — CLI transport methods in the next task will depend on these routes and DTOs
- `web/src/generated/agh-openapi.d.ts` — Generated web types will change once the OpenAPI spec is updated
- `web/src/systems/automation/` — The UI task will use the transport contracts created here

### Related ADRs
- [ADR-001: Built-In Daemon Component with Extension Integration Points](adrs/adr-001.md) — Automation is a built-in daemon surface that later extensions can observe and manage
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Requires one consistent API surface across schedules and triggers

## Deliverables
- Shared automation API contracts, handler methods, and route registration
- Updated OpenAPI operation registry and generated spec output
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for HTTP and UDS automation surfaces **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Handler validation rejects definition edits for config-backed jobs and triggers while allowing enabled toggles
  - [x] Handler validation rejects webhook requests with an invalid scope or malformed endpoint path before dispatch
  - [x] Contract serialization includes `scope`, `workspace_id`, `source`, `endpoint_slug`, and `webhook_id` fields in the expected wire shape
- Integration tests:
  - [x] `GET/POST/PATCH/DELETE /api/automation/jobs` round-trips the expected payloads through the shared handler layer
  - [x] `GET/POST/PATCH/DELETE /api/automation/triggers` and run-history endpoints return the expected trigger and run data
  - [x] `POST /api/webhooks/global/:endpoint` rejects invalid HMAC and accepts a valid signed request
  - [x] UDS exposes the automation management routes that CLI consumers need, while webhook delivery remains HTTP-only
  - [x] `GET /api/observe/health` includes the additive automation status block and `internal/api/spec` includes automation operations
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Automation is reachable through the shared daemon API surface with typed contracts
- OpenAPI and generated downstream clients can describe the new automation endpoints
