# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Replace `design-system/{pill,pill-button,page-content,panel,section-heading,toolbar,texture-canvas}` and `ui/empty` with `@agh/ui` primitives (`Pill`, `Pills`, `PageHeader`, `SearchInput`, `Empty`, `Section`, `Toolbar`). Rewrite every call site and delete the old files in the same PR.

## Important Decisions

- **Exposed two exports from `pills.tsx`**: `Pill` (static span, replaces the 30+ legacy `Pill` call sites) and `Pills` (segmented toggle group, replaces every `PillButton` group AND serves the mock's `Pills` primitive). Keeping them colocated avoids duplicating the variant/tint table.
- **`Pills` is a tablist**: renders buttons with `role="tab"` + `aria-selected` + `aria-pressed` + `data-active`. Any test previously checking `aria-checked` (for a radiogroup) must be updated — did this in `settings/-general.test.tsx` for the permissions pills.
- **`Empty`'s `icon` accepts both a Lucide component and a pre-rendered ReactNode.** The naive `typeof icon === "function"` check misses Lucide because its icons are `forwardRef`+`memo` objects (`{$$typeof, render}`). The detection now also returns true when the value is an object with a `render` key.
- **No `pillButtonVariants` / `pillVariants` export chain preserved.** The DESIGN.md-aligned variant + size tokens live inside `@agh/ui` `pills.tsx`; legacy `design-system` `pillVariants` callsite in `design-system-showcase.tsx` was removed by rewriting the showcase entirely (see below).
- **`design-system-showcase.tsx` trimmed to use only `@agh/ui` + surviving `metric-strip` + `status-dot`.** Task 15 will delete it; this trim keeps the `/design-system` route functional while satisfying the "no migrated-primitive imports" rule.
- **Local Panel shim in two task files.** `task-editor-surface.tsx` and `tasks-detail-preview-panel.tsx` previously used `Panel/PanelHeader/PanelTitle/PanelDescription/PanelBody`. Since the old `.ds-panel*` CSS classes were already stripped (see `styles.test.ts`), the legacy Panel was a styling no-op with only flex/gap. The two files now declare minimal local Panel* components (styled divs) rather than pulling a new cross-cutting primitive into `@agh/ui`. Task 15 (showcase delete) and future task work can remove these if a richer Card primitive is preferred.
- **Tone → variant mapping** lives in `web/src/lib/pill-variant.ts` (`pillVariantFromTone`). Maps `neutral|amber|green|violet|danger|accent|warning` → `default|accent|success|info|danger|accent|warning`. Domain helpers (`taskStatusTone`, `bridgeStatusTone`, etc.) still return the legacy string; the helper keeps the call-site noise minimal.

## Learnings

- Two separate single-import lines from `@agh/ui` get left by oxfmt (e.g. `import { Button } from "@agh/ui"` + `import { Pill } from "@agh/ui"`). Run a post-pass merge to collapse them — `make web-fmt` does not do this. Used a small Python script with a regex merge pass.
- `make verify` fails on two Go lint issues (`internal/observe/tasks.go:1749` gosec G202 and `internal/store/globaldb/global_db_task_aux.go:585` gocyclo) that pre-date this task. They are also recorded in shared MEMORY.md under Open Risks. Web-scope verification (`make web-lint`, `make web-typecheck`, `make web-build`, `make web-test`, `bun run --cwd packages/ui test`) is the appropriate gate for frontend-only redesign tasks until those Go issues are resolved.
- Lucide React icons are memoized forwardRef components, so `typeof Icon === "function"` is false at runtime. Any primitive that accepts either an icon component or a pre-rendered icon must detect both shapes.

## Files / Surfaces

- Added: `packages/ui/src/components/{pills,page-header,search-input,empty,section,toolbar}.tsx` + colocated tests/stories; `web/src/lib/pill-variant.ts`.
- Updated: `packages/ui/src/index.ts`.
- Rewritten: every `design-system/{pill,pill-button,page-content,panel,section-heading,toolbar}` and `ui/empty` importer under `web/src/**` (≈40 files across routes, `settings/`, `tasks/`, `bridges/`, `network/`, `automation/`, `workspace/`, `skill/`, `settings/`). `design-system-showcase.tsx` replaced with a minimal `@agh/ui`-only composition. `web/src/components/design-system/stories/story-frame.tsx` rewritten to drop its `TextureCanvas` dependency.
- Deleted: `web/src/components/design-system/{pill,pill-button,page-content,panel,section-heading,toolbar,texture-canvas}.tsx`, `web/src/components/ui/empty.tsx`, and the six stories that targeted those components (plus `web/src/components/ui/stories/empty.stories.tsx`).

## Errors / Corrections

- First pass of `Empty.icon` dispatched only on `typeof === "function"`. Rendered Lucide icons blew up React with "Objects are not valid as a React child". Fixed by accepting the forwardRef/memo object shape as a component too.
- Original rewrite in `web/src/routes/_app/tasks.tsx` dropped the `tasks-mode-inbox-unread` test id on the Pills badge. Updated the test to query `[data-slot="pills-badge"]` on the inbox pill instead — the new primitive renders the badge count automatically when `item.badge > 0`.
- `-general.test.tsx` was asserting `aria-checked="true"` (radio semantics) on the active permission pill. Switched assertion to `aria-selected="true"` because `Pills` is a tablist.

## Ready for Next Run

- Task 07 (add `Metric`, `MonoBadge`, `KindChip`, `StatusDot`, `ConnectionIndicator` to `@agh/ui`) can now land without bumping into `@/components/design-system/pill*`. It still has to migrate `metric-strip.tsx`, `status-dot.tsx`, `connection-indicator.tsx` and update the tone strings those files expose.
- Task 13 (app-sidebar rewrite) + task 14 (root layout + motion) can consume the new `PageHeader`, `Pills`, `SearchInput`, `Empty`, `Section`, `Toolbar` primitives immediately.
- Task 15 (showcase rewrite + folder delete) now only has to remove four files (`metric-strip`, `status-dot`, `connection-indicator`, `design-system-showcase`) plus the `stories/` folder and `index.ts`.
