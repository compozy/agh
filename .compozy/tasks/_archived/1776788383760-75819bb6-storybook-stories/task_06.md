---
status: completed
title: Stories for small systems (agent, daemon, knowledge, workspace, skill)
type: frontend
complexity: medium
dependencies:
  - task_05
---

# Task 6: Stories for small systems (agent, daemon, knowledge, workspace, skill)

## Overview
Cover the five systems with the smallest component counts in one batch: agent (2), daemon (1), knowledge (2), workspace (3), skill (3) — totaling 11 stories. Each story consumes the global MSW handlers composed in task_05, exercises the data-bound render path, and demonstrates overridden handlers for loading and error states where meaningful.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add stories under `web/src/systems/<name>/components/stories/<component>.stories.tsx` for: `agent-icon`, `agent-sidebar-group`, `connection-status`, `knowledge-detail-panel`, `knowledge-list-panel`, `workspace-page-shell`, `workspace-selector`, `workspace-setup`, `marketplace-view`, `skill-detail-panel`, `skill-list-panel`.
- MUST title stories `systems/<name>/<ComponentName>`.
- MUST rely on the default MSW handlers composed in task_05 for success-state rendering; only override handlers when the story is loading/error/empty.
- MUST include at least a `Default` story for every component and, for panels bound to queries (`connection-status`, `knowledge-*`, `workspace-selector`, `skill-*`, `marketplace-view`), add an additional `Loading` or `Error` story.
- MUST NOT bypass the system's public barrel when importing the component (import from the component file directly since stories live alongside).
- MUST NOT introduce new fixtures inside story files — extend the system `fixtures.ts` from task_05 if needed.
</requirements>

## Subtasks
- [x] 6.1 Agent stories: `agent-icon` (variants), `agent-sidebar-group` (default + empty group).
- [x] 6.2 Daemon story: `connection-status` (Default + Disconnected + Reconnecting).
- [x] 6.3 Knowledge stories: `knowledge-list-panel` (Default + Loading), `knowledge-detail-panel` (Default + Empty).
- [x] 6.4 Workspace stories: `workspace-page-shell` (Default), `workspace-selector` (Default + Empty), `workspace-setup` (Default + Validation error).
- [x] 6.5 Skill stories: `skill-list-panel` (Default + Loading), `skill-detail-panel` (Default), `marketplace-view` (Default + Error).

## Implementation Details
See TechSpec "Core Interfaces" for the story and handler-override contracts. For loading stories, override with `http.get(..., async () => { await delay("infinite"); })`. For error stories, return `HttpResponse.json({ error: "..." }, { status: 500 })`. Keep every `render` under 20 lines; hoist repeated data into module-level constants co-located with the story file when it would bloat the render.

### Relevant Files
- `web/src/systems/agent/components/*` — agent subjects.
- `web/src/systems/daemon/components/connection-status.tsx` — daemon subject.
- `web/src/systems/knowledge/components/*` — knowledge subjects.
- `web/src/systems/workspace/components/*` — workspace subjects.
- `web/src/systems/skill/components/*` — skill subjects.
- `web/src/systems/<name>/mocks/index.ts` — source of typed fixtures for overrides.

### Dependent Files
- `task_11` — verification step depends on these stories existing.
- `web/.storybook/preview.ts` — per-story `parameters.msw.handlers` overrides layer on top of the global set.

### Related ADRs
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md) — Global data layer used here.
- [ADR-003: stories/ Subfolder Placement, Opt-in Autodocs](adrs/adr-003.md) — Placement + no-autodocs policy.

## Deliverables
- 11 new story files across five systems.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for small-system rendering under MSW **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `connection-status` `Disconnected` story overrides the health handler to return 503 and renders the offline indicator copy.
  - [ ] `workspace-setup` `Validation error` story renders a field error without navigating.
  - [ ] `knowledge-list-panel` `Loading` story suspends and renders `Skeleton` rows.
  - [ ] `skill-list-panel` `Default` story renders the fixture list count from `systems/skill/mocks/fixtures`.
  - [ ] `marketplace-view` `Error` story renders the fallback empty-state affordance.
- Integration tests:
  - [ ] All 11 stories index in `build-storybook` and render without console errors.
  - [ ] The dark-theme variant of every story passes a11y critical checks.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- 11 story files present under `web/src/systems/{agent,daemon,knowledge,workspace,skill}/components/stories/`.
- No handler definitions or inline fixtures duplicated from the system's `mocks/` folder.
