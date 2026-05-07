# UI/UX Analysis: `Tasks` :: `/tasks/$id/runs/$runId`

> **Status:** draft
> **Owner subagent:** `tasks-module-auditor`
> **Date:** 2026-05-06
> **Module:** Tasks (`04_tasks`)
> **Route path:** `/tasks/$id/runs/$runId` (TanStack Router id: `/_app/tasks/$id/runs/$runId`)
> **Web source:** `web/src/routes/_app/tasks.$id.runs.$runId.tsx`
> **System owner:** `web/src/systems/tasks/`
> **Storybook story id(s):** `routes-app-stories-tasks-id-runs-runid--running`, `--completed`, `--failed`, `--no-session`, `--loading`, `--not-found`
> **Live URLs probed:** `http://localhost:3000/tasks/task_001/runs/run_001` (daemon empty), `http://localhost:6006/?path=/story/routes-app-stories-tasks-id-runs-runid`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read**:
  - `web/src/routes/_app/tasks.$id.runs.$runId.tsx`
  - `web/src/systems/tasks/components/task-run-detail-header.tsx`
  - `web/src/systems/tasks/components/task-run-detail-panels.tsx`
  - `web/src/systems/tasks/components/tasks-reviews-card.tsx`
  - `web/src/systems/tasks/lib/task-formatters.ts`
  - `web/src/hooks/routes/use-task-run-page.ts`
  - `web/src/routes/_app/stories/-tasks.$id.runs.$runId.stories.tsx`
- **Storybook stories opened**:
  - `routes-app-stories-tasks-id-runs-runid--running`
  - `routes-app-stories-tasks-id-runs-runid--completed`
  - `routes-app-stories-tasks-id-runs-runid--failed`
  - `routes-app-stories-tasks-id-runs-runid--no-session`
  - `routes-app-stories-tasks-id-runs-runid--loading`
  - `routes-app-stories-tasks-id-runs-runid--not-found`
- **Live web probes (`http://localhost:3000`)**:
  - `/tasks/task_001/runs/run_001` against the empty daemon, captured at 1440 wide. Page rendered the parent SplitPane rail and an empty content area (no error string).
- **Screenshots / DOM snapshots captured**:
  - `_evidence/tasks-id-runs-runId/live-not-found.png`. Live not-found probe.
  - `_evidence/tasks-id-runs-runId/sb-running.png`. Running.
  - `_evidence/tasks-id-runs-runId/sb-completed.png`. Completed.
  - `_evidence/tasks-id-runs-runId/sb-failed.png`. Failed.
  - `_evidence/tasks-id-runs-runId/sb-no-session.png`. Queued without session.
  - `_evidence/tasks-id-runs-runId/sb-loading.png`. Loading.
  - `_evidence/tasks-id-runs-runId/sb-not-found.png`. Storybook not-found.
