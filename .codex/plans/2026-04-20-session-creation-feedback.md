# Session Creation Pending Feedback

## Summary

- Add explicit pending feedback to the sidebar create-session flow.
- Replace the current invisible “buttons are disabled” behavior with a per-agent spinner and a temporary `starting...` row.
- Fix the underlying flow by separating cache/mutation responsibilities from navigation timing so the UI remains deterministic during creation.

## Implementation Changes

- `web/src/systems/session/hooks/use-session-actions.ts`
  - Remove embedded navigation from `useCreateSession`.
  - Keep the mutation data-oriented: create the session, seed `sessionKeys.detail(session.id)`, merge the created session into session list caches without duplication, and trigger background invalidation.
- `web/src/hooks/routes/use-app-layout.ts`
  - Orchestrate creation with `mutateAsync`.
  - Track `pendingSessionAgentName` and `pendingSessionWorkspaceId` locally.
  - Clear pending state only after `navigate({ to: "/session/$id", ... })` completes or the mutation fails.
  - Surface failures with `toast.error(...)`.
- `web/src/components/app-sidebar.tsx`
  - Thread the pending agent/workspace through the sidebar tree.
  - Show a spinner instead of `+` on the clicked agent.
  - Render a temporary non-clickable `starting...` row only under the agent currently being created.
  - Keep other create buttons disabled while a creation is in flight.

## Public APIs / Interfaces / Types

- `useCreateSession()` becomes a pure mutation hook and no longer owns route navigation.
- `AppSidebarProps` adds `pendingSessionAgentName: string | null` and `pendingSessionWorkspaceId: string | null`.
- Agent/sidebar internal props gain the pending-session inputs needed to render contextual loading UI.

## Test Plan

- `web/src/components/app-sidebar.test.tsx`
  - Spinner replaces the clicked agent’s `+` while a session is pending.
  - Temporary `starting...` row renders only under the pending agent in the active workspace.
  - Other create buttons remain disabled during the pending state.
- `web/src/hooks/routes/use-app-layout` tests
  - Pending state is set before the mutation resolves.
  - Pending state survives until navigation completes.
  - Pending state clears and `toast.error` fires on failure.
- `web/src/systems/session/hooks/use-session-actions` tests
  - Created session seeds detail cache and merges into list cache without duplication.
  - Lists still invalidate for background revalidation.

## Assumptions

- Only one create-session mutation runs at a time.
- The temporary row label is `starting...`.
- Error feedback follows existing `sonner` toast patterns already used in the web app.
