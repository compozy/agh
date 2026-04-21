# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Bootstrap Storybook 10 in `packages/ui` and extend the existing web Storybook with MSW, story-scoped TanStack Query, and a memory-history router stub.
- Deliver the required validation evidence for both Storybook builds plus the local MSW startup check.

## Important Decisions

- `web/.storybook/main.ts` serves `../public` so the generated `mockServiceWorker.js` is reachable from Storybook.
- Web preview exports helper factories/decorator arrays so the Storybook bootstrap can be tested directly in Vitest without duplicating setup.
- `packages/ui` Storybook injects `@tailwindcss/vite` via `viteFinal`; the same plugin is enabled in `web/vitest.config.ts` so preview CSS imports resolve during tests.

## Learnings

- Baseline before edits:
- `packages/ui/.storybook/` was missing.
- `packages/ui/package.json` had no Storybook scripts or dependencies.
- `web/.storybook/preview.ts` contained only the theme decorator.
- `web/public/mockServiceWorker.js` was absent.
- `msw init web/public --save` also records the worker directory in the repo-root `package.json`, so that file is part of the implementation surface.
- A live Playwright smoke test against `components-design-system-panel--default` is stable enough to verify `[MSW]` registration and unhandled-request bypass behavior.

## Files / Surfaces

- `packages/ui/package.json`
- `packages/ui/.storybook/`
- `web/package.json`
- `web/.storybook/main.ts`
- `web/.storybook/preview.ts`
- `web/public/mockServiceWorker.js`
- `package.json`
- `web/vitest.config.ts`
- `web/src/test-setup.ts`
- `web/src/storybook/`
- `web/e2e/storybook-bootstrap.spec.ts`

## Errors / Corrections

- `packages/ui` Storybook initially emitted Tailwind at-rule warnings during `build-storybook`; adding `@tailwindcss/vite` to the Storybook Vite config fixed the build cleanly.

## Ready for Next Run

- Task 01 is complete. Downstream tasks can author stories against the new `packages/ui` Storybook workspace and the shared web preview providers without adding extra global setup.
