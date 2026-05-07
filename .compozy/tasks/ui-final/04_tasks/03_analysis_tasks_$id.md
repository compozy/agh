# UI/UX Analysis: `Tasks` :: `/tasks/$id`

> **Status:** draft
> **Owner subagent:** `tasks-module-auditor`
> **Date:** 2026-05-06
> **Module:** Tasks (`04_tasks`)
> **Route path:** `/tasks/$id` (TanStack Router id: `/_app/tasks/$id`)
> **Web source:** `web/src/routes/_app/tasks.$id.tsx`
> **System owner:** `web/src/systems/tasks/`
> **Storybook story id(s):** `routes-app-stories-tasks-id--overview`, `routes-app-stories-tasks-id--runs-tab`, `routes-app-stories-tasks-id--timeline-tab`, `routes-app-stories-tasks-id--agents-tab`, `routes-app-stories-tasks-id--children-tab`, `routes-app-stories-tasks-id--dependencies-tab`, `routes-app-stories-tasks-id--loading`, `routes-app-stories-tasks-id--not-found`
> **Live URLs probed:** `http://localhost:3000/tasks/task_001` (daemon empty, not-found branch), `http://localhost:6006/?path=/story/routes-app-stories-tasks-id`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read**:
  - `web/src/routes/_app/tasks.$id.tsx`
  - `web/src/systems/tasks/components/tasks-detail-header.tsx`
  - `web/src/systems/tasks/components/tasks-detail-tabs.tsx`
  - `web/src/systems/tasks/components/tasks-detail-overview-panel.tsx`
  - `web/src/systems/tasks/components/tasks-detail-runs-panel.tsx`
  - `web/src/systems/tasks/components/tasks-timeline-panel.tsx`
  - `web/src/systems/tasks/components/tasks-multi-agent-panel.tsx`
  - `web/src/systems/tasks/components/tasks-detail-children-panel.tsx`
  - `web/src/systems/tasks/components/tasks-detail-dependencies-panel.tsx`
  - `web/src/systems/tasks/components/tasks-detail-orchestration-panel.tsx`
  - `web/src/systems/tasks/lib/task-formatters.ts`
  - `web/src/hooks/routes/use-task-detail-route.ts`
  - `web/src/routes/_app/stories/-tasks.$id.stories.tsx`
- **Storybook stories opened** (eight ids in the `routes-app-stories-tasks-id` namespace).
- **Live web probes (`http://localhost:3000`)**:
  - `/tasks/task_001` against an empty daemon, captured at 1440 wide (not-found branch).
- **Screenshots / DOM snapshots captured**:
  - `_evidence/tasks-id/live-not-found.png`. Live not-found branch.
  - `_evidence/tasks-id/sb-overview.png`. Overview tab.
  - `_evidence/tasks-id/sb-runs-tab.png`. Runs tab.
  - `_evidence/tasks-id/sb-timeline-tab.png`. Events tab.
  - `_evidence/tasks-id/sb-agents-tab.png`. Agents tab.
  - `_evidence/tasks-id/sb-children-tab.png`. Children tab.
  - `_evidence/tasks-id/sb-dependencies-tab.png`. Dependencies tab.
  - `_evidence/tasks-id/sb-loading.png`. Loading branch.
  - `_evidence/tasks-id/sb-not-found.png`. Storybook not-found branch.