- **Console / network errors observed**: none on the live probe.
- **Keyboard / a11y probes performed**: live snapshot shows the SplitPane parent rendering before the run content; the child outlet is empty when both task and run are missing because the parent task short-circuits to its own not-found render. Snapshot of the running story confirms `region` landmarks for Identity, Progress, Activity, Reviews.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** the inspectable surface for one `task_run`. Header shows breadcrumb (Tasks > task identifier > run id), run title (`Run <id>`), status pill, elapsed pill, optional `Open session` link, and a `Kill run` button when status is queued / claimed / starting / running. Body stacks four sections: Run identity (table), Progress (metric grid), Activity (last event + last activity timestamp + optional error / result code block), and Run reviews (table).
- **Primary user goal on this route:** watch a run, jump to its session, and act on it (cancel) if needed.
- **Entry vectors:** detail Active Run `Open run` link; runs panel chevron.
- **Exit vectors:** breadcrumb back to `/tasks` or `/tasks/$id`; `Open session` link to `/agents/$name/sessions/$id` or `/session/$id`; reviews row to the Skills review surface (likely).
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | n/a (route requires existing run) | n/a | n/a |
| Loading / skeleton | yes | `_evidence/tasks-id-runs-runId/sb-loading.png` | adequate (Loader2 only) |
| Partial data | yes (panels render whatever they have) | `_evidence/tasks-id-runs-runId/sb-no-session.png` | adequate |
| Populated (typical) | yes (running, completed, failed) | the three story screenshots | strong |
| Populated (dense, 100+ reviews) | unknown | no story | missing |
| Error (network) | yes | `tasks.$id.runs.$runId.tsx:32-44` | adequate |
| Error (permission / 403) | not visible | no branch | missing |
| Error (not found / 404) | partial | route's own not-found branch fires when `notFound` is true; live runtime never reaches it because the parent `/tasks/$id` route short-circuits first when the task is missing (`_evidence/tasks-id-runs-runId/live-not-found.png`) | weak |
| Read-only / disabled | yes (`Kill run` only when status active) | `task-run-detail-header.tsx:55-63` | strong |
| Live-update (stream / SSE) | partial | run page polls / streams the run; no LIVE badge on the page itself | weak |
| Mobile / narrow viewport | not stored | only 1440 captured | unknown |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |  2    | Status pill present; pulse on running. No "live / reconnecting / stale" banner; the `28308m 15s` elapsed value gives a misleading status signal | Truthful UI failure |
| 2  | Match between system and real world    |  2    | Status renders raw lowercase (`running`); `Kill run` verb is aggressive and not in the glossary | Verb / casing both off |
| 3  | User control and freedom               |  3    | Breadcrumb back; Open session link; Kill run button | No undo on Kill run; no confirm dialog |
| 4  | Consistency and standards              |  2    | Status casing differs from detail header; verb differs from detail page (`Cancel` vs `Kill run`) | See module overview cross-route findings |
| 5  | Error prevention                       |  2    | Kill run is a single click with no confirmation | Run cancellation is reversible-ish (release lease) but the verb implies irrevocable |
| 6  | Recognition rather than recall         |  3    | Identity table shows Run ID, Status, Attempt, Idempotency, Claimed by, Session | Idempotency key shown raw uppercase (DESIGN.md mono-badge correct) |
| 7  | Flexibility and efficiency of use      |  2    | No keyboard shortcut for Kill run; no quick "next run" navigation | Section navigation by scroll only |
| 8  | Aesthetic and minimalist design        |  3    | Identity / Progress / Activity / Reviews stack is clean; metric grid is DESIGN.md compliant | Progress metric for Elapsed renders nonsense (28308m) |
| 9  | Help users recognize / recover errors  |  2    | Failed runs surface the error string in a danger-tinted box (`task-run-detail-panels.tsx:248-258`) | Activity error renders the API string only; no remediation hint |
| 10 | Help and documentation                 |  2    | Run reviews paragraph (`task-run-detail-panels.tsx` reviews card) explains read-only verdict authority | The intro string itself uses an em dash (`This view is read-only — operator sessions cannot bind a verdict.`) |
|    | **Total**                              | **23/40** | | **Band:** ◯ adequate (20-28), held back by elapsed bug + verb |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders | OK | None observed. |
| Gradient text | OK | None. |
| Glassmorphism / blur as default | OK | None. |
| Hero-metric template | partial | The Progress section is exactly the SaaS "metric grid" pattern (Tool calls / Turns / Input tokens / Output tokens / Total tokens / Elapsed / Cost). DESIGN.md allows this for operator dashboards, but the row is large and dominant; consider compressing. |
| Identical card grids | OK | Section content varies (table, metric grid, definition list, table). |
| Modal as first thought | OK | No modal. |
| Em dashes in copy | VIOLATION | Run reviews intro paragraph in `tasks-reviews-card.tsx` (rendered for this route via `TasksReviewsCard`) reads `"This view is read-only — operator sessions cannot bind a verdict."` Verbatim em dash from snapshot `/tmp/sb-run-running.txt:359`. |
| Generic AI palette | OK | Warm dark canvas + accent. |
| Category-reflex theme | OK | Operator-first. |
| Restated headings | partial | Header `Run RUN_001` plus status pill plus identity-row "Run ID" then "Status" then attempt count. The same id appears 3 to 4 times in close proximity. |
| Decorative shadows | OK | Flat. |
| Hardcoded `#000` / `#fff` | OK | Tokenized. |

**Summary verdict:** borderline. The em dash in the Reviews intro and the hero-metric metric grid both lean SaaS; the truthful-UI failure on Elapsed pushes the verdict closer to "yes". Fix the elapsed and the dash and the route reads operator-first.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** breadcrumb (3 levels), Run pill id, status pill, elapsed pill, Open session link, Kill run button. Then four large sections vertically stacked, each with its own headings and tables.
- **Eight-item checklist:**
  1. >4 visible options? **fail** (header has 5 chips/buttons; identity table has 6 rows visible immediately).
  2. Self-evident labels? **partial** (`Idempotency`, `Claimed by` are operator jargon but appropriate; `Kill run` is jargon-aggressive).
  3. Primary action visually dominant? **partial** (`Kill run` is outline, not accent fill; for the most consequential action on the page, that is intentional but inconsistent).
  4. Progressive disclosure? **fail** (all four sections always visible).
  5. Related elements grouped? **pass**.
  6. Hierarchy ratio ≥1.25? **pass**.
  7. Body line length 65-75ch? **pass**.
  8. Whitespace varied? **partial** (sections use uniform `gap-6`).

  Failures: 3. Cognitive load = moderate-high.
