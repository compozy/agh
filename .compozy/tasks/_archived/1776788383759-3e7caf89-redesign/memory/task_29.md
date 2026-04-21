# Task Memory: task_29.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite `web/src/systems/daemon/components/connection-status.tsx` (already a thin wrapper over `@agh/ui` `ConnectionIndicator` — verified mid-audit) and rewrite `web/src/routes/_app/index.tsx` from the legacy `Terminal` empty shell into a derived-per-ADR-004 home dashboard composed entirely from `@agh/ui` primitives. Add a route-level view-model hook aggregating daemon health, workspace count, agent count, and active session count. Surface daemon `StatusDot` tone mapping (success/warning/danger/neutral) + persistent `ConnectionIndicator`. Cover loading/error/disconnected/empty branches with stories + Playwright baselines + unit tests.

## Important Decisions

- **Route hook lives at `web/src/hooks/routes/use-home-page.ts`** matching the existing `use-skills-page.ts` / `use-app-layout.ts` location convention. Index route stays purely presentational; the hook owns derived metrics + status-tone mapping.
- **Daemon status tone derives from `connectionStatus` + `health.status`**: connected + ok|healthy|running → `success`/healthy, connected + other status → `warning`/degraded, disconnected → `danger`/disconnected, otherwise → `neutral`/unknown. Distinct from the route's `connectionStatus` pill, which maps directly through `ConnectionIndicator`.
- **Active sessions count uses `useSessions(activeWorkspaceId)`** scoped to the active workspace (matching sidebar behavior) rather than reading `health.active_sessions` (which is daemon-wide). Falls back to `health.active_sessions` if no workspace selected.
- **Uptime formatter `formatUptimeSeconds` exported from the route hook** as a private helper rather than promoting to `@agh/ui` or a shared lib — only the home dashboard renders this surface today.
- **Connection-status component is unchanged** — already imports from `@agh/ui` directly. Subtask 29.2 was a verification, not a rewrite. No legacy `@/components/design-system/connection-indicator` import existed.
- **Disconnected daemon section swaps the StatusDot card for an `Empty` card** with `ServerOff` icon + a `ConnectionIndicator status="disconnected"` as the title + recovery-hint description. This keeps the disconnected branch visually distinct (read: "the dashboard isn't going to update") while reusing canonical primitives.
- **Loading branch renders a custom skeleton row inside the daemon section + 4 skeleton metric cards** rather than the `Loader2` spinner pattern used by other domain pages. Each metric skeleton mirrors the production card chrome (border + radius + padding) so the visual reflows are minimal once data arrives.

## Learnings

- **Route stories that need a "loading" baseline can use MSW `delay("infinite")`** on every backing endpoint (daemon + workspace + agent). The `useDaemonHealth` hook reports `connectionStatus = "disconnected"` during the very first pending tick (because `query.isSuccess === false` and there is no error yet), so the route's `isLoading` gate must be checked BEFORE any disconnect-based branching — otherwise the disconnected card flashes before the skeleton.
- **The session route's `NotFoundRedirect` story implicitly snapshots the home dashboard.** The story navigates to `/session/sess-missing`, gets a 404, and TanStack Router redirects to `/`. Any rewrite of the home dashboard regenerates that baseline as a downstream effect — expect the redirect baseline to update alongside the index baselines on every dashboard pass.
- **`makeHome()` test factory inside the route test mocks `useHomePage` directly** (instead of mocking each downstream domain hook). Saves ~200 lines of mock setup and keeps the route test purely about presentational logic; the hook itself is covered separately by `use-home-page.test.tsx` against the real adapters.

## Files / Surfaces

- `web/src/systems/daemon/components/connection-status.tsx` — verified, no rewrite needed (already on `@agh/ui` ConnectionIndicator).
- `web/src/systems/daemon/components/connection-status.test.tsx` — preserved as-is.
- `web/src/systems/daemon/components/stories/connection-status.stories.tsx` — preserved (existing healthy/disconnected/reconnecting variants already cover the spec).
- `web/src/routes/_app/index.tsx` — full rewrite as home dashboard.
- `web/src/routes/_app/-index.test.tsx` (new) — 11 specs covering happy path, tone mapping, loading skeletons, error Empty, disconnected state, version badge presence/absence.
- `web/src/hooks/routes/use-home-page.ts` (new) — aggregates daemon/workspace/agent/session hooks. Exports `useHomePage`, `formatUptimeSeconds`, plus `HomePageView`/`HomeMetricEntry`/`DaemonStatusDescriptor` types.
- `web/src/hooks/routes/use-home-page.test.tsx` (new) — 10 specs covering happy/degraded/disconnected/error/no-workspace branches + the uptime formatter.
- `web/src/routes/_app/stories/-index.stories.tsx` — extended from 2 to 6 stories: Default + Degraded + Disconnected + Loading + Error + Onboarding.
- `web/tests/visual/__snapshots__/routes-app-stories-index--*` — 1 refreshed baseline (default) + 4 new baselines (degraded/disconnected/error/loading); onboarding baseline unchanged.
- `web/tests/visual/__snapshots__/routes-app-stories-session--not-found-redirect-chromium-darwin.png` — refreshed downstream of the home rewrite (404 redirects to `/`).

## Errors / Corrections

- **First draft used `void sessionsError` to silence an unused destructured variable** — wasted effort. Cleaner to drop the destructure entirely and let TanStack Query's stale-while-revalidate behavior degrade silently when `useSessions` errors.

## Ready for Next Run

- Task 30 (Settings shell rewrite) starts the Phase 6 settings sweep. Settings already use `useSettingsPage` etc. — same route-hook pattern as `useHomePage`.
- The home dashboard intentionally does NOT surface bridges/dream/memory metrics (out of scope per task spec). When a future task wants to add them, extend the `HomeMetricEntry` union and the `metricsByKey` rendering map; no hook signature changes needed.
