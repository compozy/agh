# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Shell route wired at `/_app/settings` with a default index child and in-shell section nav covering all 10 Paper sections so later tasks (09–14) can drop section pages under it.

## Important Decisions
- Section metadata (`SETTINGS_SECTIONS`) lives next to the shell in `web/src/routes/_app/settings.tsx` to avoid encroaching on task_09's `web/src/systems/settings` scaffold; task_09 is free to move it without refactoring the shell.
- Sidebar Settings control became a TanStack `Link` with `fuzzy: true` match so the active rail indicator stays on for any `/settings/*` child route.

## Learnings
- `routeTree.gen.ts` stores route ids on `route.options.id` at runtime, not on the node root; navigation assertions must walk `children` and read `.options.id` rather than relying on the TypeScript-only `.types` metadata.
- `routeTree.gen.ts` regenerates automatically when vite runs — `bunx vite build` or `vite dev` is enough; vitest alone does not trigger it, so frontend tasks that add routes must run a vite command before committing.

## Files / Surfaces
- `web/src/components/app-sidebar.tsx` — Settings footer button replaced with `SettingsNavItem` Link
- `web/src/components/app-sidebar.test.tsx` — added Settings nav link/active indicator coverage
- `web/src/routes/_app/settings.tsx` — new shell with section nav
- `web/src/routes/_app/settings/index.tsx` — default child placeholder
- `web/src/routes/_app/-settings.test.tsx` — shell + nav tests
- `web/src/routes/_app/settings/-index.test.tsx` — index placeholder test
- `web/src/routes/-settings-route-tree.test.ts` — asserts routeTree subtree shape
- `web/src/routeTree.gen.ts` — regenerated

## Errors / Corrections
- First routeTree assertion attempted to read ids off the node root and off the compile-time `.types` metadata; neither exists at runtime. Fix: traverse `children` and read `.options.id` per node.

## Ready for Next Run
- task_09 can safely import `SETTINGS_SECTIONS` from `@/routes/_app/settings` or move it into `systems/settings/lib`; the shell does not depend on any `systems/settings` module yet.
- Per-section routes (task_10+) only need to add `web/src/routes/_app/settings/<slug>.tsx`; the shell, navigation highlight, and shared layout already render around them.