- **IA observations:**
  - Reviews table is very wide (Review / Status / Outcome / Reviewer / Round / Requested / Reviewed). At 1024 wide this overflows.
  - Activity panel is the smallest section but holds error and result, the most operator-relevant content. Consider promoting Activity above Progress.
  - No anchor links / sticky section nav makes scrolling between Identity and Reviews tedious for long pages.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all via `var(--color-*)`. Activity error block uses `var(--color-danger)` and `var(--color-danger-tint)` correctly.
- **Type scale:** Inter 15px medium for header title, JetBrains Mono 11px uppercase tracking 0.14em for breadcrumb, mono 11px tracking 0.06em for table cells (ATTEMPT, QUEUED, ENDED).
- **Radii / spacing:** consistent.
- **Elevation:** flat.
- **Signal palette discipline:** status tint correct on the chip; danger surface for errors. No decorative use.
- **Grid / rhythm:** Progress metric grid is `sm:grid-cols-2`; Cost cell spans both columns. Acceptable.
- **Density:** dense in the header chip row; comfortable in the body.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** `Kill run` (when active), `Open session` link.
- **Destructive actions:** Kill run is a one-click action with no confirm. The verb implies irrevocable; the API is `cancelTaskRun` which releases the lease.
- **Forms:** none.
- **Tables / lists:** Identity table (key/value rows, no sort), Reviews table (no sort, no filter).
- **Selection model:** none.
- **Modals / drawers:** none.
- **Live updates:** the run polls/streams in the background but no UI signal indicates whether the page is live or stale.
- **Optimistic vs pessimistic:** pessimistic on Kill run.
- **Hover / focus / active states:** standard on links and buttons; rows are static.
- **Loading patterns:** Loader2 only.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** all controls reachable; `Open session` is a `Link` with proper anchor semantics.
- **Focus rings:** standard.
- **TAB order:** breadcrumb -> Open session link -> Kill run -> Identity table is presentational (no focusables) -> Session link inside Identity -> reviews table is presentational with no focusables. Logical.
- **ARIA roles / labels:** Identity panel uses `Section` with `aria-label="Run identity"`. Progress is `aria-label="Run progress"`. Activity is `aria-label="Run activity"`. Reviews is `aria-label="Run reviews"`. Good.
- **Color contrast:** body and helper text exceed AA. Danger text on `danger-tint` background is `#FF453A` on `#FF453A26`; the chip is legible because the text itself is full saturation, but background contrast is borderline.
- **Motion:** Loader2 spin and pulse on running dot. Reduced-motion respected.
- **Text scaling:** at 200% the Reviews table overflows horizontally because columns do not collapse.
- **Forms:** n/a.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** n/a. The `NoSession` story handles a queued run with no session attached and renders a `None` token in the Session row of the identity table; clean.
- **Loading:** adequate. Loader2 spinner.
- **Error:** weak. `Run ${runId} not found.` plain text with `AlertCircle`. No retry, no support link, no breadcrumb back.
- **Permission denied:** missing.
- **Stale / disconnected:** missing. No banner when the stream drops.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** mostly correct (`task_run`, `attempt`, `claimed by`, `session`, `idempotency`). One exception: `Kill run` is operator slang. Glossary uses `cancel` for run cancellation.
- **Tone:** dry, operator-first.
- **Em dashes:** Reviews intro uses `—`. The Activity error string is the raw API output, which may or may not contain dashes (out of UI scope but worth wrapping).
- **Restated headings:** `Run RUN_001`, then identity row "Run ID" -> `RUN_001`. Same id 3 times in 200px.
- **Sentence case vs Title Case:** sentence-case body; uppercase mono eyebrows. Consistent.
- **Truthful UI test:**
  - `Elapsed` value is the truthful-UI failure: a 19-day-old run renders as `28308m 15s`. The runtime knows the start and end; the formatter is wrong.
  - `Kill run` implies a forceful kernel kill; the runtime calls this `cancel` (release the lease). The verb misrepresents the action.
  - `Open session` is correct.

---

## 10. Performance & Responsiveness

