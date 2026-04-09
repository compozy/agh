---
status: done
title: Build Skills page
type: frontend
complexity: high
dependencies:
  - task_02
  - task_05
---

# Task 06: Build Skills page

## Overview

Build the Skills page with a three-panel layout matching the Paper design: sidebar (shared) + skill list panel + skill detail panel. The page has two tabs — Installed (grouped list with detail view) and Marketplace (search, category filters, skill rows with install buttons). Wired to the skill frontend system (task_05) for real data.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Skills Page" section under "Page Designs" for complete layout spec
- REFERENCE DESIGN.md for component styling tokens (list items, badges, filter pills, buttons)
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `routes/_app/skills.tsx` route with three-panel layout
- MUST implement tab pills: INSTALLED (default) and MARKETPLACE
- MUST implement Installed view with search input and grouped skill list (BUNDLED, WORKSPACE, MARKETPLACE sections)
- MUST implement skill list items: status dot + name + version, selected state with accent left bar
- MUST implement skill detail panel: name, version, source badge, enabled status, description, content preview card, Disable/View in CLI buttons
- MUST implement Marketplace view with full-width search, category filter chips (ALL, TESTING, etc.), marketplace rows with tags and INSTALL/INSTALLED buttons
- MUST handle loading, error, and empty states for all data-dependent views
- MUST use `useSkills` and `useSkill` hooks from skill system for data
</requirements>

## Subtasks
- [x] 6.1 Replace empty skills route with three-panel layout and INSTALLED/MARKETPLACE tab pills
- [x] 6.2 Build skill list panel component with search, grouped sections, and selection state
- [x] 6.3 Build skill detail panel component with name, badges, description, content preview, and action buttons
- [x] 6.4 Build marketplace view with search, category filter chips, and marketplace row components
- [x] 6.5 Wire all components to skill system hooks and handle loading/error/empty states
- [x] 6.6 Write tests for page rendering, tab switching, skill selection, and filter interactions

## Implementation Details

See TechSpec "Skills Page" section for complete layout specification.

Reference DESIGN.md sections: "Badges & Tags" for source/status badges, "Inputs & Filters" for search and filter chips, "List Items" for skill list rows and marketplace rows, "Page Layout" for page header bar pattern.

### Relevant Files
- `web/src/routes/_app/skills.tsx` — Route file (created as empty placeholder in task_02, now populated)
- `web/src/systems/skill/` — Skill data layer (hooks, types)
- `web/src/systems/workspace/` — Active workspace context for skill queries

### Dependent Files
- `web/src/routeTree.gen.ts` — Already updated in task_02

### Related ADRs
- [ADR-003: Full Systems Architecture](../adrs/adr-003.md) — Real data, no mocks
- [ADR-004: Foundation-First Build Order](../adrs/adr-004.md) — Skills page depends on sidebar + skill system

## Deliverables
- Fully implemented Skills page matching Paper artboards "AGH Skills Page" and "AGH Marketplace Page"
- Installed tab with grouped skill list and detail panel
- Marketplace tab with search, filters, and skill rows
- All states handled (loading, error, empty)
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Skills page renders INSTALLED tab by default with skill list
  - [x] Clicking MARKETPLACE tab switches to marketplace view
  - [x] Skill list groups skills by source (BUNDLED, WORKSPACE, MARKETPLACE)
  - [x] Selecting a skill highlights it with accent left bar
  - [x] Selecting a skill shows detail panel with correct name and description
  - [x] Detail panel shows BUNDLED badge for bundled skills
  - [x] Detail panel Disable button calls `useDisableSkill` mutation
  - [x] Marketplace search input filters displayed skills
  - [x] Category filter chips toggle active state and filter results
  - [x] Marketplace row shows INSTALLED pill for already-installed skills
  - [x] Loading state shows spinner
  - [x] Empty state shows appropriate message
- Integration tests:
  - [x] Full page flow: load skills → select skill → view detail → toggle enable/disable
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Skills page matches Paper artboards for both Installed and Marketplace views
- `make web-lint && make web-typecheck` passes
- Data loads from real backend endpoints
