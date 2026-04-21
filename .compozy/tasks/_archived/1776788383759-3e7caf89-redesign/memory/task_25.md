# Task Memory: task_25.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite `web/src/systems/bridges/**` + `web/src/routes/_app/bridges.tsx` on `@agh/ui` primitives: `PageHeader` + `SplitPane` shell, grouped list, `Section` + `Metric` + `Table` detail, dialogs on `Dialog` + `Field`. Preserve every hook contract and `data-testid` used by existing tests + e2e.

## Important Decisions

- **Event stream `Table` = routes table.** Bridges have no daemon-emitted event stream yet; routes are the closest operator-meaningful stream. Table columns: Status · Agent · Target · Scope · Last activity. Empty state preserves `bridge-routes-empty` testid.
- **4-tile Metric layout.** Events (24h) = `backlog+failures+dropped+routes`; Success rate = `routes / total` with `success≥90 → default≥70 → warning` ramp; Last delivery = `formatBridgeRelativeTime(health.last_success_at)`; Active routes = `health.route_count ?? routes.length`. Testids: `bridge-metric-{events-24h|success-rate|last-delivery|active-routes}`.
- **Disabled status gets danger tone.** `statusToStatusDotTone` / `statusToMonoBadgeTone` both short-circuit on `status === "disabled"` to `danger` — `bridgeStatusTone` maps disabled to `neutral` by default but DESIGN.md + task spec mandate `danger`. "Send Test" button is `disabled={effectiveStatus === "disabled"}`.
- **Dialog `onSubmit` delegates validation to the hook.** My initial dialogs early-returned when `!canSubmit || isPending`, but the pre-existing `-bridges.test.tsx` fires `fireEvent.submit()` and expects `useBridgesPage.handleCreateBridge` to emit the validation toast. Reverted to unconditional `onSubmit()` so hook-level toast path still fires; submit button's `disabled` attribute remains the visible guard.
- **Grouped list structure.** `BridgeListPanel` groups by `extension_name::platform` with header `bridge-list-group-header-{ext}-{platform}`. Groups sorted by platform name ASC; bridges within a group render in input order (parent sorts). Tests use regex `^bridge-list-group-header-` because `^bridge-list-group-` matches both the wrapper and header testids.
- **Stale baseline cleanup.** Renamed stories (`Error` → `InvalidProviderConfig`, empty `Default` → `WithProviders`/`WithoutProviders`/`AllProvidersUnavailable`) left orphaned PNGs. Removed: `bridgecreatedialog--error`, `bridgeeditdialog--error`, `bridgetestdeliverydialog--error`, `bridgeemptystate--default`.

## Learnings

- MonoBadge auto-uppercases — one pre-existing test expected lowercase "bound"; updated the test to match the new "BOUND" rendering. Same rule will bite future tasks migrating from `<Pill>{status}</Pill>` → `<MonoBadge>{status}</MonoBadge>`.
- When removing `WorkspacePageShell` usage, the corresponding `vi.mock("@/systems/workspace", ...)` in a route test must drop the shell mock entry but keep `useActiveWorkspace`. Otherwise `useActiveWorkspace` from the real module tries to hit the real `@tanstack/react-query` and the test explodes with missing QueryClient.

## Files / Surfaces

- Rewritten: `web/src/systems/bridges/components/{bridge-list-panel,bridge-detail-panel,bridge-create-dialog,bridge-edit-dialog,bridge-test-delivery-dialog,bridge-empty-state,bridge-provider-card}.tsx`
- Rewritten: `web/src/routes/_app/bridges.tsx`
- Updated: `web/src/routes/_app/-bridges.test.tsx` (dropped `WorkspacePageShell` mock; all 9 route tests green)
- New tests: `web/src/systems/bridges/components/{bridge-list-panel,bridge-empty-state,bridge-test-delivery-dialog,bridge-provider-card}.test.tsx` (+3 new detail-panel tests, +2 new edit-dialog tests)
- Stories updated: all 7 `components/stories/*.stories.tsx` + `src/routes/_app/stories/-bridges.stories.tsx` (fixed stale `bridge-test-delivery-open` → `open-test-delivery-btn` testid)
- Visual baselines: 26 bridge-related darwin PNGs in `web/tests/visual/__snapshots__/` (4 stale removed, 22 current states)

## Errors / Corrections

- Regex `^bridge-list-group-` matched both `bridge-list-group-{ext}-{platform}` and `bridge-list-group-header-{ext}-{platform}`. Fixed to `^bridge-list-group-header-` for the group-count assertion.

## Ready for Next Run

- Bridges domain phase-5 visual rewrite is complete and fully covered by unit + visual tests. The next phase-5 task (Knowledge — task_26) can reuse the same PageHeader + Pills + SplitPane + Section + Metric + Table composition template.