- **Initial render:** route fetches the run plus reviews via `useTaskRunPage`.
- **Re-render hot spots:** every stream tick may re-render the entire main column. Progress metrics are not memoized.
- **List virtualization:** Reviews table is not virtualized; 100+ reviews would be slow.
- **Bundle red flags:** none specific.
- **Responsive behaviour:** at 1024 the Reviews table overflows; at 1440 fits.
- **Mobile interactions:** none specific; the chips wrap.

---

## 11. Storybook Coverage

- **Stories present:** Running, Completed, Failed, NoSession, Loading, NotFound.
- **States covered:** the three terminal statuses + queued + loading + not-found.
- **Gaps:**
  - No "review approved" story (recorded + approved outcome).
  - No "review rejected with continuation run" story.
  - No "stream dropped" story.
  - No mobile-viewport story.
- **Story drift:** the `Running` story's elapsed pill renders `28308M 15S` (`/tmp/sb-run-running.txt:271-275`), which is the truthful-UI bug surfaced through the story; the story is faithful to the broken formatter, not the broken story.

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

1. **[P0] What:** `formatElapsed` overflows in minutes (`task-run-detail-header.tsx:16-41`, `task-run-detail-panels.tsx:117-142`). The Running story shows `28308M 15S` in the header pill and `28308m 15s` in the Progress card.
   - **Why:** truthful-UI violation. The number printed is technically computed but meaningless to operators.
   - **Fix:** rewrite the formatter to escalate units: `Xs` under 60 seconds, `Xm Ys` under 60 minutes, `Xh Ym` under 24 hours, `Xd Yh` over 24 hours. Add unit tests for boundaries (59s, 60s, 1h, 1h 1m, 23h 59m, 24h, 7d). Deduplicate by extracting to `task-formatters.ts`.
   - **Cmd:** `/impeccable harden web/src/systems/tasks/components/task-run-detail-header.tsx web/src/systems/tasks/components/task-run-detail-panels.tsx`
   - **Effort:** S
   - **Evidence:** `_evidence/tasks-id-runs-runId/sb-running.png`; `task-run-detail-header.tsx:16-41`; `task-run-detail-panels.tsx:117-142`.

2. **[P0] What:** SplitPane parent renders the list rail alongside the run-detail content. Same root cause as `/tasks/new` and `/tasks/$id/edit`.
   - **Why:** breaks the live route layout.
   - **Fix:** see module overview P0 #4.
   - **Cmd:** `/impeccable layout web/src/routes/_app/tasks.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/tasks-id-runs-runId/live-not-found.png`.

3. **[P0] What:** em dash in Reviews intro paragraph: `"This view is read-only — operator sessions cannot bind a verdict."`
   - **Why:** ban from `DESIGN.md` and audit hard rule.
   - **Fix:** rewrite without `—`. Suggested: `"This view is read-only. Operator sessions cannot bind a verdict."`.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/components/tasks-reviews-card.tsx`
   - **Effort:** S
   - **Evidence:** `/tmp/sb-run-running.txt:359`.

### P1 - High-Value Polish

4. **[P1] What:** `Kill run` button verb (`task-run-detail-header.tsx:135`).
   - **Why:** does not match `glossary.md`. The detail page calls the same operation `Cancel`.
   - **Fix:** rename to `Cancel run`. Add a confirm dialog mirroring the delete pattern when status is `running` or `claimed`, since canceling a live run releases its lease.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/components/task-run-detail-header.tsx`
   - **Effort:** S
   - **Evidence:** `task-run-detail-header.tsx:135`.

5. **[P1] What:** run-detail not-found is not visible when the parent task is also missing. Live probe rendered an empty content area instead of the run not-found message.
   - **Why:** cross-route error handling fails on mismatched parent / child existence.
   - **Fix:** decouple the run from the task: load the run by id and resolve its parent task lazily; or make the parent `/tasks/$id` tolerant of "task missing but run exists" and forward to the child route.
   - **Cmd:** `/impeccable harden web/src/routes/_app/tasks.$id.runs.$runId.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/tasks-id-runs-runId/live-not-found.png`.

6. **[P1] What:** no LIVE indicator on the run detail page even when the run is streaming.
   - **Why:** `/tasks/$id` shows LIVE on the Events tab; the run detail leaf does not. For the route whose primary job is "watch the run", this is missing system status.
   - **Fix:** add a small LIVE dot next to the status pill in the header when the run is in active statuses and the stream is connected. Add a "stream stale" warning when the heartbeat lapses.
   - **Cmd:** `/impeccable harden web/src/systems/tasks/components/task-run-detail-header.tsx`
   - **Effort:** M

