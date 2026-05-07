# UI/UX Module Overview: `04_tasks`

> **Status:** draft
> **Owner subagent:** `tasks-module-auditor`
> **Date:** 2026-05-06
> **Module:** Tasks (`04_tasks`)
> **Routes covered:** `/tasks`, `/tasks/new`, `/tasks/$id`, `/tasks/$id/edit`, `/tasks/$id/runs/$runId`
> **System owners:** `web/src/systems/tasks/`, `web/src/systems/automation/`
> **Live URL probed:** `http://localhost:3000`
> **Storybook URL probed:** `http://localhost:6006`

---

## 1. Module purpose

Tasks is the operator surface for the autonomy kernel. A task is the durable contract of work; one or more `task_run` rows execute against it. Operators (and agents through the same surfaces) create tasks, publish or start them, watch their runs, inspect lifecycle state, and follow the run to a session, a review, or a terminal status. The module is CRUD-heavy at the entry point (`/tasks` list, `/tasks/new`, `/tasks/$id`, `/tasks/$id/edit`) and live-data-heavy on the leaf (`/tasks/$id/runs/$runId`).

The five routes share one chrome (`TasksPageShell` with mode pills LIST / KANBAN / DASHBOARD / INBOX and a Task CTA) and one detail surface that swaps tab panels (Overview, Runs, Events, Agents, Children, Dependencies, Orchestration). The create and edit routes share `TaskEditorSurface`, with `mode="create" | "edit"` switching the right rail and the action footer.

Source map:

- `web/src/routes/_app/tasks.tsx` (list shell, owns the SplitPane and mode switcher)
- `web/src/routes/_app/tasks.new.tsx` (create page; renders `TaskEditorSurface`)
- `web/src/routes/_app/tasks.$id.tsx` (detail page; owns the Tabs and panel router)
- `web/src/routes/_app/tasks.$id.edit.tsx` (edit page; renders `TaskEditorSurface`)
- `web/src/routes/_app/tasks.$id.runs.$runId.tsx` (run detail; owns Identity / Progress / Activity / Reviews)

---

## 2. Cross-route consistency observations

### 2.1 Shell composition is correct, child-rendering is broken

The parent `/tasks` route always wraps the body in `TasksPageShell`. When a child route is mounted (`/tasks/new`, `/tasks/$id`, `/tasks/$id/edit`, `/tasks/$id/runs/$runId`), `tasks.tsx:24-31` flips `surfaceMode` to `"list"` and `showDetailPreview` to `false`, but it still renders the `<SplitPane>` with the `TasksListPanel` populated in the list slot and the child `<Outlet />` placed in the detail slot (`tasks.tsx:194-202`).

That makes sense for `/tasks/$id` (the list rail keeps context). It is wrong for `/tasks/new`, `/tasks/$id/edit`, and `/tasks/$id/runs/$runId`, which are full surfaces that should not be jammed against a list rail. Evidence: the create page renders both a "Nothing matches the current filters" empty list (with its own search box, lane pills, and `New task` button) and the create form (`_evidence/tasks-new/live.png`). The edit page does the same (`_evidence/tasks-id-edit/live-missing.png`), as does the run detail (`_evidence/tasks-id-runs-runId/live-not-found.png`). On `/tasks/$id/edit` the user sees two `New task` calls to action, two breadcrumbs, and a fake list of zero tasks next to the form they are editing.

This is the highest-leverage IA bug in the module. It affects three routes at once.

### 2.2 State-machine vocabulary is inconsistent across surfaces

The runtime exposes two state taxonomies that the UI has to expose:

- Task status (`TaskStatus`): `draft | pending | blocked | ready | in_progress | completed | failed | canceled`.
- Task run status (`TaskRunStatus`): `queued | claimed | starting | running | completed | failed | canceled`.

The UI also synthesises a "lifecycle phase" (`task-formatters.ts:339-457`): `saved_intent | awaiting_approval | ready_to_start | queued | running | completed | failed | canceled | blocked`. The lifecycle is a UI narrative on top of the two real taxonomies.

