# Task Memory: task_01.md

## Objective Snapshot

Phase-1 foundation: extend `@agh/ui` tokens with `--font-display` + `--font-wordmark`, install `motion`, export `UIProvider`, add the Storybook interaction story, set up a minimal Vitest harness in `packages/ui/`, and trim `web/src/styles.css`.

## Important Decisions

- Of the 10 tokens listed in the spec, 8 were already present in `packages/ui/src/tokens.css` (radii, motion, easing, accent-tint-strong). Only `--font-display` and `--font-wordmark` needed adding. Placed them inside `@theme inline` alongside the existing `--font-sans` / `--font-mono` so they expose as both `var(--font-display)` and a Tailwind `font-display` utility.
- Package manager is **Bun** (`bun add`), not pnpm. The task doc's `pnpm --filter ...` references were mapped to the Bun equivalents (`bun run --cwd ...`) per `web/CLAUDE.md`.
- Added a dedicated `packages/ui/vitest.config.ts` + `src/test-setup.ts` and wired it into the root `vitest.config.ts` `projects` list. This matches the techspec's explicit mention of `packages/ui/src/test-setup.ts` and enables `bun run --cwd packages/ui test`.
- Tests use `useReducedMotionConfig()` (not `useReducedMotion()`) to assert that `MotionConfig` forwards the `reducedMotion` prop — `useReducedMotion` ignores the context and only reads `matchMedia`.
- `web/src/styles.css` already only contained imports + essential globals (no primitive class definitions), so no trimming was necessary — documented rather than changed.

## Learnings

- `motion@12.38.0` ships as a wrapper over `framer-motion`; `motion/react` re-exports everything from `framer-motion/dist/es`. Both `useReducedMotion` and `useReducedMotionConfig` are available.
- `motion-dom`'s reduced-motion state reads `window.matchMedia("(prefers-reduced-motion)")` (no `: reduce` value). Test-setup mocks must match that query shape.
- Vitest + `useReducedMotionConfig` requires `waitFor` on first render — the underlying `useState` picks up `prefersReducedMotion.current` only after `initPrefersReducedMotion()` runs.
- Pre-existing `web/src/styles.test.ts` asserted `--font-display` was absent; updated to assert presence for `--font-display` + `--font-wordmark`.

## Files / Surfaces

- `packages/ui/src/tokens.css` — added `--font-display`, `--font-wordmark` in `@theme inline`.
- `packages/ui/src/components/ui-provider.tsx` — new `UIProvider` (MotionConfig wrapper, 15 LOC).
- `packages/ui/src/components/stories/ui-provider.stories.tsx` — Default + ReducedMotionAlways (play fn) + ReducedMotionNever (play fn).
- `packages/ui/src/components/ui-provider.test.tsx` — 4 unit tests covering default, always, never, and user-mode defaults.
- `packages/ui/src/index.ts` — new `UIProvider` export.
- `packages/ui/package.json` — motion peer dep + test/test:watch scripts + vitest/testing-library devDeps.
- `packages/ui/vitest.config.ts` — new config (jsdom, src/test-setup.ts).
- `packages/ui/src/test-setup.ts` — jsdom matchMedia + ResizeObserver mocks tuned for motion.
- `vitest.config.ts` (root) — added `packages/ui/vitest.config.ts` to `projects`.
- `web/package.json` — motion runtime dep.
- `web/src/styles.test.ts` — swapped the `does NOT declare --font-display` assertion for presence checks on both new tokens.

## Errors / Corrections

- First test attempt used `useReducedMotion` and failed to see MotionConfig context. Switched to `useReducedMotionConfig` (context-aware).
- matchMedia mock originally keyed on `"prefers-reduced-motion: reduce"`; motion-dom queries `"(prefers-reduced-motion)"` — loosened the check.

## Ready for Next Run

- `packages/ui` now has working Vitest harness; future primitives can drop `*.test.tsx` files alongside components.
- `UIProvider` is exported and ready for Task 14 (root layout wiring in `web/src/main.tsx`).
- Pre-existing failures in `make verify` (Go lint gocyclo/gosec in `internal/observe/tasks.go` + `internal/store/globaldb`, and 5 other test files) are unrelated to this task and blocked `make verify` end-to-end; verified via git stash that they pre-date this change.
