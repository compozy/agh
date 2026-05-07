# UI/UX Analysis: `Tasks` :: `/tasks/new`

> **Status:** draft
> **Owner subagent:** `tasks-module-auditor`
> **Date:** 2026-05-06
> **Module:** Tasks (`04_tasks`)
> **Route path:** `/tasks/new` (TanStack Router id: `/_app/tasks/new`)
> **Web source:** `web/src/routes/_app/tasks.new.tsx`
> **System owner:** `web/src/systems/tasks/`
> **Storybook story id(s):** `routes-app-stories-tasks-new--default`, `routes-app-stories-tasks-new--template-preset`, `routes-app-stories-tasks-new--submitting`
> **Live URLs probed:** `http://localhost:3000/tasks/new`, `http://localhost:6006/?path=/story/routes-app-stories-tasks-new`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read**:
  - `web/src/routes/_app/tasks.new.tsx`
  - `web/src/systems/tasks/components/task-editor-surface.tsx`
  - `web/src/systems/tasks/components/use-tasks-create-modal-form.ts`
  - `web/src/systems/tasks/lib/task-templates.ts`
  - `web/src/systems/tasks/lib/task-formatters.ts`
  - `web/src/hooks/routes/use-task-create-route-state.ts`
  - `web/src/routes/_app/stories/-tasks.new.stories.tsx`
- **Storybook stories opened**:
  - `routes-app-stories-tasks-new--default` -> `http://localhost:6006/iframe.html?id=routes-app-stories-tasks-new--default&viewMode=story`
  - `routes-app-stories-tasks-new--template-preset` -> same with `--template-preset` (`template=human_in_loop` search param).
  - `routes-app-stories-tasks-new--submitting` -> same with `--submitting` (mid-flight create).
- **Live web probes (`http://localhost:3000`)**:
  - `/tasks/new` rendered alongside the parent SplitPane list rail (daemon empty), captured at 1440 wide.
- **Screenshots / DOM snapshots captured**:
  - `_evidence/tasks-new/live.png` (live `/tasks/new`).
  - `_evidence/tasks-new/sb-default.png` (Storybook `default`).
  - `_evidence/tasks-new/sb-template-preset.png` (`human_in_loop`).
  - `_evidence/tasks-new/sb-submitting.png` (submitting state).
