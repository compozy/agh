# Task Memory: task_18.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite the Tasks domain Kanban (4 columns), Dashboard (metrics + queue chart + status breakdown + active runs), and Inbox (approval flow) views on `@agh/ui` primitives, sharing the `TasksListRow` row from task 17 across Kanban cards and Inbox items.

## Important Decisions

- **Kanban column model collapses 6 → 4.** Task 17 left `getKanbanColumns()` at `pending | ready | in_progress | blocked | completed | failed`. Task 18's spec requires `Pending | Running | Done | Failed`. The grouping helper now folds `draft | pending | ready | blocked` into `pending`, `in_progress` into `running`, `completed` into `done`, and `failed | canceled` into `failed`. Mock shorthand `running` and `done` are accepted too, so the mock fixtures in `docs/design/web-inspiration/src/pages-core.jsx` and the spec's "task with `status === 'running'`" test both route correctly.
- **`TasksListRow` changed from `<button>` to `<div role="button">`.** Kanban cards and Inbox items both need action buttons inside the row footer (Retry, Approve/Reject, Open). HTML forbids `<button>` nesting, so the outer element became a `<div>` with `role="button"`, `tabIndex={0}`, Enter/Space handlers, and `aria-pressed` always set when `selected` is passed. `clickable` gate (derived from `onSelect !== undefined`) controls whether the element becomes a button role and takes tab focus; non-clickable rows still advertise selection via `aria-pressed`. The primitive lived in task 17 with a "do not modify without cross-PR coordination" note — task 18 IS that coordination.
- **Dashboard card set rewritten to the spec vocabulary.** Previous cards surfaced In Progress / Blocked / Failed / Latency. Task 18 requires Active runs / Success rate / Average duration / Queue depth. Success rate is computed from `totals.completed_runs / (completed + failed + canceled)`; falls back to `"—"` when no runs have concluded. Average duration borrows claim-latency average from `cards.latency.claim_latency_ms.average_ms` until a run-duration aggregate exists in the payload.
- **Queue-health card keeps the existing 24h histogram + health metrics.** The top-row primitives became `Section` + `Pill`, and the bar chart falls back to `Empty` when no buckets are provided. The component now accepts an optional `buckets?: QueueBucket[]` prop so Storybook + visual tests can drive the chart deterministically; in production the helper `deriveBuckets(dashboard)` seeds 24 hourly buckets from the dashboard payload.
- **Inbox item action variants.** Mapping per spec: Approve → `Button variant="default"` (the @agh/ui primary), Reject → `Button variant="destructive"` (mapped from "danger"), ghost-ish rail actions (Retry, Dismiss, Mark read, Archive) → `Button variant="ghost"`, `Open` → `Button variant="outline" render={<Link>}` with `nativeButton={false}` so Base UI stops warning about non-button render targets. Each action button gets a `data-variant` attribute (primary/ghost/danger) so tests can assert the spec mapping without reading Tailwind classes.
- **`SearchInput` + `Switch` in the inbox toolbar.** Replaced the bespoke search input + native checkbox with `@agh/ui` `SearchInput` + `Switch`. SearchInput emits a `string` (not a DOM event), so the inbox view wraps it with `onChange={next => onSearchChange(next)}`. `Switch` exposes `role="switch"`, so the existing inbox-view test now queries the switch by `role=switch` inside the unread-toggle label, and asserts `onToggleUnread.mock.calls[0]?.[0] === true` rather than `toHaveBeenCalledWith(true)` — Base UI's Switch onCheckedChange passes an event details object as the 2nd argument.
- **Inbox lane tabs on Base UI Tabs.** Replaced the custom LaneTab buttons with `@agh/ui` `Tabs` + `TabsList variant="line"` + `TabsTrigger`. Each trigger keeps its old `data-testid` and inline count/unread badges so the existing tests continue to pass. Base UI's Tab fires `onValueChange` on `fireEvent.click`.

## Learnings

- **`@agh/ui` `Button` does not expose `primary/danger` variants — those are `default` and `destructive`.** The task spec describes the Approve/Edit/Reject actions as `primary / ghost / danger`; that vocabulary is maintained at the call-site via an ActionButton helper that maps `primary → default`, `danger → destructive`, `ghost → ghost`, and surfaces the original intent through a `data-variant` attribute so tests stay semantic.
- **Base UI `Button` warns when `render={<Anchor>}` is passed without `nativeButton={false}`.** Setting `nativeButton={false}` silences the "expected a native <button>" warning and lets the `<Link>` render target carry Base UI's button behaviors (focus ring, active state) over an `<a>` element.
- **Base UI `Switch` + jsdom quirks.** Clicking the switch via `fireEvent.click` invokes `onCheckedChange` with `(next, eventDetails)`; asserting with `toHaveBeenCalledWith(true)` fails because the event details is the 2nd argument. Inspect `mock.calls[0]?.[0]` instead.

