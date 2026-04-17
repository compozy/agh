---
status: completed
title: web/src/systems/settings domain scaffold
type: frontend
complexity: high
dependencies:
  - task_08
---

# Task 09: web/src/systems/settings domain scaffold

## Overview

Create the reusable frontend domain layer for settings under `web/src/systems/settings`, mirroring the existing systems pattern used elsewhere in the app. This task should own generated-type adaptation, query keys/options, mutation hooks, restart polling, and the shared route/page hook that later section pages build on.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "System Architecture", "API Endpoints", and "Testing Approach"
- FOCUS ON "WHAT" — establish one frontend settings system, not ad hoc fetches inside route files
- MINIMIZE CODE — follow the existing systems pattern instead of inventing a parallel frontend architecture
- TESTS REQUIRED — adapters, query behavior, restart polling, and shared page state need coverage
- GREENFIELD: não espalhar lógica de settings pelas routes; centralizar em `web/src/systems/settings`
</critical>

<requirements>
- MUST create `web/src/systems/settings` as the canonical frontend domain for settings
- MUST add typed API adapters, query keys/options, and mutation hooks for sections, collections, restart action, restart status polling, and log-tail metadata
- MUST add a shared route/page hook for settings shell state and section-level orchestration
- MUST consume generated API types from the updated OpenAPI contract instead of introducing loose client-side DTOs
- MUST support source-precedence metadata and restart-required state needed by later pages
- SHOULD expose a stable public barrel so section routes can import from one domain entrypoint
</requirements>

## Design References

The `web/src/systems/settings` domain feeds every settings page, collection page, restart banner, and log-tail surface. All 10 Paper artboards are downstream consumers of this scaffold. See `_techspec.md` → *Design References* for the full 10-artboard table and the task-to-screen mapping.

## Subtasks

- [x] 9.1 Create the `web/src/systems/settings` directory structure and public exports
- [x] 9.2 Add settings API adapters and typed request/response shaping on top of generated types
- [x] 9.3 Add query keys, query options, and mutation hooks for sections, collections, and restart actions
- [x] 9.4 Add shared settings route/page hook state, including section navigation and restart polling
- [x] 9.5 Add tests for adapters, hooks, and restart/status state management

## Implementation Details

See TechSpec sections "System Architecture", "API Endpoints", "Response behavior", and "Testing Approach". Follow the existing `web/src/systems/*` conventions already used by `automation`, `network`, and `workspace`, and keep route files thin by moving API/state orchestration into this domain.

### Relevant Files

- `web/src/generated/agh-openapi.d.ts` — generated API types that this system should adapt and consume
- `web/src/systems/automation/` — reference for adapter, hooks, and query-organization patterns
- `web/src/systems/network/` — reference for runtime-oriented read models and query patterns
- `web/src/hooks/routes/use-automation-page.ts` — reference for route-level orchestration that should become a shared settings hook
- `web/src/lib/` — shared query and client utilities the settings system may reuse

### Dependent Files

- `web/src/routes/_app/settings/*.tsx` — later section pages should consume this system instead of calling the API directly
- `web/src/components/app-sidebar.tsx` — already points to the shell from task_08 and should not take on data logic
- `web/src/systems/settings/**/*.test.ts` — should provide package-level coverage for the new domain
- `web/src/hooks/routes/use-settings-page.ts` — introduced or completed in this task

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Defines the frontend settings shell and section model
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Requires restart polling and action-aware UI state

## Deliverables

- New `web/src/systems/settings` package with adapters, hooks, query primitives, and exports
- Shared settings page hook with restart polling and section-state handling
- Typed integration with generated OpenAPI output **(REQUIRED)**
- Unit tests with >=80% coverage for adapters and hooks **(REQUIRED)**
- Route-hook tests that verify settings shell state and restart polling behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Settings adapters decode section and collection payloads from generated API types correctly
  - [x] Collection adapters preserve precedence metadata, restart metadata, and workspace context without manual shape fixing in routes
  - [x] Query keys/options remain stable for section and collection resources
  - [x] Restart action mutations trigger status polling and expose progress state to consumers
  - [x] Shared settings page hook derives active section and restart banner state correctly
  - [x] Mutation helper utilities invalidate only the affected section or collection query families
- Integration tests:
  - [x] Route-level hook integration resolves section changes without duplicating fetch logic in route files
  - [x] Generated types and settings system exports typecheck together without `any` fallbacks
  - [x] Restart polling state can be consumed consistently across multiple settings pages without duplicated route-local state
  - [x] Scoped collection hooks keep global and workspace caches isolated while sharing the same adapter layer
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for the new settings frontend system
- Route files can render settings pages using a single domain layer
- Restart polling and source-precedence metadata are available to all later settings pages
