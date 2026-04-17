---
status: pending
title: General, Memory, and Observability pages
type: frontend
complexity: high
dependencies:
  - task_09
---

# Task 10: General, Memory, and Observability pages

## Overview

Implement the settings pages that are primarily configuration and diagnostics surfaces: `general`, `memory`, and `observability`. These pages should exercise the base settings shell, show the difference between config and runtime data, surface restart-required behavior clearly, and expose the manual actions defined in the TechSpec.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Data Models", "Runtime apply matrix", and "Testing Approach"
- FOCUS ON "WHAT" — implement the product-visible settings pages, not backend or shared system plumbing
- MINIMIZE CODE — reuse shared section patterns and the settings system from task_09
- TESTS REQUIRED — route rendering, save flow, action states, and restart banners need coverage
- GREENFIELD: separar claramente config, runtime e actions; não misturar tudo em uma tela opaca
</critical>

<requirements>
- MUST implement route pages for `general`, `memory`, and `observability` under the shared settings shell
- MUST render both config-backed and runtime-backed data where the TechSpec defines mixed section envelopes
- MUST expose restart-required UI state for config mutations in these sections
- MUST wire the Memory "Trigger now" consolidate action to the existing operational endpoint
- MUST expose observability log-tail capability metadata or entrypoint consistent with the TechSpec
- SHOULD reuse shared settings components and avoid section-specific one-off state containers where a common pattern works
</requirements>

## Design References

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| General | `docs/design/paper/settings/AGH Settings — General@2x.png` | `AGH Settings — General` (`VP8-0`) |
| Memory | `docs/design/paper/settings/AGH Settings — Memory@2x.png` | `AGH Settings — Memory` (`Z6D-0`) |
| Observability | `docs/design/paper/settings/AGH Settings — Observability@2x.png` | `AGH Settings — Observability` (`ZZL-0`) |

## Subtasks

- [ ] 10.1 Implement the `general` settings route with config-path and restart action affordances
- [ ] 10.2 Implement the `memory` settings route with config fields, health state, and consolidate action
- [ ] 10.3 Implement the `observability` settings route with config, DB metrics, and log-tail capability metadata
- [ ] 10.4 Add save-state, warning, and restart-required presentation for these sections
- [ ] 10.5 Add route and interaction tests for load, save, action, and banner behavior

## Implementation Details

See TechSpec sections "Data Models", "Runtime apply matrix", "API Endpoints", and "Web route coverage". These pages should consume the shared system from task_09 and the shared shell from task_08, while keeping page-specific UI logic focused on rendering and user interaction.

### Relevant Files

- `web/src/routes/_app/settings/general.tsx` — new route page for general settings
- `web/src/routes/_app/settings/memory.tsx` — new route page for memory settings
- `web/src/routes/_app/settings/observability.tsx` — new route page for observability settings
- `web/src/systems/settings/hooks/` — shared query and mutation hooks these pages should consume
- `web/src/systems/workspace/components/workspace-page-shell.tsx` — reference for page-shell composition and section framing

### Dependent Files

- `web/src/systems/settings/components/` — likely home for shared setting rows, banners, and action controls introduced or extended here
- `web/src/routes/_app/-settings*.test.tsx` — should add route-level coverage for these section pages
- `web/src/hooks/routes/use-settings-page.ts` — should expose the state these pages need without page-local fetch sprawl
- `web/src/routeTree.gen.ts` — may update if route files change or are added in this task

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Defines the page-per-section route model
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Defines restart-required banners and action-trigger behavior

## Deliverables

- `general`, `memory`, and `observability` settings routes implemented under the shared shell
- Restart-required and warning UX for these section mutations
- Memory consolidate action wiring and observability capability presentation **(REQUIRED)**
- Route and component tests with >=80% coverage for the new page logic **(REQUIRED)**
- Verified form and action flows against the shared settings system **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] `general` page renders config/runtime fields and surfaces restart-required state on save
  - [ ] `memory` page triggers consolidate action and displays action result state correctly
  - [ ] `observability` page renders DB metrics and log-tail capability metadata from the section envelope
  - [ ] Shared warning and restart banner components render correctly for these sections
- Integration tests:
  - [ ] Navigating among `general`, `memory`, and `observability` under the settings shell preserves section state correctly
  - [ ] Save flows invalidate and refetch the correct section queries after mutation
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for the new route and shared component logic touched by these sections
- These pages accurately separate config, runtime, and manual actions
- Users can understand when a save applied immediately versus when it requires daemon restart
