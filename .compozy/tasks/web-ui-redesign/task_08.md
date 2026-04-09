---
status: pending
title: Build Knowledge page
type: frontend
complexity: high
dependencies:
  - task_02
  - task_07
---

# Task 08: Build Knowledge page

## Overview

Build the Knowledge page with a three-panel layout matching the Paper design: sidebar (shared) + knowledge list panel + knowledge detail panel. The page has scope tabs (ALL/GLOBAL/WORKSPACE), a search input, a grouped list by scope with type badges, and a detail panel showing name, description, content preview, metadata table, and action buttons. Wired to the knowledge frontend system (task_07) for real data via existing memory endpoints.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Knowledge Page" section under "Page Designs" for complete layout spec
- REFERENCE DESIGN.md for component styling tokens (list items, badges, metadata table, content preview cards)
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `routes/_app/knowledge.tsx` route with three-panel layout
- MUST implement tab pills: ALL (default), GLOBAL, WORKSPACE
- MUST implement page header with icon + "Knowledge" title + count badge + "Dream: Xh ago" status indicator
- MUST implement search input in list panel
- MUST implement grouped knowledge list by scope (GLOBAL, WORKSPACE) with counts
- MUST implement list items: title, description, date right-aligned, type+scope badges (colored per DESIGN.md badge tint formula)
- MUST implement selected item state: bg `#2C2C2E` with 3px left accent bar
- MUST implement detail panel: title, version, status dot, file path, DESCRIPTION section, CONTENT preview card with "View full content →" link, Delete/View in CLI action buttons, METADATA striped table
- MUST handle loading, error, and empty states for all data-dependent views
- MUST use `useMemories` and `useMemory` hooks from knowledge system
</requirements>

## Subtasks
- [ ] 8.1 Replace empty knowledge route with three-panel layout and scope tab pills
- [ ] 8.2 Build knowledge list panel with search, grouped sections, type/scope badges, and selection state
- [ ] 8.3 Build knowledge detail panel with title, status, description, content preview card, and action buttons
- [ ] 8.4 Build metadata table component (striped key-value rows)
- [ ] 8.5 Wire all components to knowledge system hooks and handle loading/error/empty states
- [ ] 8.6 Write tests for page rendering, tab filtering, item selection, and detail display

## Implementation Details

See TechSpec "Knowledge Page" section for complete layout specification.

Reference DESIGN.md sections: "Badges & Tags" for type/scope badges (USER=accent, FEEDBACK=accent, PROJECT=success, REFERENCE=info, GLOBAL/WS=neutral), "Data Display" for content preview card and metadata table, "List Items" for knowledge list item pattern, "Page Layout" for header bar with dream status.

### Relevant Files
- `web/src/routes/_app/knowledge.tsx` — Route file (created as empty placeholder in task_02, now populated)
- `web/src/systems/knowledge/` — Knowledge data layer (hooks, types)
- `web/src/systems/workspace/` — Active workspace context for scoped queries

### Dependent Files
- `web/src/routeTree.gen.ts` — Already updated in task_02

### Related ADRs
- [ADR-003: Full Systems Architecture](../adrs/adr-003.md) — Real data via existing memory endpoints
- [ADR-004: Foundation-First Build Order](../adrs/adr-004.md) — Knowledge page depends on sidebar + knowledge system

## Deliverables
- Fully implemented Knowledge page matching Paper artboard "AGH Memory / Knowledge Page"
- Three-panel layout with scope tabs, grouped list, and detail panel
- All states handled (loading, error, empty)
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Knowledge page renders ALL tab by default with full memory list
  - [ ] GLOBAL tab filters to show only global-scope memories
  - [ ] WORKSPACE tab filters to show only workspace-scope memories
  - [ ] List groups memories by scope (GLOBAL section, WORKSPACE section) with counts
  - [ ] Selecting a memory highlights it with accent left bar
  - [ ] Selecting a memory shows detail panel with correct title and description
  - [ ] Detail panel shows USER badge with accent tint for user-type memories
  - [ ] Detail panel shows content preview card with truncated content
  - [ ] Detail panel "View full content →" link is clickable
  - [ ] Detail panel Delete button calls `useDeleteMemory` mutation
  - [ ] Metadata table renders striped rows for type, scope, agent, modified
  - [ ] Dream status indicator shows "Dream: Xh ago" in page header
  - [ ] Search input filters the memory list
  - [ ] Loading state shows spinner
  - [ ] Empty state shows appropriate message
- Integration tests:
  - [ ] Full page flow: load memories → select memory → view detail → delete memory
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Knowledge page matches Paper artboard "AGH Memory / Knowledge Page"
- `make web-lint && make web-typecheck` passes
- Data loads from real backend memory endpoints
