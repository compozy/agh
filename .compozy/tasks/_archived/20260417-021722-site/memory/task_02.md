# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Migrate web/ to consume @agh/ui for design tokens and 12 base components.

## Important Decisions

- Deleted local component files (zero-legacy-tolerance) instead of creating re-exports.
- `web/src/styles.css` kept only `@import "@agh/ui/tokens.css"`, `@import "shadcn/tailwind.css"`, and `#app { min-height: 100% }` — everything else is in tokens.css.
- `web/src/lib/utils.ts` re-exports `cn` from `@agh/ui` (no web-specific additions exist).

## Learnings

- Tests that mock `@agh/ui` must also include `cn` in the mock if the component under test imports `cn` from `@/lib/utils` (which now re-exports from `@agh/ui`).
- `styles.test.ts` reads CSS files with `readFileSync` — after token extraction, tests must read `packages/ui/src/tokens.css` for token assertions.

## Files / Surfaces

- `web/package.json` — added `@agh/ui: workspace:*`
- `web/src/styles.css` — replaced inline tokens with `@import "@agh/ui/tokens.css"`
- `web/src/lib/utils.ts` — re-export from `@agh/ui`
- `web/src/styles.test.ts` — updated to read tokens from `packages/ui/src/tokens.css`
- `web/src/components/app-sidebar.test.tsx` — added `cn` to `@agh/ui` mock
- 12 deleted files: button, badge, card, input, label, separator, skeleton, spinner, alert, progress, table, kbd from `web/src/components/ui/`
- ~36 files across web/src/ with updated import paths

## Errors / Corrections

- First test run: `styles.test.ts` failed (20 tests) because token regex matched against the now-slim `styles.css`. Fixed by reading `tokens.css` directly.
- First test run: `app-sidebar.test.tsx` failed (29 tests) because `cn` was undefined — the `@agh/ui` mock didn't include `cn`. Fixed by adding `cn` to the mock.

## Ready for Next Run

Task complete. All gates pass.
