# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite the Tasks domain list + detail panel surfaces on `@agh/ui` primitives — first screen of Phase 3. Scope covered: `tasks-page-shell`, `tasks-list-panel`, `tasks-empty-state`, `task-card`, the four `tasks-detail-*` files (header/overview/preview/tabs) plus the five tab-body panels (children/dependencies/runs/multi-agent/timeline), plus the new shared `tasks-list-row` primitive. Kanban/Inbox/Dashboard (task 18) and forms/run detail (task 19) are out of scope.

## Important Decisions

- **`taskStatusSignal(status)` is the canonical helper** — maps both production statuses AND mock shorthand (`done | running | pending | blocked | failed`) → `{ tone: StatusDotTone, pulse?: boolean }`. Added to `web/src/systems/tasks/lib/task-formatters.ts` and exported via the systems index. Reused by list row, detail header, detail overview, children/deps/runs/timeline/multi-agent tables.
- **Shared row primitive lives at `tasks-list-row.tsx`** — slots: row button with `StatusDot` + title + `MonoBadge` id + timestamp + optional `Pills` lane badge + optional `trailing` (right-of-timestamp) + optional `footer` (below metadata). `TaskCard` composes it with the rich footer metadata (owner/attempts/children/deps/priority/approval/publish/retry). Kanban cards + Inbox rows in task 18 can bring their own footer.
- **SplitPane ownership lives in `tasks.tsx`** — not in `tasks-list-panel.tsx`. The list panel is self-contained content for `SplitPane`'s list slot; the route owns the SplitPane composition and passes `onDetailClose` to participate in narrow-viewport back-button behavior.
- **Detail header status pill moved into PageHeader title span** — the existing test asserted status text inside `tasks-detail-meta`; updated the test to look for the dedicated `tasks-detail-status` testid instead.
- **Tabs wrapper uses Base UI primitive without Panels** — detail panels are rendered by the route (`tasks.$id.tsx`) based on `page.panel`, not via `TabsContent`. Base UI Tabs emits `aria-selected` natively on `TabsPrimitive.Tab`, so the existing testids/assertions keep working.
- **Preview CodeBlock language switches on `task.kind`** — `kind === "yaml"` → `yaml`, otherwise `markdown`. Preview body composes scope/owner/origin/prompt as a commented mock-shell block. `showPrompt=false` so `$ ` never prefixes content lines.
- **MonoBadge default slot is preserved on the row** — overriding `data-slot="tasks-list-row-id"` masked `mono-badge` from queries. Row differentiation now relies on parent container's `data-slot="tasks-list-row"` scope.
- **Run ids are NOT shortened in detail panels** — `taskShortId` only applies to list rows. Runs and agent cards in the detail tree show the full id in MonoBadge, matching existing test expectations like `tasks-detail-runs-item-run_001 → "run_001"` and `tasks-detail-active-run → "run_active"`.

## Learnings

- `SearchInput` forwards `data-testid` via `{...props}` to the inner `<input>`, not the outer container. Use `screen.getByTestId(...)` directly to get the input element — no `.querySelector('[data-slot="search-input-control"]')` hop required.
- `PageHeader.title` accepts a ReactNode, not just a string — use a `<span>` so the title can include sibling `MonoBadge` + `Pill` + `StatusDot` children.
- Section/Metric/Empty all spread `{...props}` — `data-testid` flows through cleanly.
- Base UI's `TabsPrimitive.Tab` emits `aria-selected` natively on the active trigger; no manual ARIA wiring needed.
- Visual baselines generated cleanly for all 13 new stories (25 png files) on darwin; 196 total web visual snapshots, 0 failures.

## Files / Surfaces

Touched:
- `web/src/systems/tasks/lib/task-formatters.ts` — added `taskStatusSignal` + `taskShortId`.
- `web/src/systems/tasks/index.ts` — exports `TasksListRow`, `TasksListRowProps`, `TaskStatusSignal`, `taskShortId`, `taskStatusSignal`.
- `web/src/systems/tasks/components/tasks-list-row.tsx` (NEW) + `tasks-list-row.test.tsx` (NEW).
- Rewritten: `tasks-page-shell.tsx`, `tasks-list-panel.tsx`, `tasks-empty-state.tsx`, `task-card.tsx`, `tasks-detail-header.tsx`, `tasks-detail-overview-panel.tsx`, `tasks-detail-preview-panel.tsx`, `tasks-detail-tabs.tsx`, `tasks-detail-children-panel.tsx`, `tasks-detail-dependencies-panel.tsx`, `tasks-detail-runs-panel.tsx`, `tasks-multi-agent-panel.tsx`, `tasks-timeline-panel.tsx`.
- Updated tests: `task-card.test.tsx`, `tasks-page-shell.test.tsx`, `tasks-list-panel.test.tsx`, `tasks-detail-header.test.tsx`, `tasks-detail-preview-panel.test.tsx`.
- `web/src/routes/_app/tasks.tsx` — now composes SplitPane with list/detail/detailEmpty slots.
- `web/src/routes/_app/-tasks.test.tsx` — updated heading-by-role → testid assertion.
- `web/src/routes/_app/-tasks.router.integration.test.tsx` — added list → detail selection integration test.
- NEW stories folder `web/src/systems/tasks/components/stories/` with `fixtures.ts` + 6 stories (list-row, list-panel, empty-state, detail-header, detail-overview-panel, detail-preview-panel, detail-tabs).
- 25 Playwright baselines committed under `web/tests/visual/__snapshots__/` (all `systems-tasks-*-chromium-darwin.png`).

## Errors / Corrections

- First `tasks-list-row.test.tsx` failed because I overrode `data-slot="mono-badge"` with `data-slot="tasks-list-row-id"` on the identifier badge. Fix: dropped the override so MonoBadge's default slot stays intact.
- `tasks-list-panel.test.tsx` search event failed with `Unable to fire a "change" event - please provide a DOM element`. Root cause: `data-testid` on `SearchInput` lands on the inner `<input>`, not the container. Fix: query the testid directly as the input element.
- `make lint` (Go) fails on two pre-existing issues (gosec SQL in `observe/tasks.go`, gocyclo in `globaldb/global_db_task_aux.go`). These are documented open risks in shared memory; task 17 touches zero Go files, so the failures are not in scope.

## Ready for Next Run

- **Shared list-row contract for task 18.** `TasksListRow({ task, selected, onSelect, lane, trailing, footer, testId })` — Kanban cards and Inbox rows must reuse this primitive with their own `trailing`/`footer` content to keep the status-dot + title + mono-id + timestamp row visual consistent.
- **Status signal helper is canonical.** Task 18/19 should consume `taskStatusSignal(status)` + `taskShortId(task)` rather than re-deriving status→tone/pulse mappings per-view.
- **SplitPane narrow-viewport close wiring** — `onDetailClose` resets `selectedTaskId` AND navigates back to `/tasks` when a child route is active. Task 18 Kanban view does not use SplitPane (full-width board), but task 19's run-detail must respect the same `onDetailClose` contract.
- **Detail Tabs primitive is controlled** — `TasksDetailTabs` wraps Base UI `Tabs` with `value` + `onValueChange` but does NOT render `TabsContent`. The route renders panel bodies based on `page.panel` state. Task 19's run-detail tabs should adopt the same pattern.
