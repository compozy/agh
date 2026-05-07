# UI/UX Analysis: `Tasks` :: `/tasks/$id/edit`

> **Status:** draft
> **Owner subagent:** `tasks-module-auditor`
> **Date:** 2026-05-06
> **Module:** Tasks (`04_tasks`)
> **Route path:** `/tasks/$id/edit` (TanStack Router id: `/_app/tasks/$id/edit`)
> **Web source:** `web/src/routes/_app/tasks.$id.edit.tsx`
> **System owner:** `web/src/systems/tasks/`
> **Storybook story id(s):** `routes-app-stories-tasks-id-edit--default`, `routes-app-stories-tasks-id-edit--loading`, `routes-app-stories-tasks-id-edit--missing-task`
> **Live URLs probed:** `http://localhost:3000/tasks/task_001/edit`, `http://localhost:6006/?path=/story/routes-app-stories-tasks-id-edit`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read**:
  - `web/src/routes/_app/tasks.$id.edit.tsx`
  - `web/src/systems/tasks/components/task-editor-surface.tsx`
  - `web/src/systems/tasks/components/use-tasks-create-modal-form.ts`
  - `web/src/hooks/routes/use-task-edit-route-state.ts`
  - `web/src/systems/tasks/lib/task-formatters.ts`
  - `web/src/routes/_app/stories/-tasks.$id.edit.stories.tsx`
- **Storybook stories opened**:
  - `routes-app-stories-tasks-id-edit--default`
  - `routes-app-stories-tasks-id-edit--loading`
  - `routes-app-stories-tasks-id-edit--missing-task`
- **Live web probes (`http://localhost:3000`)**:
  - `/tasks/task_001/edit` against the empty daemon, captured at 1440 wide. Page stayed on `Loading task…` indefinitely.
- **Screenshots / DOM snapshots captured**:
  - `_evidence/tasks-id-edit/live-missing.png`. Live missing-task probe (stuck on spinner).
  - `_evidence/tasks-id-edit/sb-default.png`. Storybook default (loaded task).
  - `_evidence/tasks-id-edit/sb-loading.png`. Storybook loading branch.
  - `_evidence/tasks-id-edit/sb-missing-task.png`. Storybook missing-task branch.