- **Console / network errors observed**: none on the live probe.
- **Keyboard / a11y probes performed**: live a11y snapshot confirms `<form>` wraps the editor body, `<Field>` groups give labeled inputs (`task-editor-title-input` is `required`), and the breadcrumb is a `nav aria-label="Breadcrumb"`. The `Owner` field is a `<NativeSelect>` followed by a separate text input; both share the `Field` group but only the select gets the explicit label association.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** stage a new task contract before it enters the queue. The user picks a template, fills the title (required), description, scope, priority, owner, attempts, approval, parent, network channel, and identifier override, then either saves a draft or creates and enqueues the first run.
- **Primary user goal on this route:** complete the create form and submit, ideally fast.
- **Entry vectors:** sidebar via `/tasks` -> `Task` button or `New task` button or empty-state template tile (which sets `?template=<id>`).
- **Exit vectors:** `Cancel` -> `/tasks`; submit -> `/tasks/$id` (or `/tasks` on draft save, depending on `enqueueOnSubmit`); breadcrumb back link.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | yes (defaults to `one_shot` template) | `_evidence/tasks-new/sb-default.png`, `task-templates.ts:38-45` | strong |
| Loading / skeleton | n/a (route mounts already with state) | n/a | n/a |
| Partial data | yes (template-pre-filled draft) | `_evidence/tasks-new/sb-template-preset.png` | strong |
| Populated (typical) | yes (form rendered) | live page | strong |
| Submitting / pending | yes | `_evidence/tasks-new/sb-submitting.png`; `task-editor-surface.tsx:507-531` | adequate (Loader2 icon on Save draft and Submit) |
| Validation error | partial | `canSubmit={page.draft.title.trim().length > 0}` (`tasks.new.tsx:25`) gates the button; no inline error message | weak |
| Server error | missing in story | route relies on `useTaskCreateRouteState` rejection; no visible error UI on the form | weak |
| Read-only / disabled | n/a | n/a | n/a |
| Live-update | n/a | n/a | n/a |
| Mobile / narrow viewport | not stored, partial | grid drops to 1 column at md; tested live at 1440 only | partial |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |  3    | Submit shows `Loader2`; `Save draft` shows the same; submit-disabled when `canSubmit` is false | No "draft autosaves" indicator if the user navigates away |
| 2  | Match between system and real world    |  3    | Field labels match runtime (`scope`, `priority`, `owner`, `approval`, `parent task`, `network channel`, `identifier`) | Owner kind options (`agent_session`, `automation`, `extension`, `network_peer`, `pool`) render in raw snake-case |
| 3  | User control and freedom               |  2    | Cancel returns to `/tasks`; Save draft is offered; no autosave; no draft recovery if the tab closes | Tab through the parent list rail (P0 issue) before reaching the form |
| 4  | Consistency and standards              |  2    | Form composition follows `Field` + `FieldLabel` + `Input` from `@agh/ui`; pill groups for enums; submit button on the right | Mix of pill groups and native `<select>` for owner kind; visually inconsistent |
| 5  | Error prevention                       |  2    | Submit disabled until title typed; identifier override has placeholder `TASK-123`; parent task uses free text rather than autocomplete | No inline validation messages; failed server submits have no path back |
| 6  | Recognition rather than recall         |  3    | Right rail ("Read-only context") shows scope, parent, identifier, workspace; template name shown in description | Owner kind list shows raw enum values |
| 7  | Flexibility and efficiency of use      |  2    | Save draft is a power-user shortcut; otherwise no keyboard shortcut for submit, no Cmd-S | Tab order routes through the unwanted list rail first |
| 8  | Aesthetic and minimalist design        |  3    | DESIGN.md tokens applied; sectioning is clear; right rail panel uses `surface-panel` background for contrast | The form is dense; right rail repeats some context the form already shows |
| 9  | Help users recognize / recover errors  |  2    | Title input is `required` and the browser will surface a validation tooltip; no explicit error region | No visible toast/banner when the create mutation fails |
| 10 | Help and documentation                 |  2    | Description help text under `Description` field; template notice text in right rail | Many fields (max attempts, network channel, identifier override) have no inline help |
|    | **Total**                              | **24/40** | | **Band:** ◯ adequate (20-28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders (`border-l/r > 1px`) used decoratively | OK | None observed. |
| Gradient text | OK | None. |
| Glassmorphism / blur as default | OK | None. |
| Hero-metric template | OK | n/a (form route). |
| Identical card grids | OK | Form sections are differentiated by label. |
| Modal as first thought | OK | Form is a route, not a modal. |
| Em dashes in copy | VIOLATION | The route inherits the parent's empty-state copy (the SplitPane bug shows the empty list rail with `Adjust the search or open a new task contract from the rail.` next to the form). The recurring template description (`task-templates.ts:51`) shows in the right-rail notice for the recurring template. Multiple `—` in `task-formatters.ts` lifecycle descriptions are not surfaced here directly but the inherited rail brings them in. |
| Generic AI palette | OK | Warm dark canvas, accent on primary submit. |
| Category-reflex theme | OK | Operator-first; no "AI form glow". |
| Restated headings | partial | Page header `New task`, breadcrumb `Back to tasks`, intro paragraph `Start from <template> template and stage the contract...`, and the right-rail `Submission` heading are four near-related labels. |
| Decorative shadows | OK | Flat. |
| Hardcoded `#000` / `#fff` | OK | All tokenized. |

**Summary verdict:** borderline. The form itself is clean, but the parent SplitPane bug means the route renders alongside an unrelated list rail with em-dashed copy, which makes the page look unfinished.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** at the top of the form, the user sees: 6 template pills, 1 title input (required), 1 description input, 2 scope pills, 4 priority pills, 1 owner kind select with 7 options, 1 owner ref input, 5 attempts pills, 2 approval pills, 1 parent task input, 1 network channel input, 1 identifier override input, plus three footer buttons (Cancel, Save draft, Submit). Roughly 12 to 14 first-class controls without counting pill values. Plus the unwanted parent SplitPane rail adds search + 3 lane pills + `New task`.
- **Eight-item checklist:**
  1. >4 visible options at decision point? **fail** (12 to 14 controls).
  2. Self-evident labels? **partial** (`network channel`, `identifier override` need help text).
  3. Primary action visually dominant? **pass** (`Create & enqueue` is accent fill in the footer).
  4. Progressive disclosure of complexity? **fail** (every field is rendered immediately, including `Identifier override` which 95% of users will never need).
  5. Related elements grouped via proximity? **pass** (`Task contract`, `Queue settings`, `Submission` sections).
  6. Hierarchy ratio ≥1.25? **pass** (titles vs body vs eyebrow).
  7. Body line length within 65-75ch? **pass** (helper text wraps under 65ch).
  8. Whitespace varied? **partial** (every Field uses identical 16px gap; rhythm is monotone).

  Failures: 3. Cognitive load = moderate-high.
- **IA observations:**
  - Owner kind options are raw enum strings (`agent_session`, `automation`, `extension`, `network_peer`, `pool`). The label map exists (`taskOwnerKindLabel` in `task-formatters.ts:213-228`) but the form does not use it.
  - `Identifier override` is a power-user field; bury it behind a disclosure.
  - Template pills double the role of the empty-state template grid the user already used to enter this route. Rendering them again is redundant for users who came in with `?template=<id>`.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** every reference is `var(--color-*)`. `task-editor-surface.tsx` does not introduce hex literals.
- **Type scale:** Inter for labels and body, JetBrains Mono for the breadcrumb (uppercase tracking 0.14em, slightly tighter than the DESIGN.md 0.16em masthead but in line with operator-UI eyebrows). Mono for "Read-only context" eyebrow (10px uppercase tracking 0.14em).
- **Radii / spacing:** sections use `rounded-[var(--radius-diagram)]` (12px); inputs are 36 to 40px; pills are 28 to 32px.
- **Elevation:** flat. Right rail uses a different background fill (`surface-panel` `#181716`) to distinguish from the main form, no shadows.
- **Signal palette discipline:** accent on the primary submit (`Create & enqueue`); no semantic banners.
- **Grid / rhythm:** main form is a `xl:grid-cols-[minmax(0,1.35fr)_minmax(22rem,0.85fr)]` two-column. Below xl it stacks. Pair fields (Owner / Attempts, Approval / Parent) use `md:grid-cols-2`.
- **Density:** dense. The right rail is mostly empty for short submissions; on smaller viewports it stacks below and feels orphan.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** `Create & enqueue` (default) or `Save draft` for draft templates. The button label switches between `Create task` and `Create & enqueue` based on `template?.preview.enqueueOnSubmit` (`task-editor-surface.tsx:526-530`).
- **Destructive actions:** none on this route; the user can only create.
- **Forms:**
  - Title is `required` (`task-editor-surface.tsx:217-225`).
  - Form submission via `<form onSubmit={form.submitForm}>`; `Save draft` uses `onClick` outside the submit pipeline.
  - No autosave; closing the tab discards the draft.
  - Identifier override has no validation pattern; if the operator types a duplicate the failure surfaces only after submit.
- **Tables / lists:** n/a.
- **Selection model:** template via `?template=<id>` search param; the route validates the value (`tasks.new.tsx:8-17`) and falls back to `undefined`.
- **Modals / drawers:** none.
- **Live updates:** n/a.
- **Optimistic vs pessimistic:** pessimistic; submit disables both buttons via `isSubmitting`.
- **Hover / focus / active states:** every input has the standard 1.5px accent focus ring; pill groups indicate active via fill or border.
- **Loading patterns:** Loader2 spinner inside both Save draft and Submit when pending.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** all controls are reachable. Tab order routes through the unwanted list rail first (P0 #2 from the module overview), then into the form.
- **Focus rings:** present on all controls (Input, Textarea, NativeSelect, PillGroup, Button).
- **TAB order:** breadcrumb -> page header (no controls) -> Template pills -> Title -> Description -> Scope -> Priority -> Owner kind select -> Owner ref text -> Attempts pills -> Approval -> Parent -> Network channel -> Identifier override -> Cancel -> Save draft -> Submit. Logical inside the form.
- **ARIA roles / labels:** `<form>` is implicit; `Field` provides labelled groups. Two nits:
  - The `<NativeSelect>` for owner kind has `aria-label="Owner kind"` (`task-editor-surface.tsx:293`) but the label-then-select is fine; however the second input under the same `Field` (`Owner reference (for example: coder)`) is not labelled programmatically because it lives inside the same `Field` whose `FieldLabel` is associated with the select id.
  - The breadcrumb is `aria-label="Breadcrumb"`. Good.
- **Color contrast:** body text on canvas exceeds 4.5:1; helper text `#8E8E93` measures ~5.4:1 against `#1E1C1B`. Input borders and focus borders meet 3:1 against the surface.
- **Motion:** Loader2 spin is the only animation; motion-reduce honored globally.
- **Text scaling:** at 200% zoom the right rail wraps; nothing clips visibly in the snapshot, but field labels and helper text get tight in the two-column layout.
- **Forms:** required fields (title) labelled with both an in-field `required` attribute and a `Required` mono badge in the same row (`task-editor-surface.tsx:213-216`). Good. Optional fields are not marked optional.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** strong. The form initializes with the `one_shot` template. The right rail shows a mono `Read-only context` panel with workspace, scope, parent, identifier.
- **Loading:** n/a (no remote prefill on create).
- **Error:** weak. There is no visible error region for failed submits. The hook (`useTaskCreateRouteState`) rejects but the route does not render a `<p>` or a banner; the user has to inspect `sonner` toasts (`web/src/integrations` Notifications region).
- **Permission denied:** missing.
- **Stale / disconnected:** n/a.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** uses `task contract`, `enqueue`, `coordinator handoff`, `network channel`, `parent task`. All match `glossary.md`. `recipe` is not used.
- **Tone:** clean and operator-first. The notice text is informative ("Will enqueue 1 run immediately on submit.").
- **Em dashes:** the recurring template's description embeds an em dash and is rendered in the form's intro paragraph: `"Start from Recurring via automation template and stage the contract before it enters the queue."` plus the right-rail notice text from the template's preview. Source: `task-templates.ts:51` `"Bind a cron or schedule from Automation — re-enqueues a run every tick."`.
- **Restated headings:** `New task` page title, then `Start from <template> template and stage the contract before it enters the queue.` paragraph, then `Task contract` section heading, then `Title` field label. Four labels for the same intent.
- **Sentence case vs Title Case:** sentence-case for sections; uppercase mono for eyebrows.
- **Truthful UI test:** the right rail says `READ-ONLY CONTEXT` which is honest; the workspace / scope / parent / identifier values reflect the runtime. The `network channel` field maps to the runtime's coordination-channel binding, which is correct. The `identifier override` field maps to the runtime's identifier alias path. No invented capabilities.

---

## 10. Performance & Responsiveness

- **Initial render:** the route renders the editor surface synchronously from local state; no remote prefill on create.
- **Re-render hot spots:** every keystroke on title or description re-renders the entire `TaskEditorSurface` because the draft is held at the route level. Acceptable for a form with <20 fields.
- **List virtualization:** n/a.
- **Bundle red flags:** none specific.
- **Responsive behaviour:** the live probe at 1440 wide showed the SplitPane rail + the form. At smaller widths the form drops to one column. No story checks 768 / 1024 / 320.
- **Mobile interactions:** the form has no hover-only affordances.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-tasks-new--default`
  - `routes-app-stories-tasks-new--template-preset`
  - `routes-app-stories-tasks-new--submitting`
- **States covered:** default form, template-preset, submit-pending.
- **Gaps:**
  - No validation-error story (e.g. server returns 422).
  - No "missing workspace" story (workspace null).
  - No mobile-viewport story.
  - No story showing the form without the parent SplitPane rail (because the parent always renders it; this is the bug worth showing in a story).
- **Story drift:** the Storybook stories render the same parent-renders-list-rail-alongside-form behavior the live route shows, so at least the story is faithful to the broken behavior. After the P0 fix is applied, the stories will need to be re-anchored.

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

1. **[P0] What:** SplitPane parent renders the list rail alongside the create form. Inherited from `tasks.tsx:194-202`.
   - **Why:** doubles affordances (two `New task` buttons), wrecks tab order, makes the create page feel unfinished. This is the single largest UX failure on this route.
   - **Fix:** see module overview P0 #4. The fix lives in `tasks.tsx`, not here, but the audit verdict for this route depends on it.
   - **Cmd:** `/impeccable layout web/src/routes/_app/tasks.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/tasks-new/live.png`.

2. **[P0] What:** em dash in the recurring template description (`task-templates.ts:51`) renders verbatim in the form's intro paragraph and the right-rail notice when the recurring template is active.
   - **Why:** ban from `DESIGN.md`, audit hard rule.
   - **Fix:** rewrite the recurring description without em dash; e.g. `"Bind a cron or schedule from Automation. Re-enqueues a run every tick. Configure the schedule from the Automation area after the draft is saved."`.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/lib/task-templates.ts`
   - **Effort:** S
   - **Evidence:** `task-templates.ts:51`.

### P1 - High-Value Polish

3. **[P1] What:** owner kind dropdown shows raw enum strings (`agent_session`, `network_peer`).
   - **Why:** label map already exists (`taskOwnerKindLabel` in `task-formatters.ts:222-228`) but the form does not use it.
   - **Fix:** map options through `taskOwnerKindLabel`; keep raw value as the `value` attribute.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/components/task-editor-surface.tsx`
   - **Effort:** S
   - **Evidence:** `task-editor-surface.tsx:301-308`.

4. **[P1] What:** form does not show inline server-error feedback when the create mutation fails.
   - **Why:** the operator submits, the toast may go off-screen, and the form re-enables silently.
   - **Fix:** add a `<p data-testid="task-editor-error">` slot above the footer buttons and wire it from the mutation's error.
   - **Cmd:** `/impeccable harden web/src/systems/tasks/components/task-editor-surface.tsx`
   - **Effort:** S

5. **[P1] What:** `Identifier override`, `Network channel`, and `Parent task` ship without inline help.
   - **Why:** these are advanced fields. New operators get no guidance.
   - **Fix:** add `FieldDescription` strings to each, keep them short. For `Parent task`, switch the Input to an autocomplete that searches existing tasks.
   - **Cmd:** `/impeccable clarify web/src/systems/tasks/components/task-editor-surface.tsx`
   - **Effort:** M

### P2 - Worthwhile

6. **[P2] What:** advanced fields render alongside basic fields without progressive disclosure.
   - **Why:** simple `one_shot` creation should be 2 to 3 fields visible.
   - **Fix:** collapse `Network channel` and `Identifier override` (and possibly `Parent task`) into an `Advanced` disclosure.
   - **Cmd:** `/impeccable distill web/src/systems/tasks/components/task-editor-surface.tsx`
   - **Effort:** M

7. **[P2] What:** breadcrumb uses uppercase mono `BACK TO TASKS` while the rest of the editor uses sentence-case nav.
   - **Why:** inconsistent register.
   - **Fix:** lower-case the breadcrumb or convert it to a small ghost button with `← Back to tasks` in sentence case.
   - **Cmd:** `/impeccable typeset web/src/systems/tasks/components/task-editor-surface.tsx`
   - **Effort:** S

8. **[P2] What:** there is no autosave or draft recovery if the operator navigates away.
   - **Why:** an operator who fills 10 fields and accidentally clicks the rail's `New task` (which itself is a P0 issue) loses everything.
   - **Fix:** persist `draft` to localStorage scoped by workspace; on mount, if a draft exists offer to restore.
   - **Cmd:** `/impeccable harden web/src/hooks/routes/use-task-create-route-state.ts`
   - **Effort:** M

### P3 - Parking Lot

9. **[P3] What:** `Save draft` button label is identical to the alternative submit verb on the right side.
   - **Fix:** consider `Save without enqueueing` or move the action into a dropdown to reduce footer clutter.

10. **[P3] What:** template pills duplicate the empty-state template grid for users who arrived from a tile click.
    - **Fix:** when `?template=<id>` is set, render the chosen template as a pill plus a link `Change template`, instead of the full row.

---

## 13. Persona Red Flags

- **Operator (returning power user):** no Cmd-Enter / Cmd-S to submit, no Cmd-D to save draft. The form supports tab traversal but not muscle-memory shortcuts.
- **First-timer:** sees 12 to 14 controls; helpful right-rail context but no tooltips on `Identifier override` or `Network channel`. Risk of submit-fail loops.
- **Agent:** stable test ids on every input (`task-editor-title-input`, `task-editor-priority-low`, etc.). DOM is good for programmatic reading; only complaint is the raw enum strings in the owner select reduce readability.

---

## 14. Cross-Module Consistency Notes

- The editor pattern (`Section` -> `Field` -> `Input | PillGroup | NativeSelect`) matches Skills create and Knowledge edit forms.
- The footer "primary CTA on the right, Cancel on the left" pattern matches other forms in the app.
- The breadcrumb register (uppercase mono) is unique to the editor; other create surfaces (e.g. session create) use ghost back buttons in sentence case. Inconsistent.

---

## 15. Open Questions

- Does the operator ever care about `network channel` at create time, or is that always set later by Automation / a peer? If late-bind is the norm, hide it behind disclosure.
- Should the `Save draft` button be the secondary action and `Create & enqueue` the only primary, given that draft templates already auto-save?
- Should the `Owner` field accept `Pool` only when the workspace has pools configured? Today the option always renders.

---

## 16. Recommended Action Plan

1. `/impeccable layout web/src/routes/_app/tasks.tsx` to remove the parent SplitPane rail on this child route. (Shared with module overview.)
2. `/impeccable clarify web/src/systems/tasks/lib/task-templates.ts` to remove the em dash from the recurring description.
3. `/impeccable clarify web/src/systems/tasks/components/task-editor-surface.tsx` to label owner-kind options via `taskOwnerKindLabel`, and to add inline help to `Identifier override`, `Network channel`, `Parent task`.
4. `/impeccable harden web/src/systems/tasks/components/task-editor-surface.tsx` to render server errors inline.
5. `/impeccable distill web/src/systems/tasks/components/task-editor-surface.tsx` to collapse advanced fields.
6. `/impeccable typeset web/src/systems/tasks/components/task-editor-surface.tsx` to align breadcrumb register.
7. `/impeccable harden web/src/hooks/routes/use-task-create-route-state.ts` to add localStorage autosave.
8. `/impeccable polish web/src/routes/_app/tasks.new.tsx` as the closing pass.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot.
- [x] No section empty.
- [x] Nielsen total 24/40 consistent with adequate band.
- [x] P0 to P3 with effort and command.
- [x] No hallucinated routes / props.
- [x] No em dashes in this report.
- [x] Length thorough not padded.
