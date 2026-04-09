---
status: completed
title: Build custom sidebar with workspace icon rail
type: frontend
complexity: high
dependencies:
    - task_01
---

# Task 02: Build custom sidebar with workspace icon rail

## Overview

Replace the shadcn Sidebar component with a custom two-zone sidebar matching the Paper design: a 40px workspace icon rail (always visible) plus a ~220px collapsible panel containing workspace name, search, agent list, workspace nav (Knowledge/Skills), and system footer. This establishes the app shell layout that all pages share.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Custom Sidebar Component" section for structure, props, and styling specs
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement a two-zone layout: 40px icon rail + ~220px collapsible panel
- MUST render workspace circle avatars (32px) in the icon rail with single-letter labels
- MUST show app logo with `#E8572A` background and white letter in the icon rail
- MUST highlight active workspace with `#E8572A` ring border
- MUST display agent list with expandable sessions per agent (reuse existing data hooks)
- MUST add Knowledge and Skills nav items below agent list, navigating to `/_app/knowledge` and `/_app/skills`
- MUST show active nav indicator: 3px left accent bar `#E8572A`
- MUST show system footer with connection status, version, and settings
- MUST support collapsed mode (icon rail only, 40px) with sidebar state in Zustand store
- MUST update `_app.tsx` to use custom sidebar layout instead of SidebarProvider/SidebarInset
- MUST create empty route files for `/_app/skills` and `/_app/knowledge` (placeholder content)
</requirements>

## Subtasks
- [ ] 2.1 Create `useSidebarStore` Zustand store for sidebar collapse state
- [ ] 2.2 Build icon rail component with workspace avatars and app logo
- [ ] 2.3 Build sidebar panel with header, agent list, workspace nav, and system footer
- [ ] 2.4 Update `_app.tsx` layout to use custom sidebar instead of shadcn SidebarProvider
- [ ] 2.5 Create empty route files for `/_app/skills` and `/_app/knowledge`
- [ ] 2.6 Write tests for sidebar rendering, collapse toggle, and navigation

## Implementation Details

See TechSpec "Custom Sidebar Component" section for the complete layout diagram, agent list pattern, and navigation items.

The sidebar replaces the current `AppSidebar` component which uses shadcn `Sidebar`, `SidebarContent`, `SidebarHeader`, etc. The existing data hooks (`useAgents`, `useSessions`, `useWorkspaces`, `useDaemonHealth`) remain unchanged — only the presentation layer changes.

### Relevant Files
- `web/src/components/app-sidebar.tsx` — Current sidebar, to be rewritten
- `web/src/routes/_app.tsx` — App layout wrapper, must remove SidebarProvider
- `web/src/systems/agent/components/agent-sidebar-group.tsx` — Current agent group component, restyle
- `web/src/systems/session/components/session-sidebar-item.tsx` — Current session item, restyle
- `web/src/systems/workspace/components/workspace-selector.tsx` — Current workspace selector, adapt for icon rail
- `web/src/systems/daemon/components/connection-status.tsx` — Footer status display

### Dependent Files
- `web/src/routes/_app/skills.tsx` — New empty route file (created by this task)
- `web/src/routes/_app/knowledge.tsx` — New empty route file (created by this task)
- `web/src/routeTree.gen.ts` — Auto-regenerated when new routes are added
- `web/src/components/app-header.tsx` — May need updates for breadcrumb/session header pattern

### Related ADRs
- [ADR-002: Custom Sidebar with Workspace Icon Rail](../adrs/adr-002.md) — Mandates custom implementation over shadcn extension

## Deliverables
- Rewritten `app-sidebar.tsx` with icon rail + panel layout
- New `useSidebarStore` Zustand store
- Updated `_app.tsx` layout
- Empty `skills.tsx` and `knowledge.tsx` route files
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Icon rail renders workspace circle avatars from workspace data
  - [ ] App logo renders with accent background color
  - [ ] Active workspace shows accent ring border
  - [ ] Agent list renders agents with session counts
  - [ ] Knowledge nav item links to `/_app/knowledge` route
  - [ ] Skills nav item links to `/_app/skills` route
  - [ ] Collapse toggle hides panel, icon rail remains visible
  - [ ] System footer shows connection status component
  - [ ] Active nav item shows 3px left accent bar
- Integration tests:
  - [ ] Navigating to `/_app/skills` renders empty placeholder
  - [ ] Navigating to `/_app/knowledge` renders empty placeholder
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Sidebar matches Paper artboard "AGH Sidebar — Collapsed" visually
- Navigation to all routes works (index, session, skills, knowledge)
- `make web-lint && make web-typecheck` passes
