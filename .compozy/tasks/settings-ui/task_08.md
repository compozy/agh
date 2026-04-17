---
status: pending
title: Settings entrypoint and route shell
type: frontend
complexity: medium
dependencies:
  - task_06
  - task_07
---

# Task 08: Settings entrypoint and route shell

## Overview

Create the navigable entrypoint for settings in the web app by turning the existing sidebar button into a real route and introducing the shared `/_app/settings/*` shell. This task should leave the app with a stable settings frame and navigation model before the section-specific data layer and pages are added.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "System Architecture", "Impact Analysis", and "Development Sequencing"
- FOCUS ON "WHAT" — establish navigation and shell behavior, not final page content
- MINIMIZE CODE — build one shared shell and route subtree instead of duplicating layout per section
- TESTS REQUIRED — sidebar navigation and route activation need coverage
- GREENFIELD: manter a shell limpa e route-per-section; não cair numa página única com estado local gigante
</critical>

<requirements>
- MUST make the existing Settings sidebar entry navigate to the settings subtree
- MUST add the shared `/_app/settings/*` route shell and an index/default route
- MUST preserve the current application visual language and sidebar behavior while adding the new entrypoint
- MUST provide section navigation state that later pages can consume without re-implementing layout
- MUST regenerate `web/src/routeTree.gen.ts` after adding the settings routes
- SHOULD keep this task limited to shell, routing, and navigation, leaving data integration for later tasks
</requirements>

## Design References

The shared `/_app/settings/*` shell frames every settings screen, so all 10 Paper artboards are in scope for navigation, header, and layout parity. See `_techspec.md` → *Design References* for the full 10-artboard table and the task-to-screen mapping.

## Subtasks

- [ ] 8.1 Turn the sidebar Settings control into a real navigational entrypoint
- [ ] 8.2 Add the shared `/_app/settings/*` parent route and default child route
- [ ] 8.3 Build the shell layout and section navigation scaffold for later pages
- [ ] 8.4 Regenerate the route tree and update route-level imports
- [ ] 8.5 Add route and navigation tests for the new settings entrypoint

## Implementation Details

See TechSpec sections "System Architecture", "Development Sequencing", and ADR-001. Follow the existing route-per-surface patterns already used by `automation`, `network`, `skills`, and the app sidebar; do not introduce a one-off navigation model for settings.

### Relevant Files

- `web/src/components/app-sidebar.tsx` — existing Settings icon/button that must become a real link
- `web/src/components/app-sidebar.test.tsx` — sidebar activation and navigation coverage
- `web/src/routes/_app.tsx` — parent application shell that the new settings subtree lives under
- `web/src/routes/_app/index.tsx` — useful reference for default child-route behavior
- `web/src/routeTree.gen.ts` — generated route tree that must be refreshed after new routes land

### Dependent Files

- `web/src/routes/_app/settings.tsx` — new shared shell route introduced in this task
- `web/src/routes/_app/settings/index.tsx` — new default child route introduced in this task
- `web/src/systems/settings/index.ts` — consumed by later frontend tasks once the shell exists
- `web/src/hooks/routes/use-settings-page.ts` — introduced in task_09 on top of this shell

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Defines the nested settings shell and route-per-section approach

## Deliverables

- Navigable Settings entry in the app sidebar
- Shared `/_app/settings/*` route shell plus default/index route
- Regenerated `routeTree` and updated route wiring **(REQUIRED)**
- Route and sidebar tests with >=80% coverage for modified files **(REQUIRED)**
- No section-specific API logic yet; this task only establishes the shell

## Tests

- Unit tests:
  - [ ] Sidebar renders a Settings link that navigates into the settings subtree
  - [ ] Settings shell renders the expected base layout and default section state
  - [ ] Settings shell highlights the active section in navigation and preserves section metadata for child routes
  - [ ] Route tree generation includes the new settings subtree
  - [ ] Direct section-route rendering mounts inside the shared settings shell instead of duplicating layout state
- Integration tests:
  - [ ] Navigating from the sidebar reaches the settings shell without dead links
  - [ ] Refreshing the default settings route resolves the expected index child
  - [ ] Deep-linking directly to a non-index settings section renders the shell, section navigation, and matching child content
  - [ ] Browser history between settings subsections stays inside the `_app` shell without dropping shared layout state
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for modified navigation and route files
- The app has a working settings entrypoint and shared shell
- Later settings tasks can add sections without reworking global navigation