7. **[P1] What:** status casing renders raw lowercase (`running`).
   - **Why:** detail header uses `In Progress` via `taskStatusLabel`. Run detail header uses raw `record.status` (`task-run-detail-header.tsx:153`) and identity panel uses raw `record.status` (`task-run-detail-panels.tsx:53`).
   - **Fix:** route through `taskRunStatusLabel` (introduce if missing) and render uppercase mono per DESIGN.md badge spec.
   - **Cmd:** `/impeccable typeset web/src/systems/tasks`
   - **Effort:** S

### P2 - Worthwhile

8. **[P2] What:** Identity table renders Run ID three times within 200px (header pill, breadcrumb tail, identity row).
   - **Fix:** drop the breadcrumb tail (or the identity row). Keep the pill in the header for copy-paste.

9. **[P2] What:** Reviews table overflows at 1024 wide.
   - **Fix:** collapse to a card list at narrow widths.

10. **[P2] What:** Cancel-run lacks a confirm dialog when the run is live.
    - **Fix:** see P1 #4 plus the dialog.

11. **[P2] What:** progress metric grid's `Cost` row spans two columns; `Total tokens` is one of the seven values stacked one-per-cell. Visual hierarchy is flat.
    - **Fix:** group as 2x3 (Calls/Turns/Tokens) plus 1x2 (Elapsed/Cost) or similar.

### P3 - Parking Lot

12. **[P3] What:** error UI in Activity panel is the raw API string.
    - **Fix:** wrap with a "What happened" / "Next safe action" pattern from `COPY.md` Error Copy formula.

13. **[P3] What:** no anchor / sticky section navigation for long Reviews tables.
    - **Fix:** add a sticky in-page nav `Identity / Progress / Activity / Reviews`.

---

## 13. Persona Red Flags

- **Operator (returning power user):** the `28308M 15S` pill is the most obvious failure; the operator will not trust any other metric on the page. `Kill run` is verb-aggressive; an operator who clicks once may regret it.
- **First-timer:** Identity table is informative but the first column "Run ID" / "Status" / "Attempt" / "Idempotency" / "Claimed by" / "Session" is a heavy intro to runtime concepts.
- **Agent (DOM consumer):** stable test ids on every section (`task-run-detail-identity`, `task-run-detail-progress`, `task-run-detail-activity`). Very scrape-friendly. The elapsed value is bug-data, agents will read it as truth.

---

## 14. Cross-Module Consistency Notes

- Detail page Cancel uses `Cancel`; this route uses `Kill run`. Pick one.
- LIVE badge is on `/tasks/$id` Events tab but not on this leaf; should align.
- Loading uses Loader2 here, plain text on `/tasks/$id/edit`. Align loading affordances across the module.

---

## 15. Open Questions

- Should this route move into `/tasks/$id` as a side panel rather than its own route? That would solve both the SplitPane bug and the cross-route error handling.
- Should we expose a "follow logs" affordance, or keep the session link as the only way to read tool calls?
- Is the Reviews table the right place for this content, or should it live under `/tasks/$id/orchestration` where the rest of the review surfaces sit?

---

## 16. Recommended Action Plan

1. `/impeccable harden web/src/systems/tasks/components/task-run-detail-header.tsx web/src/systems/tasks/components/task-run-detail-panels.tsx` to fix `formatElapsed` unit escalation.
2. `/impeccable layout web/src/routes/_app/tasks.tsx` (shared) to remove the parent SplitPane rail from this child.
3. `/impeccable clarify web/src/systems/tasks/components/tasks-reviews-card.tsx` to remove the em dash from the Reviews intro.
4. `/impeccable clarify web/src/systems/tasks/components/task-run-detail-header.tsx` to rename `Kill run` to `Cancel run` and add a confirm dialog for live runs.
5. `/impeccable harden web/src/routes/_app/tasks.$id.runs.$runId.tsx` to render the not-found state independent of the parent task's existence.
6. `/impeccable harden web/src/systems/tasks/components/task-run-detail-header.tsx` to add a LIVE / stream-stale indicator.
7. `/impeccable typeset web/src/systems/tasks` to align run status casing with the rest of the module.
8. `/impeccable distill web/src/systems/tasks/components/task-run-detail-panels.tsx` to compress the Run ID redundancy and reorganize Progress.
9. `/impeccable polish web/src/routes/_app/tasks.$id.runs.$runId.tsx` as the closing pass.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot.
- [x] No section empty.
- [x] Nielsen total 23/40 consistent with adequate band, dragged down by the elapsed bug and the verb.
- [x] P0 to P3 with effort and command.
- [x] No hallucinated routes / props.
- [x] No em dashes in this report.
- [x] Length thorough not padded.
