---
status: pending
title: Multi-agent live route and live-state polish
type: frontend
complexity: high
dependencies:
  - task_15
---

# Task 17: Multi-agent live route and live-state polish

## Overview

Implement the multi-agent live experience for task trees and finish the live-state UX so parent/child execution can be followed coherently. This task should turn the task-tree live read into a usable operator surface with descendant status, active-run visibility, and linked-session drill-downs that do not collapse into N+1 fetching.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_13.md`, `task_15.md`, and `analysis_multi-agent-live.md` before building the live tree view
- REFERENCE TECHSPEC sections "Core Interfaces", "Known Risks", "System Architecture", and "Testing Approach"
- FOCUS ON "WHAT" — implement the multi-agent live surface and finish task-live UX, not unrelated list/dashboard work
- MINIMIZE CODE — reuse task-native live hooks and the existing detail-route scaffolding instead of custom per-descendant polling
- TESTS REQUIRED — tree rendering, live updates, session links, and fallback states all need coverage
- GREENFIELD: a tela multi-agent live precisa nascer em cima de task-tree live; nao aceite N+1 de detail fetch ou varios streams colados manualmente
</critical>

<requirements>
- MUST implement the multi-agent live view over the task-tree live read rather than recursive client-side detail fetching
- MUST present parent and descendant task state, active runs, latest activity, and linked-session drill-downs coherently
- MUST keep live-state updates and fallback behavior stable when no active run or descendant stream exists
- MUST integrate with the existing task-detail route family and navigation model
- SHOULD preserve layout stability as descendant state changes or more agents appear
</requirements>

## Subtasks
- [ ] 17.1 Extend detail-route orchestration with task-tree live state and view-mode control
- [ ] 17.2 Implement the multi-agent live panel/tree components for parent and descendant execution state
- [ ] 17.3 Add linked-session and active-run affordances without regressing route-state clarity
- [ ] 17.4 Add tests for tree rendering, live updates, and fallback states

## Implementation Details

See TechSpec sections "Core Interfaces", "Known Risks", and the multi-agent-live analysis. This screen should reuse the task-live hooks from the tasks system and share route state with task detail, rather than creating a parallel live route architecture.

### Relevant Files
- `web/src/hooks/routes/use-task-detail-page.ts` — detail-route orchestration that should grow multi-agent live state and view-mode behavior
- `web/src/systems/tasks/hooks/use-task-live.ts` — task-tree live read and live-state updates consumed by this screen
- `web/src/systems/tasks/components/` — new multi-agent live components introduced by this task
- `.compozy/tasks/tasks-ui/analysis/analysis_multi-agent-live.md` — identifies the current task-tree live gaps and UI expectations
- `docs/design/paper/tasks/` — local Paper export for the multi-agent live screen

### Dependent Files
- `web/src/routes/_app/tasks.$id.tsx` — task-detail route will host or navigate into the multi-agent live experience
- `web/src/routes/_app/-tasks.$id.test.tsx` — detail route coverage for live mode and tree rendering
- `web/src/systems/tasks/**/*.test.tsx` — hook/component coverage for task-tree live state
- `web/e2e/tasks.spec.ts` or `web/e2e/tasks-*.spec.ts` — browser QA in task_19 will verify the multi-agent live flow or fallback state

### Related ADRs
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Multi-agent live must depend on the dedicated task-tree live API instead of client-side stitching

## Deliverables
- Multi-agent live UI for task-tree execution state
- Stable live-state handling for parent/child tasks and active-run/session visibility
- Route and component tests with >=80% coverage for task-tree live behavior **(REQUIRED)**
- No recursive N+1 detail-fetch architecture for the multi-agent view **(REQUIRED)**
- Polished fallback states for no-live-run or disconnected-live scenarios **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Multi-agent live components render parent and descendant task state from the tree live read with stable grouping and hierarchy cues
  - [ ] Live-state components handle no-active-run, loading, disconnected, and no-descendant states gracefully
  - [ ] Descendant status chips, active-run badges, and latest-activity summaries remain stable as tree data changes
  - [ ] Linked-session affordances resolve the correct descendant run and session context for drill-down actions
  - [ ] Route-level live mode selection remains stable as tree data changes or refreshes occur
- Integration tests:
  - [ ] The detail route can switch into the multi-agent live view without refetch storms or recursive detail joins
  - [ ] Task-tree live updates refresh the visible descendant state coherently as parent and child execution changes arrive
  - [ ] Navigation from the live tree into run detail or session drill-down preserves route state correctly
  - [ ] The live view falls back cleanly when the stream disconnects or the tree read returns no active descendants
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified live-view files
- Operators can follow parent and child task execution through one coherent multi-agent live surface
- The UI uses the dedicated task-tree live model rather than fragile client-side stitching
