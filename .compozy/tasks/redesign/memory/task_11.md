# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Wire Playwright visual harness for `packages/ui` storybook with 0.1% diff threshold and per-platform baselines. Done.

## Important Decisions

- Drive Playwright off the **static Storybook build** (`storybook build --output-dir .tmp/storybook-static`), served by a tiny Bun HTTP script (`scripts/serve-storybook.ts`). Dev-server churn is non-deterministic; a static bundle is hermetic and runs the same in CI and locally.
- `test:visual` chains `build:visual` then `playwright test`. Running Playwright directly without the build prints an actionable error that points at `build:visual`.
- Spec enumerates stories at **module load time** (`readFileSync(.tmp/storybook-static/index.json)`) — Playwright cannot add tests inside `beforeAll`, so the index must exist before the suite is loaded. `build:visual` guarantees that.
- Snapshots filtered to `type: "story"` entries; `play-fn` tagged stories are excluded because their play functions mutate state between capture and assertion.
- Snapshots live under `src/components/stories/__snapshots__/<id>-chromium-<platform>.png` via `snapshotPathTemplate`. Platform is baked into the filename so darwin (local dev) and linux (CI) baselines coexist without clobbering each other.
- CI job is opt-in via path filter `ui-visual`, runs on `ubuntu-22.04`, installs Playwright chromium explicitly, uploads diff report + test-results on failure. Not gated on Go `verify`.

## Learnings

- **Font race on first navigation per worker.** `page.goto(..., { waitUntil: "networkidle" })` returned before Inter Variable / JetBrains Mono actually rendered — first screenshot per worker captured with fallback metrics, later workers captured with Inter metrics, producing ~12% pixel drift on text-heavy stories (Button, Tooltip, Sheet, Dialog, DropdownMenu). Fix: switch `waitUntil` to `"load"`, then explicitly `document.fonts.load(...)` for every Inter + JetBrains Mono weight used in the design system, then await `document.fonts.ready`. Three consecutive clean runs after that.
- **Playwright requires the snapshot name to end in `.png`.** `toHaveScreenshot("foo")` throws `Screenshot name "foo-chromium-darwin" must have '.png' extension`. Keep the `.png` suffix on the argument; `{ext}` in `snapshotPathTemplate` is resolved from it.
- **`{arg}`/`{ext}` interplay in `snapshotPathTemplate`.** With template `{snapshotDir}/{arg}-{projectName}-{platform}{ext}` and arg `ui-button--default.png`, Playwright writes to `…/ui-button--default-chromium-darwin.png`. No manual stripping needed.
- **`page.evaluate(async () => document.fonts.ready)` must actually `await` the Promise inside the page.** Returning the unfulfilled promise to the driver is a no-op; explicit `await document.fonts.ready` (and an explicit `document.fonts.load(...)` before it) is required.
- **Playwright's CJS/ESM types on `webServer`.** `playwright.config.ts`'s `webServer` is either a single object or an array. The unit test that asserts config shape normalizes via `Array.isArray(cfg.webServer) ? cfg.webServer[0] : cfg.webServer` to stay forward-compatible with multi-server configs.

## Files / Surfaces

- New: `packages/ui/playwright.config.ts`, `packages/ui/scripts/serve-storybook.ts`, `packages/ui/tests/visual/story-index.ts`, `packages/ui/tests/visual/story-index.test.ts`, `packages/ui/tests/visual/playwright-config.test.ts`, `packages/ui/tests/visual/stories.spec.ts`, `packages/ui/.gitignore`, `packages/ui/src/components/stories/__snapshots__/*-chromium-darwin.png` (168 baselines).
- Modified: `packages/ui/package.json` (added `@playwright/test@1.59.1` devDep + `build:visual`/`test:visual`/`test:visual:update`/`test:visual:install` scripts), `packages/ui/vitest.config.ts` (include `tests/**/*.test.ts`, exclude `tests/visual/*.spec.ts`), `.github/workflows/ci.yml` (new `ui-visual` job + `ui-visual` path filter on `changes`).
- Design references consulted (read-only): `DESIGN.md`, `docs/design/design-system/preview/*.html`.

## Errors / Corrections

- First regeneration pass (before the font fix) produced ~29 flaky baselines; rerunning the suite surfaced 622 px vs 550 px drift because the baseline capture happened while Inter was still in fallback. After adding `waitForFonts`, `rm -rf src/components/stories/__snapshots__/*.png && playwright test --update-snapshots` reseeded deterministic baselines.
- A stray root-level `test-results/.last-run.json` appeared when `bunx playwright test` was invoked from the repo root. Cleaned up; subsequent runs stay inside `packages/ui/.tmp/playwright/`.

## Ready for Next Run

- **Linux baselines are NOT committed yet.** The `-chromium-darwin.png` set is the source of truth for local dev on macOS; the first CI run on `ubuntu-22.04` will fail with "missing snapshot" for every `-chromium-linux.png`. Follow-up: trigger the `ui-visual` job with `--update-snapshots` once (either via a `workflow_dispatch` variant or a temporary branch) and commit the linux baselines. ADR-005 sanctions this split.
- Task 12 (`packages/ui/README.md`) must document: `bun run test:visual` local gate, `bun run test:visual:update` for intentional drift, and the ubuntu-22.04 baseline bootstrap note above.
- Task 16 (web visual baseline) reuses the same `story-index.ts` patterns and `serve-storybook.ts` approach — the helpers are small enough to duplicate rather than package-extract.
