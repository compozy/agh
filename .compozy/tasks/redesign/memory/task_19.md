# Task Memory: task_19.md

## Objective Snapshot

Close Phase 3 Tasks domain rewrite: tasks-create-modal, task-editor-surface, task-run-detail-header, task-run-detail-panels (Identity/Progress/Activity), and task-run-detail-session-link rewritten onto `@agh/ui` primitives (`Dialog`, `Field`, `Input`, `Textarea`, `NativeSelect`, `Pills`, `PageHeader`, `Section`, `Metric`, `CodeBlock`, `MonoBadge`, `StatusDot`, `Table`) while preserving the existing hook signatures, validation, and route loaders.

## Important Decisions

- Reused `Pills` (segmented tablist) for every toggle-style field in the modal + editor (Template / Scope / Priority / Attempts / Approval). It preserves the existing `data-testid` + `aria-pressed` contract that the Vitest tests rely on.
- Owner Kind dropdown → `@agh/ui` `NativeSelect` (Base UI Select was ruled out in shared memory because `Field.Control` intercepts state).
- PageHeader ownership moved to both editor + run-detail header; the breadcrumb sits OUTSIDE the PageHeader in both surfaces to keep the existing `task-run-detail-breadcrumb` / `task-editor-back-link` selectors intact.
- Run-detail title text is `Run {id}` with the id rendered inside a `MonoBadge`; a JSX space node keeps `textContent` = "Run run_...".
- Run-detail `Events` / `Output` / `run.events` primitives mentioned in the spec test bullets were aspirational — the real `TaskRunDetailView` has `summary` + `result` + `error`. Kept the 3-panel shape (Identity / Progress / Activity) + new `Section` shells; result now renders via `CodeBlock language="json"` inside Activity.

## Learnings

- `toHaveTextContent` uses `textContent`, which is whitespace-normalized concatenation. To make "Run <MonoBadge>id</MonoBadge>" match "Run id", inject a JSX space text node (`{" "}`) between siblings — flex `gap-*` does not add a space to `textContent`.
- `Pills` testIds on individual items forward `data-testid` to the `<button role="tab">`, and `aria-pressed` flips on the active item. Tests written against the old custom buttons kept passing without changes.
- Section component spreads props (including `data-testid`) on the outer `<section>`, so `<Section data-testid="task-run-detail-progress" label="Progress">` keeps the existing selector contract while adding the mono eyebrow header.

## Files / Surfaces

- Rewritten components:
  - `web/src/systems/tasks/components/tasks-create-modal.tsx`
  - `web/src/systems/tasks/components/task-editor-surface.tsx`
  - `web/src/systems/tasks/components/task-run-detail-header.tsx`
  - `web/src/systems/tasks/components/task-run-detail-panels.tsx`
  - `web/src/systems/tasks/components/task-run-detail-session-link.tsx`
- New Storybook stories (empty / populated / pending / error / validation-error):
  - `web/src/systems/tasks/components/stories/tasks-create-modal.stories.tsx`
  - `web/src/systems/tasks/components/stories/task-editor-surface.stories.tsx`
  - `web/src/systems/tasks/components/stories/task-run-detail.stories.tsx`
- Playwright darwin baselines (15 new PNGs under `web/tests/visual/__snapshots__/`).

## Errors / Corrections

- First pass put `data-testid="task-run-detail-title"` on an inner span without the MonoBadge → textContent = "Run" only, failing the existing assertion. Re-nested the MonoBadge inside the testid span with `{" "}` to restore "Run run_..." concatenation.
- Format pass split the story imports (duplicate `../task-run-detail-panels` line); merged manually — oxfmt does not merge sibling named imports from the same module.

## Ready for Next Run

- Tasks domain (17 + 18 + 19) is visually complete; grep `@/components/(ui|design-system)/` in `web/src/systems/tasks/components/**/*.tsx` returns zero.
- Linux baselines for the 15 new run-detail / editor / modal snapshots still need to be generated in CI via the one-shot `--update-snapshots` pattern used for every previous web-visual batch.
- `make verify` fails at the documented pre-existing Go lint issues (see shared memory "Open Risks"). Frontend scope (lint + typecheck + vitest + vite build + playwright visual) all pass green.
