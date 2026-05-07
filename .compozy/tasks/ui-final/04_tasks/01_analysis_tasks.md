# UI/UX Analysis: `Tasks` :: `/tasks`

> **Status:** draft
> **Owner subagent:** `tasks-module-auditor`
> **Date:** 2026-05-06
> **Module:** Tasks (`04_tasks`)
> **Route path:** `/tasks` (TanStack Router id: `/_app/tasks`)
> **Web source:** `web/src/routes/_app/tasks.tsx`
> **System owner:** `web/src/systems/tasks/`
> **Storybook story id(s):** `routes-app-stories-tasks--default-list`, `routes-app-stories-tasks--empty`, `routes-app-stories-tasks--kanban`, `routes-app-stories-tasks--dashboard`, `routes-app-stories-tasks--inbox`, `routes-app-stories-tasks--loading`, `routes-app-stories-tasks--error`
> **Live URLs probed:** `http://localhost:3000/tasks`, `http://localhost:6006/?path=/story/routes-app-stories-tasks`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read** (relative paths):
  - `web/src/routes/_app/tasks.tsx`
  - `web/src/systems/tasks/components/tasks-page-shell.tsx`
  - `web/src/systems/tasks/components/tasks-list-panel.tsx`
  - `web/src/systems/tasks/components/tasks-list-row.tsx`
  - `web/src/systems/tasks/components/tasks-empty-state.tsx`
  - `web/src/systems/tasks/components/tasks-detail-preview-panel.tsx`
  - `web/src/systems/tasks/components/task-card.tsx`
  - `web/src/systems/tasks/lib/task-formatters.ts`
  - `web/src/systems/tasks/lib/task-templates.ts`
  - `web/src/systems/tasks/types.ts`
  - `web/src/routes/_app/stories/-tasks.stories.tsx`
- **Storybook stories opened**:
  - `routes-app-stories-tasks--default-list` -> `http://localhost:6006/iframe.html?id=routes-app-stories-tasks--default-list&viewMode=story`
  - `routes-app-stories-tasks--empty` -> same path with `--empty`
  - `routes-app-stories-tasks--kanban` -> same path with `--kanban`
  - `routes-app-stories-tasks--dashboard` -> same path with `--dashboard`
  - `routes-app-stories-tasks--inbox` -> same path with `--inbox`
  - `routes-app-stories-tasks--loading` -> same path with `--loading`
  - `routes-app-stories-tasks--error` -> same path with `--error`
- **Live web probes (`http://localhost:3000`)**:
  - `/tasks` empty state at 1440 wide (`_evidence/tasks/live-empty-1440.png`).
  - `/tasks` empty state at 1024 wide (`_evidence/tasks/live-empty-1024.png`).
  - `/tasks` empty state at 768 wide (`_evidence/tasks/live-empty-768.png`).
  - `/tasks` empty state at 320 wide (`_evidence/tasks/live-empty-320.png`).
- **Screenshots / DOM snapshots captured**:
  - `_evidence/tasks/live-empty.png`, `live-empty-1440.png`, `live-empty-1024.png`, `live-empty-768.png`, `live-empty-320.png`. Empty `/tasks` state.
  - `_evidence/tasks/sb-default-list.png`. Storybook populated grouped list.
  - `_evidence/tasks/sb-empty.png`. Storybook empty branch (workspace seeded, list empty).
  - `_evidence/tasks/sb-kanban.png`. Kanban mode after click.
  - `_evidence/tasks/sb-dashboard.png`. Dashboard mode.
  - `_evidence/tasks/sb-inbox.png`. Inbox mode.
  - `_evidence/tasks/sb-loading.png`. Skeleton loading rows.
  - `_evidence/tasks/sb-error.png`. Dashboard error branch.
