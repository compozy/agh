---
status: completed
title: Daemon & Agent Systems — API + Sidebar
type: ""
complexity: medium
dependencies:
    - task_01
---

# Task 02: Daemon & Agent Systems — API + Sidebar

## Overview

Build the daemon and agent domain systems end-to-end: API adapters, TanStack Query hooks, and UI components. The daemon system provides health polling and a connection status indicator. The agent system provides the sidebar's primary navigation — a list of configured agents, each rendered as a collapsible group with an icon and name, ready to hold session items from task 03.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "File Structure" for system directory layout
- REFERENCE TECHSPEC "API Endpoints (Consumed)" for endpoint contracts
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create daemon system: `adapters/daemon-api.ts`, `lib/query-keys.ts`, `lib/query-options.ts`, `hooks/use-daemon-health.ts`, `components/connection-status.tsx`
- MUST poll `GET /api/observe/health` at 10s interval; derive connection state (connected/disconnected/reconnecting) from query status
- MUST show connection indicator in app header (green dot = connected, red = disconnected, amber = reconnecting)
- MUST show a full-screen daemon-offline state when health polling fails on initial load: message "Daemon not found" with instruction to run `agh daemon start`, auto-retry polling in background
- MUST create agent system: `adapters/agent-api.ts`, `lib/query-keys.ts`, `lib/query-options.ts`, `hooks/use-agents.ts`, `components/agent-sidebar-group.tsx`, `components/agent-icon.tsx`
- MUST fetch agents via `GET /api/agents` with staleTime of 60s (agents rarely change)
- MUST render each agent as a collapsible `SidebarGroup` with provider icon, agent name, and a "New Session" action button
- MUST map agent provider names to icons (claude, codex, gemini, etc.) in `agent-icon.tsx`
- MUST follow app-renderer-systems dependency flow: adapters → lib → hooks → components
- MUST update barrel `index.ts` files for both systems
- MUST wire connection-status into the `_app.tsx` header area
- MUST wire agent sidebar groups into the `_app.tsx` sidebar content
</requirements>

## Subtasks
- [x] 2.1 Create daemon system: API adapter, query infrastructure, and health polling hook
- [x] 2.2 Create `connection-status.tsx` component and wire into app header
- [x] 2.3 Create agent system: API adapter, query infrastructure, and `useAgents` hook
- [x] 2.4 Create `agent-icon.tsx` with provider-to-icon mapping
- [x] 2.5 Create `agent-sidebar-group.tsx` as collapsible sidebar group with "New Session" button
- [x] 2.6 Wire agent sidebar groups into `_app.tsx` sidebar content area
- [x] 2.7 Update barrel exports for daemon and agent systems

## Implementation Details

See TechSpec "File Structure" for exact paths. Follow the app-renderer-systems pattern from web/CLAUDE.md.

The agent sidebar group uses shadcn `SidebarGroup`, `SidebarGroupLabel`, `SidebarGroupAction`, `SidebarMenuSub` components. The "New Session" button per agent will be wired to the create mutation in task 03.

For daemon health, use TanStack Query with `refetchInterval: 10_000`. Connection status is derived: query `isSuccess` = connected, `isError` = disconnected, `isFetching && isError` = reconnecting.

### Relevant Files
- `web/src/components/ui/sidebar.tsx` — SidebarGroup, SidebarMenu*, SidebarMenuSub* components
- `web/src/components/ui/badge.tsx` — For connection status badge
- `web/src/components/ui/collapsible.tsx` — For collapsible agent groups
- `web/src/components/ui/tooltip.tsx` — For collapsed sidebar tooltips
- `web/src/routes/_app.tsx` — Wire sidebar content and header (created in task_01)
- `internal/httpapi/agents.go` — Backend agent list handler (reference for response shape)
- `internal/httpapi/observe.go` — Backend health handler (reference for response shape)
- `.resources/harnss/src/components/sidebar/` — Reference sidebar patterns

### Dependent Files
- `web/src/routes/_app.tsx` — Modified to include agent sidebar and connection status
- `web/src/systems/agent/index.ts` — Updated barrel
- `web/src/systems/daemon/index.ts` — Updated barrel
- Task 03 (session sidebar items) will nest under agent groups created here

## Deliverables
- Complete daemon system with health polling and connection indicator
- Complete agent system with sidebar groups rendering agent list
- Agent icons mapping providers to lucide icons
- Both systems following app-renderer-systems pattern
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for API adapter functions **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `daemon-api.ts` `fetchHealth()` returns parsed HealthPayload on success
  - [ ] `daemon-api.ts` `fetchHealth()` throws on network error
  - [ ] `use-daemon-health` derives "connected" when query succeeds
  - [ ] `use-daemon-health` derives "disconnected" when query errors
  - [ ] `agent-api.ts` `fetchAgents()` returns array of AgentPayload
  - [ ] `agent-api.ts` `fetchAgent(name)` returns single AgentPayload or throws 404
  - [ ] `agent-icon` maps "claude" provider to correct icon component
  - [ ] `agent-icon` returns fallback icon for unknown provider
- Integration tests:
  - [ ] `connection-status.tsx` renders green indicator when daemon is healthy
  - [ ] `connection-status.tsx` renders red indicator when daemon is unreachable
  - [ ] `agent-sidebar-group.tsx` renders one group per agent from mock data
  - [ ] `agent-sidebar-group.tsx` shows "New Session" button in each group
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Sidebar shows agent list when daemon is running at localhost:2123
- Connection status indicator visible in app header
- `make web-typecheck` and `make web-lint` passing
