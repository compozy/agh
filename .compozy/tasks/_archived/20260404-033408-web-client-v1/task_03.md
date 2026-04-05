---
status: completed
title: Session System — API, CRUD & Sidebar Items
type: ""
complexity: medium
dependencies:
    - task_01
    - task_02
---

# Task 03: Session System — API, CRUD & Sidebar Items

## Overview

Build the session system's API layer, query/mutation hooks, and sidebar item component. This enables users to see sessions listed under each agent in the sidebar, create new sessions, stop running sessions, and resume stopped ones. Each session item displays its title, state badge, and visual indicators.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "API Endpoints (Consumed)" for session endpoint contracts
- REFERENCE TECHSPEC "Data Models" for SessionPayload shape
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `session-api.ts` adapter covering: `GET /api/sessions`, `POST /api/sessions`, `GET /api/sessions/:id`, `DELETE /api/sessions/:id` (stop — note: DELETE not POST), `POST /api/sessions/:id/resume`, `GET /api/sessions/:id/events`, `GET /api/sessions/:id/history`
- MUST create TanStack Query keys (`['sessions']`, `['session', id]`, `['session', id, 'events']`, `['session', id, 'history']`) and query options
- MUST create `use-sessions.ts` hook returning sessions list with `refetchInterval: 5000` (sessions change state frequently)
- MUST create `use-session-actions.ts` with mutations: `createSession` (invalidates sessions query + navigates to new session), `stopSession` (invalidates session query), `resumeSession` (invalidates session query)
- MUST create `session-sidebar-item.tsx` showing: session title (or truncated ID), state badge (active=green, stopped=gray, starting=amber animated), click to navigate to `/session/:id`
- MUST filter sessions by `agent_name` and render them nested under each agent group from task 02
- MUST sort sessions by `updated_at` descending (most recent first)
- MUST show `processing` state badge (animated spinner) on sessions that are actively streaming, in addition to active/stopped/starting states
- MUST wire "New Session" button in agent group to `createSession` mutation with the agent's name
- MUST handle error states: max sessions reached (409), session not found (404)
</requirements>

## Subtasks
- [ ] 3.1 Create `session-api.ts` adapter with all session CRUD and events/history endpoints
- [ ] 3.2 Create query keys, query options, and `use-sessions.ts` hook with auto-refresh
- [ ] 3.3 Create `use-session-actions.ts` with create/stop/resume mutations
- [ ] 3.4 Create `session-sidebar-item.tsx` with state badge and navigation link
- [ ] 3.5 Wire session items under agent sidebar groups, filtered by agent_name
- [ ] 3.6 Wire "New Session" button to createSession mutation with router navigation
- [ ] 3.7 Update session system barrel exports

## Implementation Details

See TechSpec "API Endpoints (Consumed)" for endpoint contracts. Session sidebar items use shadcn `SidebarMenuSubItem` + `SidebarMenuSubButton` nested under agent `SidebarMenuSub`.

The `createSession` mutation should POST to `/api/sessions` with `{ agent_name }`, invalidate the `['sessions']` query, and navigate to `/session/:newId` using TanStack Router's `useNavigate`.

State badges use shadcn `Badge` component with variant colors: active → green/default, stopped → secondary/gray, starting → amber with pulse animation.

### Relevant Files
- `web/src/systems/session/types.ts` — SessionPayload, UIMessage types (created in task_01)
- `web/src/systems/agent/components/agent-sidebar-group.tsx` — Agent group to nest sessions under (created in task_02)
- `web/src/components/ui/sidebar.tsx` — SidebarMenuSub*, SidebarMenuSubButton, SidebarMenuBadge
- `web/src/components/ui/badge.tsx` — For state badges
- `web/src/components/ui/spinner.tsx` — For loading states
- `internal/httpapi/sessions.go` — Backend session handlers (reference for request/response shapes)
- `.resources/harnss/src/components/sidebar/SessionItem.tsx` — Reference session item pattern

### Dependent Files
- `web/src/systems/agent/components/agent-sidebar-group.tsx` — Modified to accept and render session items
- `web/src/systems/session/index.ts` — Updated barrel
- Task 04 (streaming) depends on session API adapter created here

## Deliverables
- Complete session API adapter with all CRUD and query endpoints
- TanStack Query hooks for session list (auto-refresh) and session actions (mutations)
- Session sidebar items rendering under agent groups with state indicators
- "New Session" button creating sessions and navigating to chat
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for API adapter and mutations **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `session-api.ts` `fetchSessions()` returns parsed SessionPayload array
  - [ ] `session-api.ts` `createSession({agent_name})` sends correct POST body
  - [ ] `session-api.ts` `stopSession(id)` calls DELETE endpoint
  - [ ] `session-api.ts` `resumeSession(id)` calls POST resume endpoint
  - [ ] `session-api.ts` `fetchSessionEvents(id, query)` passes query params correctly
  - [ ] Sessions are sorted by `updated_at` descending
  - [ ] Sessions are filtered by `agent_name` for sidebar grouping
- Integration tests:
  - [ ] `session-sidebar-item.tsx` renders title and active badge for active session
  - [ ] `session-sidebar-item.tsx` renders stopped badge for stopped session
  - [ ] `session-sidebar-item.tsx` navigates to `/session/:id` on click
  - [ ] `createSession` mutation invalidates sessions query on success
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Sidebar shows sessions nested under agents with real data from daemon
- "New Session" creates a session and navigates to it
- Session state badges update when sessions start/stop
- `make web-typecheck` and `make web-lint` passing
