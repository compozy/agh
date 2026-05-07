---
status: completed
title: "Daemon Catalog Wiring"
type: backend
complexity: high
dependencies:
  - task_03
  - task_04
---

# Task 5: Daemon Catalog Wiring

## Overview
This task wires the catalog service, global DB store, and model sources into AGH's daemon composition root. It makes the service available to runtime dependencies without adding public routes yet.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST compose `modelcatalog.Service` only in `internal/daemon`, preserving daemon as the composition root.
- MUST inject global DB store, builtin/config/models.dev/live sources, logger, clock, HTTP client, and timeout configuration.
- MUST expose the service through handler/runtime dependency structs for later API tasks.
- MUST not introduce an event bus, NATS-based in-process coordination, or package import cycles.
- MUST ensure daemon shutdown does not leave background refresh work untracked.
- MUST keep HTTP/UDS public route registration out of this task.
- MUST update `magefile.go` boundary rules for `internal/modelcatalog` in the same task if a new package/import edge is introduced.
</requirements>

## Subtasks
- [x] 5.1 Add daemon/runtime dependency fields for the model catalog service.
- [x] 5.2 Compose store and sources in daemon boot using existing config and global DB handles.
- [x] 5.3 Inject the service into API core handler config without registering routes.
- [x] 5.4 Add shutdown/deadline behavior for background refresh work.
- [x] 5.5 Update `magefile.go` boundary rules for `internal/modelcatalog` when needed and add daemon wiring tests plus boundary import checks.

## Implementation Details
Follow `_techspec.md` sections `Architectural Boundaries` and `Implementation Plan`. Activate `agh-code-guidelines`, `golang-pro`, `agh-cleanup-failure-paths`, `agh-test-conventions`, and `testing-anti-patterns`.

### Relevant Files
- `internal/daemon/daemon.go` - daemon runtime struct and dependency ownership.
- `internal/daemon/boot.go` - composition and service wiring.
- `internal/api/core/handlers.go` - BaseHandlerConfig dependency injection.
- `internal/api/httpapi/server.go` - HTTP server handler config construction.
- `internal/api/udsapi/server.go` - UDS server handler config construction.
- `magefile.go` - boundary rules MUST be updated if the new package/import edge changes allowed boundaries.

### Dependent Files
- `internal/api/core` - Task 07 adds handlers using the injected service.
- `internal/api/httpapi` - Task 07 registers HTTP routes.
- `internal/api/udsapi` - Task 07 registers UDS routes.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - requires daemon-owned service composition.

### Web/Docs Impact
- `web/`: none directly in this task - no public route or generated contract yet.
- `packages/site`: none directly in this task - docs update after surfaces in Task 10.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: prepares source adapter injection used by extension source contract in Task 08.
- Agent manageability: prepares service for HTTP/UDS/CLI access in Task 07.
- Config lifecycle: wires sources from current config; does not add new TOML keys beyond Task 01.

## Deliverables
- Daemon-composed `modelcatalog.Service` and dependencies.
- Handler/runtime dependency injection ready for API route implementation.
- Cleanup/shutdown behavior for background refresh work.
- Required `magefile.go` boundary updates for `internal/modelcatalog`.
- Unit/integration tests with 80%+ coverage for wiring behavior **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] daemon boot creates catalog service when global DB and config are available.
  - [x] missing optional live source dependencies degrade to source status instead of boot failure.
  - [x] handler config receives the catalog service dependency.
  - [x] shutdown cancels/joins catalog refresh work without leaks.
- Integration tests:
  - [x] daemon test runtime can list catalog rows through the service before routes exist.
  - [x] boundary checks pass with the new package/import graph.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `go test ./internal/daemon ./internal/api/core` passes.
- No import cycle or boundary violation is introduced.