Concrete mismatches the audit caught:

- Detail header status chip uses sentence-case label (`In Progress`) via `taskStatusLabel` (`tasks-detail-header.tsx:99`).
- Run detail Identity panel renders status raw lowercase (`running`) without the label map (`task-run-detail-panels.tsx:53`).
- Run detail header chip renders status raw lowercase (`running`) (`task-run-detail-header.tsx:153`).
- Status badges through the list and runs table render via `Pill tone={...}` plus the raw status string, so the visual register depends on whether the surface routed through `taskStatusLabel` or not. Some say `In Progress`, others `running`, others `RUNNING`.

The task surface should pick one casing and one label set per status namespace and apply it everywhere. `DESIGN.md` §4 says status badges are JetBrains Mono uppercase. That makes `RUNNING`, `IN PROGRESS`, `COMPLETED` the canonical render; the labels exist but the consistent uppercase rendering is missing.

### 2.3 Status tone vs. status pulse mismatch

`taskStatusSignal` (`task-formatters.ts:32-50`) returns `tone: "accent", pulse: true` for `in_progress`, `running`. `taskStatusTone` (`task-formatters.ts:125-145`) returns `neutral` for the same `in_progress`. So a running row gets a pulsing accent dot next to a neutral status chip. That is intentional per the comment in source, but it produces visually inconsistent semantics: the dot says "live", the chip says "calm". An operator scanning the list cannot rely on chip color alone.

### 2.4 Cancel action verbs differ between task and run

The detail header offers a `Cancel` button (`tasks-detail-header.tsx:153`). The run detail header offers a `Kill run` button (`task-run-detail-header.tsx:135`). Both call run-cancellation APIs. `Kill` is aggressive operator slang and breaks `COPY.md` voice and `glossary.md` terminology. Pick one verb (`Cancel run`) and use it everywhere.

### 2.5 Duration formatter overflows in minutes

`formatElapsed` is duplicated in `task-run-detail-header.tsx:16-41` and `task-run-detail-panels.tsx:117-142`. Both functions only emit `Xs`, `Ym Zs`. Anything longer than 60 minutes still prints as minutes. The Storybook `Running` story has `started_at: 2026-04-17T09:59:00Z` against today, so the live render shows `28308m 15s` and the run detail header chip reads `28308M 15S`. This is the most embarrassing bug in the module: the run detail page will never show realistic durations for tasks that ran yesterday or last week, let alone live runs that have been claimed for an hour.

### 2.6 Em dashes throughout the copy

Multiple components ship em dashes in copy strings, breaking the audit hard rule and the `DESIGN.md` Copy section ban:

- `tasks-detail-runs-panel.tsx:67` empty title: `"Saved intent only — no runs yet"` and `tasks-detail-runs-panel.tsx:108` channel tooltip uses `—`.
- `tasks-detail-overview-panel.tsx:95` channel tooltip uses `—`.
- `tasks-detail-header.tsx:111` channel tooltip uses `—`.
- `task-formatters.ts:435-437` lifecycle descriptions: `"Retry, cancel, or follow up — channel chatter never owns status."`.
- `task-templates.ts:51` description: `"Bind a cron or schedule from Automation — re-enqueues a run every tick."`.
- `task-formatters.ts:140` empty cell separator: `"—"` (filling table cells when no value).

The em dashes appear in user-visible copy and in tooltips. Removing them is mechanical (replace with a comma, period, or colon). Evidence: `_evidence/tasks/live-empty.png` shows `Bind a cron or schedule from Automation — ...`; `/tmp/sb-overview.txt` snapshot of the detail page shows the channel tooltip content.

### 2.7 Storybook drift on the populated list story