## Files / Surfaces

- `web/src/systems/tasks/lib/task-grouping.ts` / `.test.ts` — collapsed to 4 columns; accepts mock status aliases.
- `web/src/systems/tasks/components/tasks-kanban-board.tsx` / `.test.tsx` — rewritten on `Section` columns + `TasksListRow` cards; columns are `role="listitem"` inside a `role="list"` board so the test can count exactly four columns.
- `web/src/systems/tasks/components/tasks-dashboard-view.tsx` — Section-centric layout, no behavioral changes beyond spacing tokens.
- `web/src/systems/tasks/components/tasks-dashboard-cards.tsx` / `.test.tsx` — 4 `Metric` primitives with the new label set and computed success rate.
- `web/src/systems/tasks/components/tasks-dashboard-active-runs.tsx` — `Section` shell + StatusDot/MonoBadge/Pill row composition.
- `web/src/systems/tasks/components/tasks-dashboard-status-breakdown.tsx` / `.test.tsx` — `Section` + `Pill`-per-status with shared count; new test asserts pill sum equals total.
- `web/src/systems/tasks/components/tasks-dashboard-queue-health.tsx` / `.test.tsx` — `Section` + bar histogram with `Empty` fallback; new `buckets` prop for deterministic testing.
- `web/src/systems/tasks/components/tasks-inbox-view.tsx` — SearchInput + Switch toolbar; lane tabs + TasksInboxItem body preserved.
- `web/src/systems/tasks/components/tasks-inbox-lane-tabs.tsx` — rewritten on `@agh/ui` Tabs + TabsTrigger, keeps per-lane test ids.
- `web/src/systems/tasks/components/tasks-inbox-item.tsx` / new `.test.tsx` — shared `TasksListRow` + StatusDot unread + variant-tagged action buttons.
- `web/src/systems/tasks/components/tasks-list-row.tsx` — outer `<button>` converted to `<div role="button" tabIndex=0>` with keyboard handlers so nested action buttons are valid HTML.
- `web/src/hooks/routes/use-tasks-page.test.tsx` — updated the pending-column assertion to reflect the 4-column collapse (2 tasks instead of 1).
- Storybook stories: `tasks-kanban-board.stories.tsx`, `tasks-dashboard-view.stories.tsx`, `tasks-inbox-view.stories.tsx` (populated/empty/loading/error, plus a backlog-warning dashboard variant).
- Playwright baselines: 13 new PNGs under `web/tests/visual/__snapshots__/` for the three views (darwin chromium).

## Errors / Corrections

- First pass used a `react:` "primary" Button variant; `@agh/ui` Button does not expose that variant. Switched to the `default`/`destructive`/`ghost` triad and surfaced the spec vocabulary through `data-variant`.
- First pass of TasksListRow remained a `<button>`; nested action buttons produced the "button cannot be descendant of button" React warning. Converted to `<div role="button">` with explicit keyboard handling.
- First SearchInput wiring called `event.target.value` — `@agh/ui` SearchInput emits `(next: string)` directly; the view now just forwards the string.

## Verification Evidence

- `bun run --cwd web typecheck` — passes (0 errors).
- `bun run --cwd web lint` — 0 warnings, 0 errors.
- `bun run --cwd web test` — **176 test files, 1252 tests passing** (includes the new tasks-inbox-item, status-breakdown, queue-health tests + the rewritten kanban/dashboard-cards tests).
- `bun run --cwd web test:visual --grep tasks` — 38 passed, 13 new baselines committed.
- `make verify` — **Go lint + Go test failures are pre-existing on the base branch and unchanged by task 18** (see Open Risks in shared memory). Confirmed by re-running `make lint` after `git stash` of all task-18 changes: the same `internal/store/globaldb/global_db_task_aux.go` gocyclo and `internal/observe/tasks.go` gosec errors appear on the bare branch. Task 18 touches zero Go code.
- Success-criteria spot check: `rg "@/components/(ui|design-system)/" web/src/systems/tasks/components/tasks-{kanban,dashboard,inbox}*` returns 0 matches; `TasksListRow` is imported by both `tasks-kanban-board.tsx` and `tasks-inbox-item.tsx`.

## Ready for Next Run

- Task 19 (forms + run detail) can proceed. `TasksListRow` is now a clickable `<div role="button">` that tolerates nested action buttons in its trailing/footer slots.
- Web-visual CI will need to seed 13 new linux baselines on the first `ubuntu-22.04` `--update-snapshots` run for the new Kanban/Dashboard/Inbox stories.
