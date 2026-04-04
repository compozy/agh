# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Build daemon and agent systems end-to-end: API adapters, TanStack Query hooks, UI components for health polling, connection status, and agent sidebar groups.

## Important Decisions

- ConnectionStatus component in daemon system is a thin wrapper over existing `ConnectionIndicator` design-system component — avoids duplicating design-system primitives.
- `useDaemonHealth` hook derives connection status from TanStack Query state: isSuccess=connected, isFetching&&isError=reconnecting, else=disconnected.
- Agent sidebar uses `Collapsible` + `SidebarGroup` composition with `SidebarGroupLabel` rendered as `CollapsibleTrigger`.
- Connection status displayed in both app header (right-aligned) and sidebar footer for visibility in both expanded/collapsed sidebar states.
- `AgentsList` is a separate component inside `app-sidebar.tsx` (not extracted to systems/) since it handles loading/error/empty states specific to the sidebar context.

## Learnings

- Collapsible component uses `data-[panel-open]` attribute for open state (not `data-[state=open]` like Radix) — this is base-ui/react specific.
- `SidebarGroupLabel` accepts `render` prop for custom element composition (e.g., rendering as CollapsibleTrigger).

## Files / Surfaces

- `web/src/systems/daemon/adapters/daemon-api.ts` — new
- `web/src/systems/daemon/lib/query-keys.ts` — new
- `web/src/systems/daemon/lib/query-options.ts` — new
- `web/src/systems/daemon/hooks/use-daemon-health.ts` — new
- `web/src/systems/daemon/components/connection-status.tsx` — new
- `web/src/systems/daemon/index.ts` — updated barrel
- `web/src/systems/agent/adapters/agent-api.ts` — new
- `web/src/systems/agent/lib/query-keys.ts` — new
- `web/src/systems/agent/lib/query-options.ts` — new
- `web/src/systems/agent/hooks/use-agents.ts` — new
- `web/src/systems/agent/components/agent-icon.tsx` — new
- `web/src/systems/agent/components/agent-sidebar-group.tsx` — new
- `web/src/systems/agent/index.ts` — updated barrel
- `web/src/components/app-sidebar.tsx` — modified (agent list + connection status)
- `web/src/components/app-header.tsx` — modified (connection status indicator)

## Errors / Corrections

None.

## Ready for Next Run

- Task 03 should add session items inside the `SidebarMenuSub` within `AgentSidebarGroup`. The "No sessions" placeholder is already in place.
- `onNewSession` callback on `AgentSidebarGroup` is ready to be wired to session create mutation.