- **Console / network errors observed**: none on the live probe.
- **Keyboard / a11y probes performed**: live snapshot confirms `<header>` for the detail header, `nav aria-label="Breadcrumb"` for the breadcrumb, `tablist` for the panel tabs (`tasks-detail-tabs.tsx`). `Pill.Dot` is `aria-hidden`.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** the inspectable surface for a single task. Header shows the title, lifecycle phase, status, priority, owner, origin, created-by, last-update, and channel binding (when present). Tabs cycle through Overview, Runs, Events, Agents, Children, Dependencies, and Orchestration. Action buttons in the header offer Edit, Delete, Cancel, Publish, and Start run (visible based on status).
- **Primary user goal on this route:** decide what to do with the task next: publish, start, cancel, edit, or open a specific run.
- **Entry vectors:** `/tasks` row click; preview panel `Open task` link; deep links from inbox and notifications.
- **Exit vectors:** Edit -> `/tasks/$id/edit`; Open run link inside Active Run / Runs panel -> `/tasks/$id/runs/$runId`; child task click -> `/tasks/$childId`; breadcrumb -> `/tasks`.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | partial (overview shows description-empty p) | `tasks-detail-overview-panel.tsx:131-141` | weak (no overall empty for a brand-new task) |
| Loading / skeleton | yes (Loader2) | `_evidence/tasks-id/sb-loading.png`, `tasks.$id.tsx:35-39` | adequate (spinner only, not skeleton) |
| Partial data | yes (panels render whatever they have) | source | adequate |
| Populated (typical) | yes | `_evidence/tasks-id/sb-overview.png` | strong |
| Populated (dense) | partial | `_evidence/tasks-id/sb-children-tab.png`, `sb-dependencies-tab.png` | adequate |
| Error (network) | yes | `tasks.$id.tsx:42-54` shows `Task ${id} not found.` or `fatalError.message` | adequate |
| Error (permission / 403) | not visible | route does not branch on 403 separately | missing |
| Error (not found / 404) | yes | live probe `_evidence/tasks-id/live-not-found.png` shows `Task not found: task_001` | adequate (but jammed against the parent SplitPane rail; see P0 #2) |
| Read-only / disabled | yes (action buttons hide based on `canCancel`, `isDraft`, etc.) | `tasks-detail-header.tsx:62-73` | strong |
| Live-update (stream / SSE) | partial | Events tab shows `LIVE` badge; agents tab shows `LIVE` badge | adequate |
| Mobile / narrow viewport | not stored | only 1440 captured | unknown |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |  3    | Lifecycle pill, status pill, channel pill, LIVE badges on Events / Agents tabs (`tasks-detail-tabs.tsx`) | Lifecycle pill description is rich and helpful (`tasks-detail-header.tsx:223-228`) |
| 2  | Match between system and real world    |  3    | Lifecycle phases match `glossary.md`; Owner / Origin / Created-by metadata match runtime fields | Origin renders raw uppercase (`WEB`, `CLI`); fine for operators |
| 3  | User control and freedom               |  3    | Edit, Cancel, Publish, Start run, Delete in header. Breadcrumb back to `/tasks`. Delete dialog requires confirm | No undo on cancel; no "go back to last tab" memory |
| 4  | Consistency and standards              |  2    | Status casing inconsistent: header chip `In Progress`, runs panel chip `running` raw lowercase | Run-row chip drops the label map |
| 5  | Error prevention                       |  3    | Delete dialog with confirm typing; cancel button hidden when `canCancel` is false | No prevent-double-submit on Publish |
| 6  | Recognition rather than recall         |  3    | Tabs surface counts (`Runs 2`, `Children 3`, `Dependencies 1`) and `LIVE` flags | 7 tabs is many to recognize at a glance |
| 7  | Flexibility and efficiency of use      |  2    | Tabs are clickable, no keyboard shortcuts. Open run link is `font-mono` accent | No `o` to open active run, no `r` to retry |
| 8  | Aesthetic and minimalist design        |  3    | DESIGN.md tokens applied; lifecycle hint text under the meta row gives context | Header carries 5 to 7 chips (lifecycle, status, channel, priority, approval); hard to scan |
| 9  | Help users recognize / recover errors  |  2    | Runs panel error state via `Empty + AlertCircle`; not-found state shows raw message | No retry button on the panel errors |
| 10 | Help and documentation                 |  3    | Lifecycle hint paragraph (`tasks-detail-header.tsx:223-228`) is genuinely useful | No "what is a task" link; no docs anchor |
|    | **Total**                              | **27/40** | | **Band:** ◯ adequate (20-28), top of the band |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders (`border-l/r > 1px`) used decoratively | OK | None observed; selected list rows in the parent rail use a 3px accent indicator which is documented in `DESIGN.md` §4. |
| Gradient text | OK | None. |
| Glassmorphism / blur as default | OK | None. |
| Hero-metric template | OK | Overview shows three Metric cards (children / dependencies / runs); not the SaaS cliché. |
| Identical card grids | OK | Tabs differentiate panel content. |
| Modal as first thought | OK | Delete dialog only. |
| Em dashes in copy | VIOLATION | `tasks-detail-runs-panel.tsx:67` "Saved intent only — no runs yet". `tasks-detail-runs-panel.tsx:108` channel tooltip uses `—`. `tasks-detail-overview-panel.tsx:95` channel tooltip uses `—`. `tasks-detail-header.tsx:111` channel tooltip uses `—`. `task-formatters.ts:435-437` lifecycle descriptions use `—`. The not-found message `—` separator at `task-formatters.ts:248` renders in run-row Ended cell when no end timestamp. |
| Generic AI palette | OK | Warm dark canvas, accent on primary CTA. |
| Category-reflex theme | OK | Operator-first; no AI-glow. |
| Restated headings | partial | Header title, breadcrumb crumb, lifecycle pill, status pill, plus a lifecycle hint line under the meta row. Lots of re-affirmation of the same state. |
| Decorative shadows | OK | Flat. |
| Hardcoded `#000` / `#fff` | OK | Tokenized. |

**Summary verdict:** borderline. Em dashes in five files plus restated heads bring the route close to "AI made this". The lifecycle pill plus hint paragraph is genuinely thoughtful and pulls it back.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** at the top of the page, the user sees: title, status dot, mono id pill, status pill, lifecycle pill, optional channel pill, plus 4 to 5 action buttons. Below: lifecycle phase description sentence. Then 7 tabs each carrying a count or LIVE flag. Counted as decision points: ~5 chips, 5 buttons, 7 tabs = 17.
- **Eight-item checklist:**
  1. >4 visible options? **fail** (header has 4 to 5 actions; tabs are 7).
  2. Self-evident labels? **pass** (Edit, Cancel, Publish, Start run, Delete).
  3. Primary action visually dominant? **fail** (`Start run` and `Publish` are accent fill but `Edit` is the same outline style as `Cancel`; the eye does not know which to do first).
  4. Progressive disclosure? **fail** (all 7 tabs always visible; even when there are 0 children, 0 dependencies, the tabs render with `0`).
  5. Related elements grouped via proximity? **pass** (header sections; meta row separated; tabs row separated).
  6. Hierarchy ratio ≥1.25? **pass** (title 15px medium, meta 13px, eyebrow 11px).
  7. Body line length 65-75ch? **pass** (description max-width prose).
  8. Whitespace varied? **partial** (panels share padding; the lifecycle hint line is tucked under the meta row tightly).

  Failures: 3. Cognitive load = moderate-high.
- **IA observations:**
  - The 7 tabs could group into Activity (Runs + Events + Agents) and Structure (Children + Dependencies) and Orchestration. That would drop the visible top-level decisions from 7 to 3.
  - Active Run section duplicates information visible in the header: status, channel, attempt counts. Consider collapsing on Overview.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** every color is `var(--color-*)`. No raw hex.
- **Type scale:** Inter 15px medium for the title, JetBrains Mono 11px uppercase tracking 0.14em for breadcrumb and meta. Mono 10px on the lifecycle hint, lighter weight.
- **Radii / spacing:** `rounded-xl` / `rounded-md` / `rounded-full` used per DESIGN.md.
- **Elevation:** flat. Active Run card uses `surface-elevated` background to distinguish.
- **Signal palette discipline:** accent on the primary action; channel pill is violet (`info`); approval pending pill is accent. All match.
- **Grid / rhythm:** one-column body; tabs row width 100%.
- **Density:** tight. Header packs 5 chips + 5 buttons in one row; on 1024 wide the chip row wraps.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** `Edit`, `Cancel`, `Publish`, `Start run`, `Delete` (header). `Open run` link inside Active Run section.
- **Destructive actions:** Delete uses `TaskDeleteAction` with confirm dialog. Cancel uses a single click (no confirm dialog); given the runtime semantics of cancellation, this is acceptable.
- **Forms:** none on this route.
- **Tables / lists:** Runs tab uses `Table` with `Run`, `Attempt`, `Queued`, `Ended`, plus a chevron right link. No sort, no filter.
- **Selection model:** none beyond active tab.
- **Modals / drawers:** delete confirm dialog.
- **Live updates:** Events tab updates via SSE; the tab shows `LIVE` badge. Agents tab shows `LIVE` count.
- **Optimistic vs pessimistic:** pessimistic. Buttons disable while pending.
- **Hover / focus / active states:** chips have hover; buttons have full hover/focus/active. Tabs have `aria-selected` and `data-state`.
- **Loading patterns:** Loader2 for full-page; per-panel spinners on Runs (`tasks-detail-runs-panel.tsx:39-47`).

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every action is a `button` or a `Link`. Tabs are `tab` elements inside a `tablist` with arrow-key handling assumed via the `@agh/ui` Pills primitive.
- **Focus rings:** standard accent ring on buttons; tabs use the segmented pill pattern.
- **TAB order:** breadcrumb -> page header (icon, title, chips, action buttons) -> meta row -> lifecycle hint -> tabs -> active panel content. Logical.
- **ARIA roles / labels:** breadcrumb is `nav aria-label="Breadcrumb"`. Tab items are `tab` with `aria-selected`. The `Channel` pill has a `title` tooltip with em dash content (which screen readers will read).
- **Color contrast:** body text ratio meets AA. Status pill backgrounds at 15% tint vs full-color text are legible.
- **Motion:** Loader2 spin, Pulse on running dot. Reduced-motion respected.
- **Text scaling:** at 200% zoom the chips wrap; no overflow visible in the snapshot.
- **Forms:** n/a.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** the description panel shows `No description provided.` when description is empty (`tasks-detail-overview-panel.tsx:138-140`). Genuinely useful.
- **Loading:** adequate. Loader2 spinner only; no skeleton matching the header shape.
- **Error:** weak. Not-found message is the raw API string `Task not found: <id>`. No retry, no support link, no breadcrumb back affordance. Live probe `_evidence/tasks-id/live-not-found.png` shows the raw text in the detail slot, jammed against the parent SplitPane rail.
- **Permission denied:** missing. No 403 branch.
- **Stale / disconnected:** missing on this route; the Events tab carries the LIVE badge but no degraded indicator.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** task, task run, lifecycle, coordinator handoff, coordination channel are all used per `glossary.md`. The lifecycle phase description text is operator-first and dry.
- **Tone:** the lifecycle hint paragraph reads `"A worker session is executing the active run. Channel messages support coordination only."` (no em dash here). Good.
- **Em dashes:** the channel tooltip across header / overview / runs uses `—`. The runs panel empty title uses `—`. Lifecycle descriptions in `task-formatters.ts:435-437` use `—`. P0 fix.
- **Restated headings:** the `Active Run` section heading + status pill + channel pill + attempt sentence + queued/started timestamps repeat lifecycle content. Consider compressing.
- **Sentence case vs Title Case:** title is sentence case; eyebrows uppercase mono. Consistent.
- **Truthful UI test:** the route does not invent capability. The only truthful-UI risk on this page is the runs panel `Open run` link, which routes correctly to `/tasks/$id/runs/$runId`. No fake schedule / trigger surfaces here.

---

## 10. Performance & Responsiveness

- **Initial render:** route uses `useTaskDetailRoute` which fetches detail, runs, timeline, agents, dependencies, profile, reviews, subscriptions in one orchestrated hook. Heavy initial fetch.
- **Re-render hot spots:** detail and timeline panels re-render on stream events; the orchestration panel may rerender on every subscription change.
- **List virtualization:** Runs panel uses `Table` without virtualization. Children and Dependencies likewise. With 100+ runs, the page will be heavy.
- **Bundle red flags:** Multi-Agent panel imports `recharts`-style timeline visualization (`tasks-multi-agent-panel.tsx`). Bundled even on tasks with 0 descendants.
- **Responsive behaviour:** chips wrap at 1024 wide. At 768 the action buttons may overflow into a second row.
- **Mobile interactions:** chips have hover-only tooltips for the channel binding; on mobile the tooltip is unreachable.

---

## 11. Storybook Coverage

- **Stories present:** Overview, RunsTab, TimelineTab, AgentsTab, ChildrenTab, DependenciesTab, Loading, NotFound. (No Orchestration tab story.)
- **States covered:** all tabs, loading, not-found.
- **Gaps:**
  - Orchestration tab has no story.
  - No `draft` task story (the `Publish` flow).
  - No `failed` task story (the `Retry` action surface).
  - No "no active run" story.
  - No mobile-viewport story.
- **Story drift:** the populated stories render under the parent SplitPane shell; the right slot panel content reflects the live source faithfully. No drift in the detail surface itself.

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

1. **[P0] What:** SplitPane parent renders the list rail alongside this route on full-screen tabs. While `/tasks/$id` arguably benefits from the rail (context), it is the same bug surface as the other child routes. The fix is shared but the impact here is muted.
   - **Why:** consistent root cause; if the fix carves out only `/tasks/$id` to keep the rail, that is fine. Calling it out for completeness.
   - **Fix:** see module overview P0 #4.
   - **Cmd:** `/impeccable layout web/src/routes/_app/tasks.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/tasks-id/live-not-found.png`.

2. **[P0] What:** em dashes in shipped copy across `tasks-detail-runs-panel.tsx:67`, `tasks-detail-runs-panel.tsx:108`, `tasks-detail-overview-panel.tsx:95`, `tasks-detail-header.tsx:111`, and `task-formatters.ts:435-437`.
   - **Why:** ban from `DESIGN.md`, audit hard rule. Tooltips that reach screen readers ship em dashes.
   - **Fix:** rewrite without `—`. Replace with `.`, `:`, or `,`.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks`
   - **Effort:** S
   - **Evidence:** the four files above.

### P1 - High-Value Polish

3. **[P1] What:** status casing varies between detail header (`In Progress`) and runs row (`running`).
   - **Why:** same status, three renders in the same page. Operators read inconsistent tone.
   - **Fix:** route every status pill through `taskStatusLabel` and render uppercase mono per DESIGN.md badge spec.
   - **Cmd:** `/impeccable typeset web/src/systems/tasks`
   - **Effort:** S
   - **Evidence:** `tasks-detail-header.tsx:99` vs `tasks-detail-runs-panel.tsx:103`.

4. **[P1] What:** primary action ambiguity in the header (Edit and Cancel and Publish and Start run all share visual weight).
   - **Why:** the eye does not know which is the next step. `Publish` and `Start run` are correctly accent fill, but `Edit` is the first child in the row, leading the eye.
   - **Fix:** reorder so the situational primary action (Publish or Start run) is leftmost; demote Edit to a quieter ghost button at the right edge.
   - **Cmd:** `/impeccable layout web/src/systems/tasks/components/tasks-detail-header.tsx`
   - **Effort:** S

5. **[P1] What:** 7 tabs without grouping.
   - **Why:** decision overhead at the top of the panel.
   - **Fix:** group into Activity (Runs / Events / Agents), Structure (Children / Dependencies), and Orchestration. Use a two-level tab pattern or a dropdown for the secondary tier.
   - **Cmd:** `/impeccable distill web/src/systems/tasks/components/tasks-detail-tabs.tsx`
   - **Effort:** M

6. **[P1] What:** runs panel uses `—` as the empty Ended cell value (`task-formatters.ts:248`).
   - **Why:** breaks the no-em-dash rule even in a one-character separator.
   - **Fix:** use `(none)` or render `—` literal as `none` mono small text. The simplest replacement is the ASCII `-` or just leave the cell empty.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/lib/task-formatters.ts`
   - **Effort:** S

### P2 - Worthwhile

7. **[P2] What:** loading uses spinner only.
   - **Fix:** render header skeleton (mono id placeholder, status pill placeholder, action button placeholders) so the layout does not jump when data arrives.
   - **Cmd:** `/impeccable harden web/src/routes/_app/tasks.$id.tsx`
   - **Effort:** S

8. **[P2] What:** active run section duplicates information already present in the header (status, channel, attempts).
   - **Fix:** keep only run-specific deltas (queued / started timestamps, attempt-of-N counter) and a clean `Open run` CTA.
   - **Cmd:** `/impeccable distill web/src/systems/tasks/components/tasks-detail-overview-panel.tsx`
   - **Effort:** S

9. **[P2] What:** no `LIVE` indicator on Reviews / Orchestration tabs.
   - **Fix:** if the Reviews query subscribes to a stream, mark it.

### P3 - Parking Lot

10. **[P3] What:** breadcrumb separator is `›` (`tasks-detail-header.tsx:196`).
    - **Fix:** use a Lucide `ChevronRight` icon for visual consistency with the run-detail breadcrumb (which uses the icon at `task-run-detail-header.tsx:83`).

11. **[P3] What:** delete button size and outline style match Edit; the visual weight is identical.
    - **Fix:** danger ghost variant for Delete to match `DESIGN.md` danger discipline.

---

## 13. Persona Red Flags

- **Operator (returning power user):** no keyboard shortcut to jump tabs (`g r` for runs, `g e` for events). Re-fetching detail on tab switch is reasonable but a list-tab shortcut would reduce mouse miles.
- **First-timer:** lifecycle pill plus hint paragraph genuinely orients the user. Status casing inconsistency may confuse.
- **Agent (DOM consumer):** stable test ids on every tab and every action (`tasks-detail-tab-runs`, `tasks-detail-publish`, `tasks-detail-cancel`). DOM scrape will work.

---

## 14. Cross-Module Consistency Notes

- The 5-action header pattern (Edit, Cancel, Publish, Start run, Delete) matches the Skills detail header but uses different verbs (Skills uses Publish / Unpublish; Tasks uses Publish / Start run). Per-domain verb choice is fine.
- Cancel verb matches the Network shell cancel; the Run detail page uses `Kill run` instead. Inconsistent (P1 in module overview).
- Lifecycle hint paragraph is unique to Tasks; consider extracting as a shared `LifecycleHint` primitive for Sessions and Skills.

---

## 15. Open Questions

- Should the Orchestration tab be a sibling route (`/tasks/$id/orchestration`) given how dense it is?
- Should the Active Run card collapse when the task has no active run, or always render as `Execution` with a hint?
- Is the `Channel` pill the right surface for coordination channel binding? It implies "click to open a chat", but it does not. Consider a smaller monospace label.

---

## 16. Recommended Action Plan

1. `/impeccable layout web/src/routes/_app/tasks.tsx` (shared) to fix the parent SplitPane behavior.
2. `/impeccable clarify web/src/systems/tasks` to scrub em dashes from the four source files listed above.
3. `/impeccable typeset web/src/systems/tasks` to align status casing across header / runs / identity.
4. `/impeccable layout web/src/systems/tasks/components/tasks-detail-header.tsx` to reorder primary actions.
5. `/impeccable distill web/src/systems/tasks/components/tasks-detail-tabs.tsx` to group the 7 tabs.
6. `/impeccable distill web/src/systems/tasks/components/tasks-detail-overview-panel.tsx` to compress Active Run.
7. `/impeccable harden web/src/routes/_app/tasks.$id.tsx` to add a header skeleton.
8. `/impeccable polish web/src/systems/tasks` as the closing pass.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot.
- [x] No section empty.
- [x] Nielsen total 27/40, top of adequate band.
- [x] P0 to P3 with effort and command.
- [x] No hallucinated routes / props.
- [x] No em dashes in this report.
- [x] Length thorough not padded.