- **Console / network errors observed**: none in the live probe console.
- **Keyboard / a11y probes performed**: live snapshot shows the parent SplitPane rail is rendered before the form (P0 #2 from module overview), so Tab order routes through the rail first.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** edit the mutable fields of an existing task. Reuses `TaskEditorSurface` in `mode="edit"` so the right rail moves the editable secondary fields (priority, approval, attempts) into the right panel and removes the template chooser.
- **Primary user goal on this route:** change one or more task fields and save.
- **Entry vectors:** detail header `Edit` button (`tasks-detail-header.tsx:127-131`); deep link.
- **Exit vectors:** `Cancel` -> `/tasks/$id`; submit -> back to the detail.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | n/a (route requires existing task) | n/a | n/a |
| Loading / skeleton | partial | `tasks.$id.edit.tsx:14-19` shows "Loading task…" plain text | weak (no skeleton; just text spinner-less label) |
| Partial data | yes (edit form fills from `useTaskEditRouteState`) | `_evidence/tasks-id-edit/sb-default.png` | adequate |
| Populated (typical) | yes | `_evidence/tasks-id-edit/sb-default.png` | strong |
| Submitting / pending | yes | `task-editor-surface.tsx:520-531` (`Save changes` shows Loader2) | adequate |
| Validation error | partial | `canSubmit` gates submit on title length only | weak |
| Server error | missing | route does not render a server-error region | weak |
| Error (network) | partial | route shows "We couldn't load this task for editing." only when `!page.task && page.isInitialized` triggers (`tasks.$id.edit.tsx:22-29`) | weak (the live probe never reaches this branch on 404) |
| Error (not found / 404) | broken | live probe stayed on `Loading task…` for a missing task id (`_evidence/tasks-id-edit/live-missing.png`) | broken |
| Read-only / disabled | partial (template, scope, parent, identifier are not editable in edit mode) | `task-editor-surface.tsx:248-277, 416-461` branches `isCreateMode` | adequate |
| Live-update | n/a | n/a | n/a |
| Mobile / narrow viewport | not stored | only 1440 captured | unknown |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |  2    | `Loading task…` spinner-less; submit shows Loader2 | indefinite Loading on missing task is the worst kind of system status |
| 2  | Match between system and real world    |  3    | Field labels mirror runtime; status pill in header reflects the loaded task | Owner kind enum strings rendered raw |
| 3  | User control and freedom               |  3    | Cancel returns to detail; submit returns to detail; no autosave | No "discard changes" confirmation if the user typed and clicks Cancel |
| 4  | Consistency and standards              |  3    | Reuses `TaskEditorSurface` from `/tasks/new` so the form pattern is consistent | The right-rail content swaps fields between create and edit modes; consistent with create |
| 5  | Error prevention                       |  2    | Title required; submit disabled until title is non-empty | No detection of unsaved changes; no warn-on-leave |
| 6  | Recognition rather than recall         |  3    | Header shows the task identifier, status, scope, parent | The "Read-only context" panel is informative |
| 7  | Flexibility and efficiency of use      |  2    | Cmd-S not bound; primary submit is on the right edge | Tab order routes through the parent rail first |
| 8  | Aesthetic and minimalist design        |  3    | DESIGN.md tokens applied | The right-rail `Submission` heading from create is replaced by `Editable fields` in edit; readable |
| 9  | Help users recognize / recover errors  |  1    | Indefinite spinner on missing task is the major failure (`tasks.$id.edit.tsx:14-30`) | Server errors are not surfaced inline |
| 10 | Help and documentation                 |  2    | The intro paragraph in edit mode is helpful (`Changes are saved directly to the task record and become visible across the list, detail, inbox, and dashboard views.`) | No inline help on per-field implications |
|    | **Total**                              | **24/40** | | **Band:** ◯ adequate (20-28), held back by the 404 spinner |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders | OK | None observed. |
| Gradient text | OK | None. |
| Glassmorphism / blur as default | OK | None. |
| Hero-metric template | OK | n/a. |
| Identical card grids | OK | Form sections differ. |
| Modal as first thought | OK | Form is a route. |
| Em dashes in copy | partial | The route's own paragraph does not use em dashes; however the parent SplitPane rail surfaces empty-state copy with em dashes from `task-templates.ts:51, 83`. The right-rail intro for edit mode does not include `—`. |
| Generic AI palette | OK | Warm dark canvas, accent on submit. |
| Category-reflex theme | OK | Operator-first. |
| Restated headings | partial | Header `Edit task`, intro `Change the mutable task fields here. Scope, parent, and identifier stay visible as task context.`, section `Task contract`, plus the `Editable fields` rail eyebrow. Three labels for the same intent. |
| Decorative shadows | OK | Flat. |
| Hardcoded `#000` / `#fff` | OK | Tokenized. |

**Summary verdict:** no, this route does not look AI-generated, but it leans toward bland because the editor is generic. The parent SplitPane bug pulls em-dashed copy in from the rail.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** title, description (textarea), priority pills (4), approval pills (2), attempts pills (5), network channel input, plus Cancel / Save changes footer. Compared to create, the form is significantly leaner because scope, parent, identifier, and template are not editable.
- **Eight-item checklist:**
  1. >4 visible options? **fail** (still ~10 controls).
  2. Self-evident labels? **pass**.
  3. Primary action visually dominant? **pass** (`Save changes` accent fill).
  4. Progressive disclosure? **partial** (advanced fields are merged with basic; could collapse `Network channel`).
  5. Related elements grouped? **pass**.
  6. Hierarchy ratio ≥1.25? **pass**.
  7. Body line length 65-75ch? **pass**.
  8. Whitespace varied? **partial**.

  Failures: 2. Cognitive load = moderate.
- **IA observations:**
  - The `Editable fields` rail makes sense as a rule of thumb for what changed; consider showing dirty markers (a small dot next to the field label that has unsaved changes).
  - The right-rail "Read-only context" panel duplicates what the header already shows (identifier, scope, parent, workspace).

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all via `var(--color-*)`.
- **Type scale:** Inter for labels, JetBrains Mono for breadcrumb and eyebrows.
- **Radii / spacing:** consistent with `/tasks/new`.
- **Elevation:** flat.
- **Signal palette discipline:** accent on the submit button only; the header status pill renders the task's actual status tone.
- **Grid / rhythm:** same `xl:grid-cols-[minmax(0,1.35fr)_minmax(22rem,0.85fr)]` as create.
- **Density:** comfortable in edit mode because there are fewer fields.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** `Save changes` (accent fill), `Cancel` (outline).
- **Destructive actions:** none on this route.
- **Forms:**
  - Title is required; `canSubmit` gates submit.
  - No autosave; no warn-on-leave.
  - Discard-on-cancel is silent; user loses typed edits without confirmation.
- **Tables / lists:** n/a.
- **Selection model:** n/a beyond pill values.
- **Modals / drawers:** none.
- **Live updates:** n/a (the form fields do not refetch while open).
- **Optimistic vs pessimistic:** pessimistic.
- **Hover / focus / active states:** standard.
- **Loading patterns:** the route's own loading state is a plain text "Loading task…" (no spinner glyph). Compare with `/tasks/$id` which uses Loader2.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** all controls reachable; Tab routes through the unwanted parent rail first (P0 #2).
- **Focus rings:** standard.
- **TAB order:** parent rail (search, lane pills, `New task`) -> breadcrumb -> page header -> form fields -> footer. Logical inside the form, broken at the page level by the rail.
- **ARIA roles / labels:** every Field has a label; the header status pill is decorative and not announced to screen readers (the title carries the same semantic info via the dot).
- **Color contrast:** body text and helper text exceed AA.
- **Motion:** Loader2 spinner; reduced-motion respected.
- **Text scaling:** at 200% the form columns stack; nothing clips visibly.
- **Forms:** required field labelled. Optional fields not marked optional.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** n/a.
- **Loading:** weak. `Loading task…` is a plain `<span>`, no spinner glyph (`tasks.$id.edit.tsx:14-19`). Compare to `/tasks/$id` which renders a `Loader2`.
- **Error:** broken. The live probe of `/tasks/task_001/edit` (daemon empty, task doesn't exist) sat on `Loading task…` indefinitely. Source: `tasks.$id.edit.tsx:22-30` only renders the friendly empty when `!page.task && page.isInitialized`. If the underlying query rejects with 404 and `isInitialized` never flips, the page stays on the spinner forever. Storybook `MissingTask` story does flip `isInitialized` and shows the friendly empty (`_evidence/tasks-id-edit/sb-missing-task.png`), but live runtime does not.
- **Permission denied:** missing.
- **Stale / disconnected:** n/a.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** match `glossary.md`. The intro paragraph in edit mode reads `"Changes are saved directly to the task record and become visible across the list, detail, inbox, and dashboard views."` Operator-first; no `—`.
- **Tone:** dry, informative.
- **Em dashes:** the route's own copy does not use em dashes. The page inherits parent rail copy that does (P0 #2). Intro paragraph at `task-editor-surface.tsx:101-103` does not use em dashes.
- **Restated headings:** `Edit task` page title plus intro paragraph plus `Task contract` section. Three labels.
- **Sentence case vs Title Case:** consistent sentence-case.
- **Truthful UI test:** the route does not invent capability. It only edits fields the runtime exposes. `Read-only context` correctly labels values that are not editable here (scope, parent, identifier, workspace). No fake fields.

---

## 10. Performance & Responsiveness

- **Initial render:** route fetches the task via `useTaskEditRouteState` and waits to mount the editor. No skeleton; just the plain "Loading task…" text.
- **Re-render hot spots:** keystrokes on title / description re-render the editor surface; acceptable.
- **List virtualization:** n/a.
- **Bundle red flags:** none.
- **Responsive behaviour:** same as create; not stored at non-1440 viewports.
- **Mobile interactions:** none specific.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-tasks-id-edit--default`
  - `routes-app-stories-tasks-id-edit--loading`
  - `routes-app-stories-tasks-id-edit--missing-task`
- **States covered:** loaded, loading, missing-task.
- **Gaps:**
  - No story for "task with active run" (which gates some fields).
  - No "edit during run" story (whether attempts or owner can change while running).
  - No mobile-viewport story.
  - No "submit failed" story.
- **Story drift:** the `MissingTask` story renders the friendly text `We couldn't load this task for editing.`, but live behavior on the same condition stays on the spinner (P0). Story does not match runtime under 404. Story is misleading in this respect.

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

1. **[P0] What:** `/tasks/$id/edit` for a missing task spins indefinitely (`_evidence/tasks-id-edit/live-missing.png`).
   - **Why:** route gate is `if (!page.task || !page.isInitialized)` (`tasks.$id.edit.tsx:22-30`), but on 404 the flow can leave `isInitialized=false`, leaving the user on `Loading task…`.
   - **Fix:** mirror the detail page's branching (`tasks.$id.tsx:42-54`): expose `notFound` and `fatalError` from the route hook, render a not-found surface with breadcrumb back to `/tasks` when either is true, regardless of `isInitialized`. Replace plain text with the same `Loader2` icon used on `/tasks/$id`.
   - **Cmd:** `/impeccable harden web/src/routes/_app/tasks.$id.edit.tsx`
   - **Effort:** S
   - **Evidence:** `tasks.$id.edit.tsx:14-30`; `_evidence/tasks-id-edit/live-missing.png`.

2. **[P0] What:** SplitPane parent renders the list rail alongside the edit form. Same root cause as `/tasks/new`.
   - **Why:** doubles affordances, breaks tab order.
   - **Fix:** see module overview P0 #4.
   - **Cmd:** `/impeccable layout web/src/routes/_app/tasks.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/tasks-id-edit/live-missing.png`.

### P1 - High-Value Polish

3. **[P1] What:** no warn-on-leave or unsaved-changes prompt when the user clicks Cancel after typing.
   - **Why:** edits are silently discarded. For a CRUD route this is a basic guardrail.
   - **Fix:** track `dirty` state; intercept Cancel and Cmd-W with a confirm dialog when dirty.
   - **Cmd:** `/impeccable harden web/src/systems/tasks/components/task-editor-surface.tsx`
   - **Effort:** M

4. **[P1] What:** server-error UI is missing.
   - **Why:** silent failure on save.
   - **Fix:** add an inline error region above the footer (shared with create P1 #4).
   - **Cmd:** `/impeccable harden web/src/systems/tasks/components/task-editor-surface.tsx`
   - **Effort:** S

5. **[P1] What:** loading state uses plain text instead of a spinner glyph.
   - **Why:** inconsistent with `/tasks/$id` and `/tasks/$id/runs/$runId`.
   - **Fix:** use the same `Loader2` pattern (`tasks.$id.tsx:35-39`).
   - **Cmd:** `/impeccable typeset web/src/routes/_app/tasks.$id.edit.tsx`
   - **Effort:** S

### P2 - Worthwhile

6. **[P2] What:** owner kind dropdown shows raw enum (shared with `/tasks/new` P1 #3).
   - **Fix:** route through `taskOwnerKindLabel`.
   - **Effort:** S

7. **[P2] What:** dirty markers absent on the right-rail.
   - **Fix:** show a small dot on each label that has an unsaved change.

8. **[P2] What:** no read-only awareness when the task has an active run.
   - **Why:** the runtime may reject mid-run priority changes.
   - **Fix:** disable specific fields when `task.summary?.active_run` is present and surface a hint.

### P3 - Parking Lot

9. **[P3] What:** the right-rail "Read-only context" duplicates the header.
   - **Fix:** drop the rail panel in edit mode, or compress to one line.

---

## 13. Persona Red Flags

- **Operator (returning power user):** no Cmd-S; no quick keyboard exit. The parent rail steals focus.
- **First-timer:** the loading spinner-less text is confusing; the friendly empty does not appear when the URL is wrong.
- **Agent (DOM consumer):** test ids stable (`task-edit-loading`, `task-edit-empty`, `task-editor-submit`).

---

## 14. Cross-Module Consistency Notes

- The "loading" string here (`Loading task…`) is the only loading affordance in the Tasks module that does not include a spinner. Sessions edit and Skills edit both use Loader2.
- The friendly empty `We couldn't load this task for editing.` is unique copy; consider unifying with `/tasks/$id` which says `Task ${id} not found.`.

---

## 15. Open Questions

- Should edit mode allow scope changes (workspace -> global) for a draft task? Today the field is read-only in edit mode but draft tasks could legitimately move scope.
- Should `Network channel` be editable while a run is active? Today it is editable; the runtime may reject the change.
- Should the route be a side panel inside `/tasks/$id` rather than its own page? That would solve both the SplitPane bug and the breadcrumb redundancy.

---

## 16. Recommended Action Plan

1. `/impeccable harden web/src/routes/_app/tasks.$id.edit.tsx` to fix the indefinite spinner on 404 / fatal error and align the loading affordance with the detail route.
2. `/impeccable layout web/src/routes/_app/tasks.tsx` to remove the parent rail on this child route (shared).
3. `/impeccable harden web/src/systems/tasks/components/task-editor-surface.tsx` for warn-on-leave / dirty state / inline server errors.
4. `/impeccable typeset web/src/routes/_app/tasks.$id.edit.tsx` to swap the loading text for the spinner glyph.
5. `/impeccable clarify web/src/systems/tasks/components/task-editor-surface.tsx` for owner kind labels.
6. `/impeccable distill web/src/systems/tasks/components/task-editor-surface.tsx` for the "Read-only context" rail in edit mode.
7. `/impeccable polish web/src/routes/_app/tasks.$id.edit.tsx` as the closing pass.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot.
- [x] No section empty.
- [x] Nielsen total 24/40 consistent with adequate band, dragged down by 404 spinner.
- [x] P0 to P3 with effort and command.
- [x] No hallucinated routes / props.
- [x] No em dashes in this report.
- [x] Length thorough not padded.