The `routes-app-stories-tasks--default-list` Storybook story renders a flat grouped vertical list with `PENDING / RUNNING / DONE / FAILED` headings (`_evidence/tasks/sb-default-list.png`). The live `/tasks` route renders `TasksListPanel` inside a `SplitPane`, which is a compact rail with a shared status filter, search, and lane pills (visible at `_evidence/tasks/live-empty-1440.png` for empty state, and the rail shape stays the same when populated). The Storybook output and the live route diverge in component, layout, and hierarchy. This is misleading reference material for designers and QA.

### 2.8 Form vs. list sharing breaks focus management

When the create or edit route opens, the user starts on the form but the parent SplitPane keeps a tab-targetable list rail (search box, lane pills, `New task` button). Tab order will hit list controls before reaching the form's required title field. This degrades keyboard flow for the routes whose primary job is the form.

### 2.9 No streaming indicator on the run detail leaf

`/tasks/$id` exposes a `LIVE` badge on the Events tab when the task is streaming (`tasks.$id.tsx:74`). `/tasks/$id/runs/$runId` does not show any streaming or connection status. The Activity panel renders `last_event_type` and `last_activity_at`, but a stale connection produces a stale render with no banner. For a route whose primary job is "watch the run", this is missing system status (Nielsen #1).

### 2.10 Error-state coverage is uneven

- `/tasks` list panel handles loading, error, and empty (`tasks-list-panel.tsx:127-159`).
- `/tasks/$id` handles loading, not-found, and fatal-error (`tasks.$id.tsx:34-66`).
- `/tasks/$id/edit` handles loading and "missing-task" but the missing-task branch is gated on `page.task` and `page.isInitialized` so a 404 from the API can leave the page on the spinner indefinitely (live probe stayed on `Loading task…`). Evidence: `_evidence/tasks-id-edit/live-missing.png`.
- `/tasks/$id/runs/$runId` handles loading and not-found, but the live probe of a missing run id rendered no error message because the parent `/tasks/$id` reached its own not-found state first and returned. Evidence: `_evidence/tasks-id-runs-runId/live-not-found.png`.

### 2.11 Truthful UI assessment

The runtime exposes the operations the surface claims. Lifecycle phases match the autonomy kernel glossary (saved intent, awaiting approval, queued, running, completed, failed, canceled, blocked), and the actions exposed (publish, start run, cancel, edit, delete) map to the documented service methods. There is no fake `Schedule` or `Trigger` button on the task surface (those are correctly delegated to the Automation system, with the recurring template noting "Configure the schedule from the Automation area"). The truthful-UI failures are limited to:

- The `28308m 15s` elapsed value: it is technically computed from runtime data, but the formatter rounds to a meaningless number; the value displayed is not the value the daemon would call elapsed for a 19-day-old run.
- The `Kill run` verb implies a forceful kernel stop; the underlying API is `cancelTaskRun`, which is closer to "cancel and release lease" than "kill".

### 2.12 Glossary discipline

Vocabulary mostly aligns with `docs/_memory/glossary.md`:

- `task`, `task_run`, `claim_token_hash`, `lease`, `coordinator handoff`, `coordination channel` are all used correctly.
- The word `recipe` does not appear in any audited file.
- One drift: `task-run-detail-header.tsx:135` uses `Kill run` instead of `Cancel run`. Glossary uses cancel/release verbs.

---

## 3. Cross-route component reuse map

| Component | Used by | Notes |
| --- | --- | --- |
| `TasksPageShell` | All five routes via parent `/tasks` | Always renders the shell; child routes inherit it. |
| `TasksListPanel` | `/tasks`, AND incorrectly `/tasks/new`, `/tasks/$id/edit`, `/tasks/$id/runs/$runId` (parent SplitPane bug). | See 2.1. |
| `TasksDetailPreviewPanel` | `/tasks` only (preview rail when a list row is selected). | Stays out of child routes. |
| `TasksDetailHeader` | `/tasks/$id`. | Owns Edit, Cancel, Publish, Start run. |
| `TasksDetailTabs` + 7 panels | `/tasks/$id`. | Tabs are `Overview, Runs, Events, Agents, Children, Dependencies, Orchestration`. |
| `TaskEditorSurface` | `/tasks/new` and `/tasks/$id/edit`. | One component, branched by `mode` prop. |
| `TaskRunDetailHeader`, `TaskRunIdentityPanel`, `TaskRunProgressPanel`, `TaskRunActivityPanel`, `TasksReviewsCard` | `/tasks/$id/runs/$runId`. | Vertical stack inside `tasks-run-detail-main`. |
| `TasksEmptyState` | `/tasks` empty branch. | Owns the 6-template grid. |
| `TasksKanbanBoard`, `TasksDashboardView`, `TasksInboxView` | `/tasks` mode-pill targets. | Each is a separate body. |

---

## 4. Information architecture observations

- **Mode pills overload the shell.** `LIST / KANBAN / DASHBOARD / INBOX` is four top-level views on the same noun. That is a lot of decisions before the operator sees data. With 0 tasks, the pills still appear; with 18 tasks, switching modes loses local list filters because each mode has its own filter store.
- **The Kanban grouping in Storybook does not exist on the live list rail.** The grouped view is the kanban mode; the list rail is a flat search-filter-driven panel. Storybook conflates the two.
- **The SplitPane list rail is duplicated by the parent on child routes.** See 2.1 and 2.8.
- **`/tasks/$id` tab list (7 tabs) is dense.** Overview, Runs, Events, Agents, Children, Dependencies, Orchestration. With counts and `LIVE` badges on three of them, the cognitive load is high. Consider grouping (Activity = Runs + Events + Agents, Structure = Children + Dependencies, Orchestration alone).
- **`/tasks/$id/runs/$runId` collapses 4 sections vertically without a tab structure.** It is consistent (always Identity, Progress, Activity, Reviews) but the page has no anchor links or sticky headings; long Reviews tables push Activity off-screen.

---

## 5. Findings rolled up across the module

### P0 (ship blockers, repeat across multiple routes)

1. **Run elapsed timer overflows in minutes.** `formatElapsed` in `task-run-detail-header.tsx:16-41` and `task-run-detail-panels.tsx:117-142` only emits `Xm Ys`. Stories produce `28308m 15s` and the run header pill reads `28308M 15S`. Truthful-UI violation. Affects `/tasks/$id/runs/$runId` and the Run detail header. Evidence: `_evidence/tasks-id-runs-runId/sb-running.png`.
2. **Em dashes in shipped copy across `tasks-detail-runs-panel.tsx`, `tasks-detail-overview-panel.tsx`, `tasks-detail-header.tsx`, `task-formatters.ts`, `task-templates.ts`.** Violates `DESIGN.md` Copy section and the audit hard rule. Visible verbatim in `_evidence/tasks/live-empty.png` (`Bind a cron or schedule from Automation — ...`).
3. **`/tasks/$id/edit` infinite spinner on a missing task.** Live probe of `/tasks/task_001/edit` against the empty daemon left "Loading task…" forever (`_evidence/tasks-id-edit/live-missing.png`). Source: `tasks.$id.edit.tsx:14-30`. The `MissingTask` branch only triggers when `isLoading` flips false AND `page.isInitialized` is true; with a 404 the route can stall.
4. **SplitPane parent renders the list rail alongside child outlets on `/tasks/new`, `/tasks/$id/edit`, `/tasks/$id/runs/$runId`.** Source: `tasks.tsx:194-202`. Visual evidence: `_evidence/tasks-new/live.png`, `_evidence/tasks-id-edit/live-missing.png`, `_evidence/tasks-id-runs-runId/live-not-found.png`. Wrecks form focus, doubles affordances, and confuses error handling.

### P1 (high-value polish)

5. **Cancel verb inconsistency.** `Cancel` on the task header (`tasks-detail-header.tsx:153`) vs `Kill run` on the run header (`task-run-detail-header.tsx:135`).
6. **Storybook `default-list` drifts from the live route.** Story renders flat grouped vertical list; live route renders `TasksListPanel` in a `SplitPane`. Compare `_evidence/tasks/sb-default-list.png` and `_evidence/tasks/live-empty-1440.png`. Designers and QA reading the story will think this is the live shape.
7. **Run detail not-found has no rendered error.** Cross-route handling: parent `/tasks/$id` short-circuits before child renders its own not-found message. Evidence: `_evidence/tasks-id-runs-runId/live-not-found.png`.
8. **`taskStatusTone` returns `neutral` while `taskStatusSignal` pulses accent for the same `in_progress` status.** Inconsistent visual register across the row.

### P2 (worthwhile)

9. **Dead `onCopyCli` prop on `TasksEmptyState`.** Component declares `onCopyCli` (`tasks-empty-state.tsx:23, 55-66`) but no caller passes it. CLI-first agents lose the "real commands" affordance promised by `COPY.md`.
10. **Editor breadcrumb uses uppercase mono `BACK TO TASKS`** (`task-editor-surface.tsx:139-162`). The rest of the editor uses sentence-case nav. Inconsistent.
11. **Status casing mixed across components.** Detail header uses `In Progress`, run detail uses `running`, list rows use various. Pick one.
12. **No live indicator on `/tasks/$id/runs/$runId`.** No "connected / reconnecting / stale" banner even though the route is the live-stream leaf.

### P3 (parking lot)

13. Mode pills lose filter state on switch; consider preserving search and lane.
14. Detail tabs (7) could be grouped into Activity / Structure / Orchestration.
15. Run detail has no sticky section nav for long Reviews.

---

## 6. Recommended action plan

Map each P0 to its `/impeccable` command, in priority order.

1. `/impeccable harden web/src/systems/tasks/components/task-run-detail-header.tsx web/src/systems/tasks/components/task-run-detail-panels.tsx` to fix `formatElapsed`. Promote past 60 min into hours and days. Add unit tests for boundaries (59s, 60s, 1h, 24h, 7d). Truthful-UI fix.
2. `/impeccable clarify web/src/systems/tasks` to scrub every em dash from copy strings and tooltips. Replace with commas, periods, or colons. Re-run grep for `—` and `--` as a guard.
3. `/impeccable harden web/src/routes/_app/tasks.$id.edit.tsx` so the missing-task branch fires on 404 / fatal error, not only on `!isLoading && page.task && !page.isInitialized`. Add an explicit error UI mirroring `tasks.$id.tsx:42-54`.
4. `/impeccable layout web/src/routes/_app/tasks.tsx` to stop rendering the SplitPane list rail on the create, edit, and run-detail child routes. Either the parent skips the SplitPane when `hasChildMatch` and the child is a full-bleed route, or each full-bleed child is moved out of the `/tasks` parent into a sibling segment.
5. `/impeccable clarify web/src/systems/tasks/components/task-run-detail-header.tsx` to change `Kill run` to `Cancel run`. Update tests and any e2e selectors.
6. `/impeccable polish web/src/routes/_app/stories/-tasks.stories.tsx` to render `TasksListPanel` populated state, matching the live route, instead of the kanban-style grouped layout.
7. `/impeccable harden web/src/routes/_app/tasks.$id.runs.$runId.tsx` to render its own not-found independent of the parent task's existence (or to push the parent into a tolerant-of-missing-parent mode).
8. `/impeccable typeset web/src/systems/tasks/lib/task-formatters.ts` to align tone and signal so that `in_progress` either pulses accent everywhere or stays neutral everywhere.
9. `/impeccable polish web/src/systems/tasks` for the remaining P2/P3 items (CLI affordance, casing, breadcrumb).

---

## 7. Sign-off checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/`.
- [x] No section is left empty.
- [x] Findings are tagged P0 to P3 with effort and command.
- [x] No hallucinated routes, components, or props (cross-referenced to source / Storybook).
- [x] No em dashes in this report.
- [x] Module-level overview, route-level reports under `01_..05_`.
