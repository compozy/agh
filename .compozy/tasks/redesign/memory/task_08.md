# Task Memory: task_08.md

## Objective Snapshot

Close `web/src/components/ui/` — relocated Avatar, Breadcrumb, ButtonGroup, Field, InputGroup, Item, NativeSelect, Textarea, Sonner, Direction into `packages/ui/src/components/`, rewrote importers to `@agh/ui`, deleted the folder.

## Important Decisions

- **Sonner decoupled from `next-themes`.** New `Toaster` in `@agh/ui` accepts `theme` prop (defaults `"system"`) — Sonner's own `prefers-color-scheme` watcher now owns theme tracking. Avoids adding `next-themes` as a `@agh/ui` peer dep. Web's `__root.tsx` drops the wrapper and mounts `<Toaster />` directly.
- **`@agh/ui` now re-exports `toast`** from `sonner` alongside `Toaster` so consumers only need one import.
- **`tsconfig.json` path aliases unchanged.** Only `@/*` → `./src/*` exists; no `@/components/ui/*` alias ever existed, nothing to remove.

## Learnings

- `web/src/routes/-__root.test.tsx` mocked `@/components/ui/sonner` directly. After the move, replace the separate mock with a `Toaster` override inside the existing `@agh/ui` mock (`importActual` + spread) — avoids accidentally blowing away `cn`/other exports the component needs.

## Files / Surfaces

- Moved: `packages/ui/src/components/{avatar,breadcrumb,button-group,field,input-group,item,native-select,textarea,sonner,direction}.tsx` + co-located `*.test.tsx` + `stories/*.stories.tsx`.
- Rewrote importers: `web/src/routes/__root.tsx`, `web/src/routes/-__root.test.tsx`, `web/src/systems/tasks/components/{task-editor-surface,tasks-create-modal}.tsx`, `web/src/systems/bridges/components/{bridge-create,bridge-edit,bridge-test-delivery}-dialog.tsx`, `web/src/systems/workspace/components/workspace-selector.tsx`.
- `packages/ui/package.json`: added `sonner` runtime dep.

## Errors / Corrections

- None. `bun add next-themes` briefly added next-themes as a devDep, rolled back via `bun remove` after deciding the `theme` prop approach.

## Ready for Next Run

- Task 13 (app-sidebar rewrite) is unblocked — `web/src/components/ui/` is gone.