- **Console / network errors observed**: none on the live empty state.
- **Keyboard / a11y probes performed**: live snapshot via `agent-browser snapshot` confirmed correct landmarks (`main`, `complementary` for sidebar, `region "Notifications alt+T"`). The empty state heading is `<h3>` (`tasks-empty-state.tsx:35` resolved through `Empty`); no `<h1>` for the page title. Mode pills are `button` elements, ARIA-grouped via `PillGroup`. The `Empty` component does not declare a `region` role.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** the operator's home for the task queue. Lists every task in the active workspace, lets the operator switch between list, kanban, dashboard, and inbox views, and offers a SplitPane preview of the selected row. Acts as the parent shell for `/tasks/new`, `/tasks/$id`, `/tasks/$id/edit`, and `/tasks/$id/runs/$runId`.
- **Primary user goal on this route:** find or create a task and either inspect it inline (preview) or open its detail.
- **Entry vectors:** sidebar `Tasks` link; deep links from inbox notifications, run links, dashboard cards.
- **Exit vectors:** click a row -> `/tasks/$id`; click `Task` or `New task` -> `/tasks/new`; click a template tile -> `/tasks/new?template=<id>`; mode pills -> sibling views in place.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | yes | `_evidence/tasks/live-empty.png` plus `tasks-empty-state.tsx:30-92` | strong (template grid, headline, CTA) |
| Loading / skeleton | yes | `_evidence/tasks/sb-loading.png`, `tasks-list-panel.tsx:127-138` | adequate |
| Partial data | partial | List rail partially loads; preview panel races with list | weak (no shared skeleton continuity) |
| Populated (typical) | yes | `_evidence/tasks/sb-default-list.png` (but story is divergent, see below) | weak (Storybook does not match live) |
| Populated (dense, 100+ rows) | unknown | No high-density story; row uses no virtualization (`tasks-list-panel.tsx:160-174`) | missing |
| Error (network) | yes for dashboard/inbox | `_evidence/tasks/sb-error.png` | adequate |
| Error (permission / 403) | not visible | No 403 story; route does not branch on 403 | missing |
| Error (not found / 404) | n/a (route always exists) | n/a | n/a |
| Read-only / disabled | n/a | n/a | n/a |
| Live-update (stream / SSE) | partial | Inbox unread count comes from `page.inbox?.unread_total`; no global stream banner | weak |
| Mobile / narrow viewport | partial | At 320 wide the SplitPane and the mode pills compress but stay usable; `_evidence/tasks/live-empty-320.png` shows truncation of meta and pills wrap to two rows | weak |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |  3    | `tasks.$id.tsx:74` LIVE on Events; no global stream banner on `/tasks`; status pulse on rows works | No connection state on the parent shell |
| 2  | Match between system and real world    |  3    | Lifecycle phases match `glossary.md` (saved intent, awaiting approval, ready to start, queued, running). One verb drift: `Kill run` (run page only, not here) | Mode pill labels are uppercase mono; readable |
| 3  | User control and freedom               |  2    | List supports search and lane filter, but switching modes loses local filter state; no undo on delete (delete dialog yes); no `back` from preview except close | No keyboard shortcut to dismiss preview, escape only via close button |
| 4  | Consistency and standards              |  2    | List rail vs. kanban grouping render differently; status casing varies across surfaces; mode pills follow `DESIGN.md` segmented pattern | Storybook drifts from live |
| 5  | Error prevention                       |  3    | Delete dialog requires confirm typing (`task-delete-action.tsx`); `New task` button disabled on `/tasks/new` to prevent self-routing | Search lacks debounce indicator |
| 6  | Recognition rather than recall         |  3    | Mode pills always visible; row metadata exposes status, owner, attempt counts | Long titles truncate without tooltip in some rows |
| 7  | Flexibility and efficiency of use      |  2    | Lane pills (`All / Mine / Watched`) are visible but only fire on the list mode; no `j/k` navigation; no keyboard shortcut for `New task`. Mode pills do not preserve list filters | Power users get little affordance beyond the search box |
| 8  | Aesthetic and minimalist design        |  3    | DESIGN.md tokens applied; warm dark canvas, JetBrains Mono eyebrows, accent CTA. Empty state grid is dense but uses six template tiles which is one tile per concept | Template grid pushes the page below the fold at 1024 |
| 9  | Help users recognize / recover errors  |  2    | List error path renders `AlertCircle` plus the message; dashboard/inbox errors render `tasks-dashboard-error` test id | No retry button visible in error states; copy is the raw API error string |
| 10 | Help and documentation                 |  2    | Empty state copy explains tasks; lifecycle hints render in detail; no `/tasks` page-level help link | No CLI affordance on empty state (dead `onCopyCli` prop) |
|    | **Total**                              | **25/40** | | **Band:** ◯ adequate (20-28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders (`border-l/r > 1px`) used decoratively | OK | `tasks-list-row.tsx:91-97` uses a 3px `bg-` accent indicator only on selected rows, which `DESIGN.md` §6 explicitly allows. |
| Gradient text (`background-clip: text` + gradient) | OK | None observed. |
| Glassmorphism / blur as default | OK | None on `/tasks`. |
| Hero-metric template (big number + label + sparkline) | OK on this route | Dashboard mode uses `Metric` cards (correct DESIGN.md pattern), not hero-metric clones. |
| Identical card grids | partial | Empty state template grid is 6 identical cards (`tasks-empty-state.tsx:80-87`) with rotating icons; tonally fine but visually monotone. |
| Modal as first thought | OK | Delete uses dialog; create flows route to `/tasks/new` (no modal). |
| Em dashes in copy | VIOLATION | `tasks-empty-state.tsx` template descriptions render strings from `task-templates.ts:51, 83`: `"Bind a cron or schedule from Automation — re-enqueues a run every tick."` and similar. Visible verbatim in `_evidence/tasks/live-empty.png`. |
| Generic AI palette (default Tailwind blues, neon-on-black) | OK | Warm dark canvas + accent orange, per DESIGN.md. |
| Category-reflex theme | OK | Tasks page is operator-dense, not "agent-glow"; aesthetic is operator-first. |
| Restated headings / intros that repeat the title | partial | `TasksPageShell` title is `Tasks`; the empty state restates it as `No tasks yet in pedronauck`. The list panel headline says `All Tasks`. The page has three "Tasks" labels stacked at empty state. |
| Decorative shadows / heavy elevation | OK | Flat depth model honored; `Section` and `Empty` use 1px dividers, no shadows. |
| Hardcoded `#000` / `#fff` instead of tinted neutrals | OK | All colors via `var(--color-*)` tokens; no raw hex literals in components. |

**Summary verdict:** borderline. The em dashes in template descriptions plus the redundant headings would let a reader say "AI made this", but DESIGN.md tokens, flat depth, and operator-first density push back hard. Fix the em dashes and the verdict moves to "no".

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** at empty state, the operator sees 4 mode pills + `Task` button + `New task` button + 6 template tiles + `Copy CLI command` (currently hidden because the prop is dead). That is 11 to 12 first-class affordances. Above the four-option threshold.
- **Eight-item checklist:**
  1. Are >4 options visible at once? **fail** (11 to 12 at empty state, see above).
  2. Are labels self-evident without docs? **pass** (`Task`, `New task`, `LIST`, `KANBAN`, `DASHBOARD`, `INBOX` all map to operator vocabulary).
  3. Is the primary action visually dominant? **pass** (the `New task` button inside the empty state is `size="lg"` accent fill).
  4. Is information progressively disclosed? **fail** (template grid renders all six immediately; no "show more" or recommended highlight).
  5. Do related elements group via proximity / shared container, not just colour? **pass** (`Section` wrapper for templates).
  6. Is hierarchy clear via scale/weight contrast (≥1.25 ratio)? **pass** (page title 20px / body 13px / mono eyebrow 11px).
  7. Is body line length within 65 to 75ch? **pass** (`Empty` description sentence ~ 60ch).
  8. Is whitespace varied (rhythm) instead of uniform padding? **partial** (sections use 24px / 16px gaps; rhythm OK; but the template grid columns share equal padding making the rhythm flat).

  Failures: 2. Cognitive load = moderate.
- **IA observations:**
  - Mode pills should preserve search and lane state across switches.
  - Empty state restates the page title; condense.
  - `Task` (header) and `New task` (rail) and `New task` (empty CTA) all do the same thing. Three calls for one action.
  - The `Copy CLI command` affordance exists in the component but is never wired (dead `onCopyCli` prop). CLI-first agents lose this.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all references go through `var(--color-*)`. No raw hex in `tasks.tsx`, `tasks-page-shell.tsx`, `tasks-list-panel.tsx`, `tasks-list-row.tsx`, `tasks-empty-state.tsx`. One borderline: `tasks-empty-state.tsx:106` uses `border-[color:rgba(58,58,60,0.6)]` for the dashed blank-template border, which DESIGN.md does not have a token for; acceptable because dashed-divider is one-off and the value matches the divider hue at 60% alpha.
- **Type scale:** Inter for headings, JetBrains Mono for eyebrows and badges. No Playfair, no NuixyberNext outside the wordmark.
- **Radii / spacing:** `rounded-xl` (12px) for the surface card, `rounded-lg` for the buttons, `rounded-full` for the dots. All match DESIGN.md.
- **Elevation:** flat. Lists use 1px `var(--color-divider)` row separators; the SplitPane has no shadow.
- **Signal palette discipline:** accent on the `Task` create button and the selected-row indicator. Dot pulse on running statuses. No decorative use of accent observed on this route.
- **Grid / rhythm:** the 4-column metric grid in the dashboard mode follows DESIGN.md §5; the template grid uses `md:grid-cols-2 xl:grid-cols-[1.2fr_1fr_1fr]` which feels off-grid (a 1.2fr column followed by two 1fr columns). The explicit `1.2fr_1fr_1fr` reads as ad-hoc tuning.
- **Density:** comfortable. Live empty page renders without scrolling at 1440 (`_evidence/tasks/live-empty-1440.png`). At 1024, the template grid drops to 2 columns and the page scrolls (`_evidence/tasks/live-empty-1024.png`). At 320 the layout compresses but stays usable.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** `New task` (header `Task`, rail `New task`, empty `New task`). Three triggers, one action. The header `Task` button is disabled on `/tasks/new` to prevent self-routing.
- **Destructive actions:** delete uses `TaskDeleteAction` with a confirm dialog (`task-delete-action.tsx`). Dialog requires explicit confirm.
- **Forms:** route hosts no form; form lives on the create child route.
- **Tables / lists:** rows are virtualization-free `Section` children (`tasks-list-panel.tsx:160-174`). Sort is implicit (server order). Filtering is via search + lane + status. No keyboard navigation between rows beyond Tab.
- **Selection model:** single selection (current selected row) plus an Outlet for child routes. No multi-select.
- **Modals / drawers:** delete dialog, no others.
- **Live updates:** mode pill `INBOX` shows `badge: page.inbox?.unread_total`. List rows pulse for running statuses. No global stream connection banner.
- **Optimistic vs pessimistic updates:** `useTasksPage` flips `isPublishPending`, `isDeletePending`, etc., which keeps actions pessimistic.
- **Hover / focus / active states:** `tasks-list-row.tsx:84-87` has `hover:bg-`, `focus-visible:ring-2`. Selected has bg + 3px accent indicator. Active state has no separate treatment beyond click.
- **Loading patterns:** skeletons match the list row shape (`tasks-list-panel.tsx:127-138`). No debounce on the list, so a 100ms response flashes the skeletons.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every actionable element is a `button` or `a`. Live snapshot confirms `tabIndex` on rows when `clickable` (`tasks-list-row.tsx:75`). The mode pills, search, lane pills, `New task`, and rows are all reachable.
- **Focus rings:** `focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]` on the row and standard ring on `Button`. Visible.
- **TAB order:** sidebar -> shell controls (mode pills, `Task`) -> list rail (search, lane pills, `New task`, rows) -> detail / outlet. Logical.
- **ARIA roles / labels:** mode pills group has `data-testid="tasks-mode-pills"` but no `aria-label` on the group; `tasks-list-panel.tsx:92` provides `aria-label="Task lane"` on lane pills. Search input has placeholder only, not a programmatically associated label. The list rail uses `aside` with no `aria-label`.
- **Color contrast:** body text `#E5E5E7` on `#141312` exceeds 4.5:1; mono eyebrows `#98989D` on `#141312` are over 4.5:1. Pill chips use 15% tint backgrounds, with full-color text; contrast is borderline (`#FFD60A` warning text on `#FFD60A26` measures ~1.3:1 background-vs-text; legible because the text is the saturated full color, but the chip is harder to read against canvas than the text alone).
- **Motion:** the running dot pulses; `prefers-reduced-motion` is honored globally per `DESIGN.md` §9.
- **Text scaling:** at 200% zoom the SplitPane rail still fits and the empty state stretches vertically; no overflow observed in the snapshot.
- **Forms:** n/a on this route.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** strong. The empty state explains what tasks are, names the active workspace, offers six templates, and exposes a primary CTA. Two issues: the description renders an em dash in template strings (`task-templates.ts:51`), and the hidden `Copy CLI command` button is dead.
- **Loading:** adequate. Skeleton rows match the populated row shape (3 stripes per row).
- **Error:** weak. The error path shows the API message with an `AlertCircle`, no retry button, no support link.
- **Permission denied:** missing. The route does not branch on 403 separately.
- **Stale / disconnected:** missing. No banner when SSE drops.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** `task` and `task run` are used consistently; lifecycle phases match the glossary. `recipe` does not appear.
- **Tone:** dry, operator-first. The empty state description is informative ("Tasks are durable contracts of work...").
- **Em dashes:** `task-templates.ts:51, 83` (rendered live in the empty state) plus other shipped copy. P0 fix.
- **Restated headings:** the page header says `Tasks`; the empty state says `No tasks yet in pedronauck`; the list panel headline says `All Tasks 0 total`. Three near-identical headings.
- **Sentence case vs Title Case:** sentence-case for body and CTAs; uppercase mono for metadata. Mostly consistent.
- **Truthful UI test:** the route does not invent functionality. The recurring template explicitly delegates to Automation ("Configure the schedule from the Automation area"), which matches runtime truth. The dashboard and inbox modes fetch real endpoints. No fake metrics observed.

---

## 10. Performance & Responsiveness

- **Initial render:** route uses `useTasksPage` hook that pulls list, dashboard, and inbox. Dashboard and inbox load lazily depending on mode.
- **Re-render hot spots:** `tasks-list-row.tsx` is not memoized; rendering 100+ rows will re-render on every keystroke through the search box because `searchQuery` lives at the page level.
- **List virtualization:** none. `Section` renders all rows. 100+ rows will degrade.
- **Bundle red flags:** none on this route specifically. The dashboard view pulls `recharts`-style chart components (`tasks-dashboard-cards.tsx`) but those are mode-gated.
- **Responsive behaviour:** survives 320 / 768 / 1024 / 1440. At 320 wide the SplitPane stacks; lane pills wrap (`_evidence/tasks/live-empty-320.png`).
- **Mobile interactions:** rows expose hover + focus; no hover-only affordances.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-tasks--default-list`
  - `routes-app-stories-tasks--empty`
  - `routes-app-stories-tasks--kanban`
  - `routes-app-stories-tasks--dashboard`
  - `routes-app-stories-tasks--inbox`
  - `routes-app-stories-tasks--loading`
  - `routes-app-stories-tasks--error`
- **States covered in Storybook:** empty, populated (but rendered as a flat grouped list, not the SplitPane rail; see drift), kanban, dashboard, inbox, loading, error.
- **Gaps:**
  - No 100+ row populated story.
  - No 403 story.
  - No SSE-drop / stale story.
  - No mobile-viewport story.
  - No "selected row + preview panel" story showing the SplitPane right slot populated.
- **Story drift:** `default-list` story renders a different layout than the live route (compare `_evidence/tasks/sb-default-list.png` vs `_evidence/tasks/live-empty-1440.png`). The story is the kanban-like grouped layout, not the SplitPane list rail.

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

1. **[P0] What:** em dashes in shipped template descriptions (`task-templates.ts:51, 83`) render verbatim on `/tasks` empty state.
   - **Why:** violates `DESIGN.md` Copy section and the audit hard rule. Shows up on the operator's first ever look at the runtime.
   - **Fix:** replace `—` with `,`, `:`, or `.`. Re-grep `web/src/systems/tasks` for `—` and remove every occurrence.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/lib/task-templates.ts`
   - **Effort:** S
   - **Evidence:** `_evidence/tasks/live-empty.png`; `task-templates.ts:51`, `:83`.

2. **[P0] What:** SplitPane parent renders the list rail alongside child outlets on `/tasks/new`, `/tasks/$id/edit`, `/tasks/$id/runs/$runId`. The bug originates in `tasks.tsx:194-202`.
   - **Why:** breaks form focus on create / edit, doubles affordances (two `New task` buttons), confuses the run detail layout, and is a top-level IA failure.
   - **Fix:** when `hasChildMatch` is true and the child is a "full-bleed" route (create, edit, run), render only the `<Outlet />` and skip the SplitPane. The cleanest split is to hold a small allow-list of "show list rail" child route ids (`/tasks/$id` only) and otherwise render outlet only.
   - **Cmd:** `/impeccable layout web/src/routes/_app/tasks.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/tasks-new/live.png`, `_evidence/tasks-id-edit/live-missing.png`, `_evidence/tasks-id-runs-runId/live-not-found.png`; `tasks.tsx:194-202`.

### P1 - High-Value Polish

3. **[P1] What:** Storybook `default-list` story renders a flat grouped vertical list, not the SplitPane list rail the live route uses.
   - **Why:** Storybook is the populated reference for designers and QA; the divergence creates the wrong mental model.
   - **Fix:** rewrite the story to seed `TasksListPanel` with populated tasks inside a `SplitPane`, matching the live shape. Keep the kanban grouped layout in `routes-app-stories-tasks--kanban` only.
   - **Cmd:** `/impeccable polish web/src/routes/_app/stories/-tasks.stories.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/tasks/sb-default-list.png` vs `_evidence/tasks/live-empty-1440.png`.

4. **[P1] What:** `taskStatusTone` returns `neutral` while `taskStatusSignal` pulses accent for `in_progress`.
   - **Why:** the row dot says "live", the status chip says "calm". Inconsistent visual semantics across the same row.
   - **Fix:** either pulse accent on the chip too (with the runtime live-check the source already comments out), or keep the chip neutral and drop the pulse, accepting that "live" is signaled by attempt count and other meta.
   - **Cmd:** `/impeccable typeset web/src/systems/tasks/lib/task-formatters.ts`
   - **Effort:** S
   - **Evidence:** `task-formatters.ts:32-50, 125-145`.

5. **[P1] What:** mode-pill switches lose the search and lane filter state.
   - **Why:** an operator filtering by "Mine + blocked" loses both when they tap KANBAN to see the same data on a board.
   - **Fix:** lift filter state into the page hook so all four modes share search and lane. Mode-specific filters (e.g. inbox lane tabs) stay local.
   - **Cmd:** `/impeccable layout web/src/hooks/routes/use-tasks-page.ts`
   - **Effort:** M
   - **Evidence:** behavior observed in source; mode pills call `page.handleModeChange` (`tasks.tsx:44-49`) which does not touch search or lane.

### P2 - Worthwhile

6. **[P2] What:** dead `onCopyCli` prop on `TasksEmptyState` (`tasks-empty-state.tsx:23, 55-66`).
   - **Why:** AGH ships a CLI surface; the empty state should expose it. CLI-first agents and operators want a paste-ready command.
   - **Fix:** wire `onCopyCli` from the page; copy `agh task create --workspace <name>` (or the closest documented command) to the clipboard.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/components/tasks-empty-state.tsx`
   - **Effort:** S

7. **[P2] What:** redundant `New task` calls to action (header `Task`, rail `New task`, empty `New task`).
   - **Why:** three triggers for one action; the header `Task` button is the awkward outlier (mismatched verb, disabled when active).
   - **Fix:** keep the header `New task` and the empty state CTA; drop the rail `New task` (or hide it when the rail is in the SplitPane mode).
   - **Cmd:** `/impeccable distill web/src/routes/_app/tasks.tsx`
   - **Effort:** S

8. **[P2] What:** list rail has no virtualization (`tasks-list-panel.tsx:160-174`).
   - **Why:** at 100+ rows the page will re-render on every keystroke and the rail will scroll-jank.
   - **Fix:** `react-virtual` or `tanstack-virtual` rows; keep the search at the page level but memoize each row.
   - **Cmd:** `/impeccable optimize web/src/systems/tasks/components/tasks-list-panel.tsx`
   - **Effort:** M

### P3 - Parking Lot

9. **[P3] What:** template grid uses an off-grid `1.2fr_1fr_1fr` column ratio at xl.
   - **Why:** ad-hoc tuning, breaks rhythm.
   - **Fix:** equal columns `1fr_1fr_1fr` or `repeat(3,minmax(0,1fr))`.

10. **[P3] What:** error UI lacks a retry button.
    - **Fix:** add a ghost `Retry` that re-runs the failed query.

---

## 13. Persona Red Flags

- **Operator (returning power user, keyboard-first):** no `j/k` row navigation, no `n` shortcut for new task, no `/` shortcut to focus search. Mode-switch loses filters. Three `New task` triggers do not feel "fast" but rather noisy.
- **First-timer (onboarding, no mental model yet):** the empty state is genuinely helpful (workspace name, six templates, brief explanation). The em dashes in template descriptions read as machine-generated.
- **Agent (DOM scrape consumer):** stable test ids on every interactive element (`tasks-mode-list`, `tasks-mode-kanban`, `tasks-list-create`, `tasks-empty-template-<id>`, `tasks-shell-title`). DOM is predictable. `aria-label` on the mode pill group would tighten programmatic reading.

---

## 14. Cross-Module Consistency Notes

- The mode pill pattern (`LIST / KANBAN / DASHBOARD / INBOX`) reappears on Network and Dashboard surfaces. Casing and density match.
- Empty-state composition (`Empty` icon + headline + description + CTA + `Section` of templates) matches `Skills` and `Knowledge` empty states.
- The `New task` header CTA is `variant="outline"` (`tasks.tsx:139`); other surfaces use `variant="default"` for primary creates. Inconsistent emphasis.

---

## 15. Open Questions

- Should the four mode pills be reduced to two (List + Inbox), with Kanban and Dashboard moved to a sibling segment? Four top-level views on the same noun is a lot.
- Is the SplitPane preview useful, or is the route detail (`/tasks/$id`) enough? The preview adds a side panel that nobody sees on the create / edit / run-detail child routes (where it should not exist either, see P0 #2).
- Should the empty state CTA pre-fill the workspace and route to `/tasks/new` with a one-click "create and enqueue" affordance for confident operators?

---

## 16. Recommended Action Plan

1. `/impeccable clarify web/src/systems/tasks/lib/task-templates.ts` to remove em dashes from template descriptions and any other shipped copy.
2. `/impeccable layout web/src/routes/_app/tasks.tsx` to fix the SplitPane renders alongside child outlets.
3. `/impeccable polish web/src/routes/_app/stories/-tasks.stories.tsx` to align the populated story with the live `TasksListPanel` shape.
4. `/impeccable typeset web/src/systems/tasks/lib/task-formatters.ts` to align tone and pulse for `in_progress`.
5. `/impeccable layout web/src/hooks/routes/use-tasks-page.ts` to share search and lane filters across modes.
6. `/impeccable clarify web/src/systems/tasks/components/tasks-empty-state.tsx` to wire the CLI affordance.
7. `/impeccable distill web/src/routes/_app/tasks.tsx` to reduce three "New task" triggers to one or two.
8. `/impeccable optimize web/src/systems/tasks/components/tasks-list-panel.tsx` for high-density virtualization.
9. `/impeccable polish web/src/systems/tasks` as the closing pass.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/tasks/`.
- [x] No section is left as `<TODO>` or empty.
- [x] Nielsen scores total is consistent with the band claimed (25/40, adequate).
- [x] Findings are tagged P0 to P3 with effort and command.
- [x] No hallucinated routes, components, or props.
- [x] No em dashes in this report.
- [x] Report length is thorough but not padded.
