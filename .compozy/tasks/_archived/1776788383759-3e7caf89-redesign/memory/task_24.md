---
name: Task 24 — Rewrite Automation domain
description: Rewrite web/src/systems/automation/** + /automation route on @agh/ui primitives
type: project
---

# Task Memory: task_24.md

## Objective Snapshot

Shipped: pure visual rewrite of `web/src/systems/automation/**` and `web/src/routes/_app/automation.tsx` over `@agh/ui`. Data layer untouched.

## Important Decisions

- Deleted `automation-form-primitives.tsx` entirely — replaced by `@agh/ui` `Field`/`FieldLabel`/`FieldTitle`/`FieldContent`/`FieldDescription` + `Input` + `Textarea` + `Switch` + `Pills`.
- Enable toggle uses `@agh/ui` `Switch` (role="switch"); test assertions migrated from `getByRole("checkbox")` to `getByTestId("{job|trigger}-enabled-toggle")`.
- Dialog header (`DialogHeader`/`DialogTitle`/`DialogDescription`) owns the "Create job / Edit trigger" copy; forms are children and only render submit/cancel buttons — tests that asserted form-local heading text now assert against the submit button label instead.
- `CodeBlock` consumes prompts via `code={prompt} copyable={false} showPrompt={false}` — `@agh/ui` `CodeBlock` does NOT accept children.
- Job stats Metric row computes success rate as `completed / (completed+failed+canceled)`; shows `"—"` when zero terminal runs; tone ramps `success≥90 / default≥70 / warning`.
- Trigger hook Section renders event as `KindChip` (right-slot of Section) AND `MonoBadge` (inside body); filters render as a wrap of `key=value` `MonoBadge`s; webhook details drop the legacy Secret column (not on the `AutomationTrigger` OpenAPI type).
- `useAutomationPage.listPanelProps` now includes `errorMessage` + `isLoading` so the panel can render loading/error fallbacks without an outer wrapper.
- Stale baseline files `systems-automation-automationformprimitives--*.png` and old `routes-app-stories-automation--{default,empty,loading}` removed; 22 new darwin baselines committed covering the 7 required states.

## Learnings

- Storybook story `export const Error` shadows the global `Error` constructor inside the story render fn; rename to `ErrorState` (or use `new globalThis.Error`) to keep typecheck clean.
- `AutomationTrigger` (OpenAPI shape) does not include `webhook_secret` — only `CreateAutomationTriggerRequest` does. Detail panel should not render a Secret column.
- Pills emit `role="tab"` + `aria-selected`; interaction-only test IDs should be asserted on the scope Pills buttons via `aria-selected="true"` instead of reading item-specific testids.
- `Field` accepts `orientation="horizontal"` which pairs a `Switch` + `FieldContent` title/description inline nicely.

## Files / Surfaces

Rewritten:
- `web/src/routes/_app/automation.tsx`
- `web/src/systems/automation/components/automation-list-panel.tsx`
- `web/src/systems/automation/components/automation-detail-panel.tsx`
- `web/src/systems/automation/components/automation-run-history.tsx`
- `web/src/systems/automation/components/automation-editor-dialog.tsx`
- `web/src/systems/automation/components/automation-job-form.tsx`
- `web/src/systems/automation/components/automation-trigger-form.tsx`
- `web/src/systems/automation/components/stories/automation-list-panel.stories.tsx`
- `web/src/systems/automation/components/stories/automation-detail-panel.stories.tsx`
- `web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx`
- `web/src/systems/automation/components/stories/automation-run-history.stories.tsx`
- `web/src/routes/_app/stories/-automation.stories.tsx`

Deleted:
- `web/src/systems/automation/components/automation-form-primitives.tsx`
- `web/src/systems/automation/components/stories/automation-form-primitives.stories.tsx`
- Stale Playwright baselines for form-primitives + old route story names.

Added:
- `web/src/systems/automation/components/automation-run-history.test.tsx` (new file, 4 tests)
- `web/src/systems/automation/components/automation-editor-dialog.test.tsx` (new file, 4 tests)

Modified:
- `web/src/hooks/routes/use-automation-page.ts` — adds `errorMessage` + `isLoading` to `listPanelProps`.

## Errors / Corrections

- First pass on run-history session link used "View session" — existing detail panel test expected exact string "View Session"; renamed to match.
- CodeBlock was mounted with `{prompt}` as children; threw `Cannot read properties of undefined (reading 'split')` — switched to `code={prompt}`.
- `webhook_secret` on AutomationTrigger caused typecheck failure; dropped the Secret field in the detail panel.

## Ready for Next Run

Automation domain is visually complete. Follow-up tasks (25 Bridges / 26 Knowledge / 27 Skills / 28 Workspace onboarding / 29 Daemon dashboard) can reuse the same shell pattern: `PageHeader` + optional `Pills` controls + `SplitPane` + Section-driven detail + `Field`-composed dialogs.
