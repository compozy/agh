---
status: pending
title: Providers and Environments collection pages
type: frontend
complexity: high
dependencies:
  - task_09
---

# Task 12: Providers and Environments collection pages

## Overview

Implement the collection-driven settings pages for `providers` and `environments`, which both need list/detail/edit flows rather than single-section forms. This task should establish the reusable collection-page interaction model for settings while keeping source metadata and mutation semantics explicit.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Data Models", "Collection mutation semantics", and "Testing Approach"
- FOCUS ON "WHAT" — build collection CRUD pages, not generic dashboards
- MINIMIZE CODE — share list/detail and edit patterns across providers and environments where it improves clarity
- TESTS REQUIRED — collection loading, replace semantics, delete behavior, and validation errors must be covered
- GREENFIELD: tornar source metadata e semantics de replace/delete explícitos; não esconder isso atrás de formulários mágicos
</critical>

<requirements>
- MUST implement route pages for `providers` and `environments`
- MUST support list/detail/edit flows using the settings system collection hooks
- MUST reflect full-replacement `PUT` semantics and overlay-reveals-builtin delete behavior where applicable
- MUST surface source metadata and validation or conflict errors returned by the backend
- MUST present save and delete results in a way that makes precedence and fallback behavior understandable
- SHOULD factor shared collection UI primitives that later collection pages can reuse
</requirements>

## Design References

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| Providers | `docs/design/paper/settings/AGH Settings — Providers@2x.png` | `AGH Settings — Providers` (`YKG-0`) |
| Environments | `docs/design/paper/settings/AGH Settings — Environments@2x.png` | `AGH Settings — Environments` (`YZ2-0`) |

## Subtasks

- [ ] 12.1 Implement the `providers` collection page with list, detail, edit, and delete flows
- [ ] 12.2 Implement the `environments` collection page with list, detail, edit, and delete flows
- [ ] 12.3 Build shared collection-page UI patterns for list/detail and editor state where useful
- [ ] 12.4 Surface source metadata, conflicts, and builtin-fallback behavior in the UI
- [ ] 12.5 Add tests for collection CRUD behavior and error handling

## Implementation Details

See TechSpec sections "Data Models", "Collection mutation semantics", and "Web route coverage". These pages should build on task_09’s collection hooks and can introduce shared collection components that `mcp-servers` and `hooks-extensions` may also reuse if the abstractions stay explicit.

### Relevant Files

- `web/src/routes/_app/settings/providers.tsx` — new providers collection route
- `web/src/routes/_app/settings/environments.tsx` — new environments collection route
- `web/src/systems/settings/hooks/` — collection query and mutation hooks these pages must use
- `web/src/systems/settings/components/` — natural place for shared collection list/detail components
- `web/src/components/design-system/` — existing primitives for lists, panels, and editors

### Dependent Files

- `web/src/routes/_app/-settings*.test.tsx` — should add route coverage for collection pages
- `web/src/systems/settings/**/*.test.ts` — should add collection-hook and component coverage
- `web/src/routes/_app/settings/mcp-servers.tsx` — may reuse shared collection UI patterns in task_13
- `web/src/routeTree.gen.ts` — may update if route files change in this task

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Keeps collection resources under the shared settings shell
- [ADR-002: Persist settings by writing canonical config overlays instead of creating a new settings store](adrs/adr-002.md) — Defines replace/delete overlay semantics and builtin fallback behavior

## Deliverables

- `providers` and `environments` collection pages under the settings shell
- Shared collection UI patterns for list/detail and mutation flows where appropriate
- UI handling for replace semantics, delete semantics, and source metadata **(REQUIRED)**
- Route and component tests with >=80% coverage for the new collection pages **(REQUIRED)**
- Verified error presentation for validation and conflict responses **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] `providers` page renders list/detail state, source metadata, and submits full-replacement edits correctly
  - [ ] Provider delete flow explains builtin fallback when an overlay is removed
  - [ ] `environments` page renders usage counts, empty states, and handles save/delete states correctly
  - [ ] Validation and conflict errors surface clearly in both collection pages
  - [ ] Shared collection editor state distinguishes create versus replace semantics without dropping the selected item
- Integration tests:
  - [ ] Selecting, editing, and deleting collection items updates the correct queries and visible detail panels
  - [ ] Shared collection components can render both providers and environments without losing semantic differences
  - [ ] Duplicate-name or conflict responses are surfaced inline without corrupting list selection or editor state
  - [ ] Deleting an overlaid provider reveals builtin fallback metadata on the next refetch without a manual refresh
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for the new collection routes and shared collection components
- Providers and environments are manageable through the settings UI without hidden semantics
- Replace, delete, and fallback behavior is clear to the operator from the UI itself
