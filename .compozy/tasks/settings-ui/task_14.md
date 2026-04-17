---
status: pending
title: Hooks and Extensions page
type: frontend
complexity: high
dependencies:
  - task_09
---

# Task 14: Hooks and Extensions page

## Overview

Implement the combined `hooks-extensions` settings page that depends on both config-backed hook declarations and operational extension data now available over HTTP. This page closes the last major surface in the Paper settings set by blending config, runtime extension summaries, and restart-aware policy editing without collapsing into an operational extension manager.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Data Models", "API Endpoints", and "Transport and security policy"
- FOCUS ON "WHAT" — implement the combined hooks/extensions settings surface, not a second full extension console
- MINIMIZE CODE — reuse the settings system and extension parity delivered by backend tasks instead of inventing separate clients
- TESTS REQUIRED — hook config, extension runtime state, mutation gating, and restart messaging all need coverage
- GREENFIELD: manter claro o que é config-backed e o que é operational action dentro da mesma tela
</critical>

<requirements>
- MUST implement the `hooks-extensions` route under the settings shell
- MUST render hook declarations, extension marketplace/resource policy config, and installed extension summaries from the combined settings surface
- MUST support extension operational actions that apply immediately while keeping policy/config edits restart-aware
- MUST consume the HTTP-visible extension surface introduced by backend tasks instead of special-casing UDS-only behavior
- MUST communicate transport or mutation availability state where the backend exposes it
- SHOULD keep hook config editing and extension operational actions visually distinct within the page
</requirements>

## Design References

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| Hooks & Extensions | `docs/design/paper/settings/AGH Settings — Hooks & Extensions@2x.png` | `AGH Settings — Hooks & Extensions` (`106W-0`) |

## Subtasks

- [ ] 14.1 Implement the `hooks-extensions` route with combined config and runtime sections
- [ ] 14.2 Render hook declarations and extension policy state from the settings section envelope
- [ ] 14.3 Wire extension operational actions that apply immediately through the shared settings or extension adapters
- [ ] 14.4 Surface restart-required versus immediate-action behavior clearly in the UI
- [ ] 14.5 Add tests for combined page rendering, actions, and mutation messaging

## Implementation Details

See TechSpec sections "Data Models", "API Endpoints", "Transport and security policy", and ADR-001/ADR-004. This page is intentionally hybrid: it mixes config-backed settings and operational extension actions, but it should still remain a settings screen and avoid recreating a separate full extension management surface.

### Relevant Files

- `web/src/routes/_app/settings/hooks-extensions.tsx` — new combined settings route
- `web/src/systems/settings/hooks/` — settings query and mutation hooks for the section envelope
- `web/src/systems/settings/components/` — likely place for hook and extension summary panels
- `web/src/systems/` — existing extension-related client code or utilities that can be reused instead of duplicating action logic
- `web/src/components/design-system/` — shared primitives for grouped config/runtime/action sections

### Dependent Files

- `web/src/routes/_app/-settings*.test.tsx` — should add route coverage for the hooks/extensions page
- `web/src/systems/settings/**/*.test.ts` — should add combined-page hook and action coverage
- `web/src/systems/settings/adapters/settings-api.ts` — may need extension-action bridging or wrappers already scaffolded in task_09
- `web/src/routeTree.gen.ts` — may update if route files change in this task

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Allows the combined Hooks & Extensions route in the settings shell
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Distinguishes restart-required policy edits from immediate operational actions
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — Constrains HTTP mutation availability for settings and extension operations

## Deliverables

- `hooks-extensions` settings route with combined hook config and extension runtime/action sections
- UI for immediate extension actions alongside restart-aware policy editing
- Integration with the HTTP-visible extension surface required by the TechSpec **(REQUIRED)**
- Route and component tests with >=80% coverage for combined page behavior **(REQUIRED)**
- Clear operator messaging for action-trigger versus restart-required flows **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] Combined page renders hook declarations, extension summaries, and policy state from the section envelope
  - [ ] Extension operational actions show immediate progress/result state without masquerading as config saves
  - [ ] Policy edits surface restart-required messaging distinctly from operational actions
  - [ ] Mutation or transport availability state is rendered correctly when the backend reports restrictions
  - [ ] Shared controls disable or explain unavailable extension actions based on transport parity or policy metadata
- Integration tests:
  - [ ] Combined page can load, mutate policy, and trigger extension actions without route-level fetch duplication
  - [ ] Query invalidation and refetch behavior updates both config-backed and runtime-backed portions of the page correctly
  - [ ] Immediate extension actions update the runtime summary without incorrectly showing a restart-required banner
  - [ ] Policy saves preserve the extension runtime list and hook declarations after refetch
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for the hooks/extensions route and supporting settings components
- The page combines hook config and extension runtime state without blurring their different mutation semantics
- The web app can deliver the full Hooks & Extensions settings screen over HTTP as required by the TechSpec
