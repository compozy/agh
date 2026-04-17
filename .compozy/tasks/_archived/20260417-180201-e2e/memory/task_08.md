# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add a reusable Playwright harness under `web/e2e/` that launches or attaches to a real AGH daemon, targets daemon-served embedded assets, captures stable browser diagnostics, and proves shell reachability with smoke coverage.

## Important Decisions

- The browser lane stays in the `web` workspace and boots the shipped product surface by running `bun run build` before `go build ./cmd/agh`; the daemon-served UI is only correct once the embedded web bundle is compiled into the binary.
- Runtime selection is environment-driven: default to launching an isolated daemon, but attach to an existing daemon when `AGH_E2E_BASE_URL` is provided. Attach mode rejects non-root paths and validates the fetched HTML is not a Vite development surface.
- Browser diagnostics reuse the task_01 artifact contract by writing `browser_trace.zip`, `browser_screenshots/`, `browser_console.json`, and `browser_network.json` into a manifest-driven artifact root.

## Learnings

- `vitest` `setupFiles` also run for `@vitest-environment node` tests in this workspace, so browser-only globals in `web/src/test-setup.ts` must be guarded behind `typeof window !== "undefined"`.
- Playwright fixtures require the first fixture callback parameter to keep the object-destructuring shape even when the values are unused.
- Context-level browser event listeners are enough to capture stable console and network diagnostics without route-specific test code.

## Files / Surfaces

- `web/package.json`
- `web/playwright.config.ts`
- `web/e2e/fixtures/artifacts.ts`
- `web/e2e/fixtures/browser-artifact-session.ts`
- `web/e2e/fixtures/runtime-helpers.ts`
- `web/e2e/fixtures/runtime.ts`
- `web/e2e/fixtures/test.ts`
- `web/e2e/fixtures/artifacts.test.ts`
- `web/e2e/fixtures/runtime.test.ts`
- `web/e2e/harness-smoke.spec.ts`
- `web/tsconfig.json`
- `web/vitest.config.ts`
- `web/src/test-setup.ts`

## Errors / Corrections

- Fixed the initial Playwright fixture callback signature so `test:e2e` no longer fails before running specs.
- Fixed node-environment unit tests by guarding jsdom-only setup in `web/src/test-setup.ts`.

## Ready for Next Run

- Later browser workflow tasks should import `test` from `web/e2e/fixtures/test.ts`, use `appPage` as the shell entrypoint, and rely on auto-persisted browser artifacts plus `runtime.resolveWorkspace(...)` for daemon-backed setup.
