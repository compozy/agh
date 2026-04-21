# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Landed Playwright visual harness for `web/` mirroring the `packages/ui` pattern: static Storybook bundle served on `127.0.0.1:6008`, per-platform baselines at `web/tests/visual/__snapshots__/`, 171 darwin PNGs committed (matches the 171 snapshottable stories after excluding 20 `play-fn` stories from the 191 total). Reused `collectVisualTargets` via the new `@agh/ui/testing/visual` subpath export.

## Important Decisions

- **Snapshot target = `page` viewport, not `#storybook-root`.** Web stories that render a Base UI Dialog with `open` default (e.g., `bridge-create-dialog`, `automation-editor-dialog`) mark the root as `aria-hidden` + `data-base-ui-inert`, which fails `locator.waitFor({state: "visible"})` and also renders the dialog body through a portal outside `#storybook-root`. Snapshotting the viewport captures portals and dodges the visibility trap. `packages/ui` keeps root-scoped screenshots because its Dialog stories render closed by default.
- **Separate Playwright config (`web/playwright.visual.config.ts`).** `web/playwright.config.ts` still drives the daemon-backed `web/e2e/` lane. Visual runs pass `--config playwright.visual.config.ts` explicitly so the two never collide.
- **Port 6008** for the web storybook static server (distinct from 6007 used by `packages/ui`) so both visual suites can run locally in parallel.
- **Route enumeration helper is new** (`web/tests/visual/route-enumeration.ts`). Unit-tests the file-based route tree directly from disk — flat-routes split on `.`, `-prefixed` files/folders are excluded, `_app` pathless layouts descend without contributing segments. Kept as a separate helper rather than reusing `routeTree.gen.ts` because the helper must work without the TanStack Router runtime.
- **Added `/design-system` showcase story** at `web/src/components/stories/design-system-showcase.stories.tsx` so the route is covered by the static-Storybook harness even though the route lives outside `_app/` and therefore is not picked up by the `_app/stories/` route-story convention.

## Learnings

- `vitest.config.ts`'s `include` at `web/` now matches `tests/**/*.test.{ts,tsx}` in addition to `src/**` + `e2e/**`. Explicit `exclude: ["tests/visual/*.spec.ts"]` keeps the Playwright specs out of vitest (they share the `.spec.ts` extension with vitest specs).
- `packages/ui` subpath exports now include `./testing/visual` → `./src/testing/visual-story-index.ts`. The helper lives in `src/` (not `tests/`) so the subpath export doesn't leak the tests tree. Existing `packages/ui/tests/visual/` files import directly from `../../src/testing/visual-story-index`.
- Full-page viewport snapshots are ~40KB each on darwin; 171 baselines ≈ 7MB on disk. Acceptable for the repo (well under the LFS threshold).

## Files / Surfaces

- `packages/ui/src/testing/visual-story-index.ts` — new shared helper.
- `packages/ui/package.json` — added `./testing/visual` subpath export.
- `packages/ui/tests/visual/stories.spec.ts`, `story-index.test.ts` — imports redirected to the new src location.
- `packages/ui/README.md` — added the "Web-side harness (`web/`)" subsection under "Playwright snapshot workflow"; `tests/readme.test.ts` heading-contract snapshot updated.
- `web/playwright.visual.config.ts` — new visual Playwright config (port 6008, viewport 1280x800, reducedMotion "reduce", colorScheme "dark", maxDiffPixelRatio 0.001).
- `web/scripts/serve-storybook.ts` — new Bun-native static server (copy of the packages/ui flavor with `AGH_WEB_STORYBOOK_PORT` env).
- `web/tests/visual/stories.spec.ts` — Playwright spec iterating the built storybook index via `@agh/ui/testing/visual`.
- `web/tests/visual/route-enumeration.ts` + `.test.ts` — route + story-file enumerator with 11 vitest cases.
- `web/tests/visual/playwright-config.test.ts` — 9 vitest cases asserting the visual config invariants.
- `web/tests/visual/__snapshots__/*.png` — 171 baselines (darwin), one per non-`play-fn` story.
- `web/src/components/stories/design-system-showcase.stories.tsx` — new story so the `/design-system` route shell is captured in the visual suite.
- `web/package.json` — added `build:visual`, `test:visual`, `test:visual:update`, `test:visual:install` scripts.
- `web/vitest.config.ts` — includes `tests/**/*.test.{ts,tsx}`, excludes `tests/visual/*.spec.ts`.
- `.github/workflows/ci.yml` — new `web-visual` job on `ubuntu-22.04`, path filter for `web/**` + `packages/ui/**`, uploads `.tmp/playwright-visual/*` on failure.

## Errors / Corrections

- **Dialog stories failed at the first `--update-snapshots` run** (6 timeouts) because `#storybook-root` becomes inert when the dialog opens. Corrected by switching `expect(root).toHaveScreenshot` to `expect(page).toHaveScreenshot` and changing `waitFor({state: "visible"})` → `waitFor({state: "attached"})`. Re-run succeeded cleanly (171/171).

## Ready for Next Run

- Linux baselines are NOT committed. First `web-visual` CI run must be dispatched with `--update-snapshots` (workflow_dispatch PR) before the visual gate becomes a merge blocker — same pattern as the `ui-visual` job documented in `packages/ui/README.md`.
- Phase 3–6 domain tasks: baselines WILL drift when a domain's `<domain>.stories.tsx` files are rewritten; commit the updated `web/tests/visual/__snapshots__/*.png` in the same PR as the redesign diff, with before/after thumbnails in the PR body.
