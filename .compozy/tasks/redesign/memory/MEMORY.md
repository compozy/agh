# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Phase 1 kickoff. Task 01 landed the token foundation, `motion` dependency, `UIProvider`, and a Vitest harness in `packages/ui/`.

## Shared Decisions

- **Package manager is Bun.** Treat all `pnpm --filter <pkg> <cmd>` references in task docs as `bun run --cwd <pkg> <cmd>`. Install with `bun add` (never edit `package.json` by hand).
- **`packages/ui` now has its own Vitest harness** (`packages/ui/vitest.config.ts` + `src/test-setup.ts`, included in root `vitest.config.ts` `projects`). New primitives should land with colocated `*.test.tsx` files rather than spinning up separate configs.
- **Motion consumer hooks**: use `useReducedMotionConfig()` (context-aware) when asserting against a `MotionConfig` wrapper. Plain `useReducedMotion()` only reads `matchMedia` and ignores the provider.

## Shared Learnings

- `motion@12.38.0` is a thin re-export of `framer-motion`; querying `useReducedMotion` hits `window.matchMedia("(prefers-reduced-motion)")` (no `: reduce` value). Any jsdom setup intended to simulate "reduce" must match that exact query shape.

## Open Risks

- `make verify` cannot pass end-to-end on the base branch: pre-existing Go lint issues (`internal/observe/tasks.go` gosec, `internal/store/globaldb/global_db_task_aux.go` gocyclo) and ~6 flaky/broken test files in `web/`, `packages/site/`, and `sdk/create-extension/`. Task 01 did not introduce these; future tasks that rely on `make verify` will need them resolved or scoped around.

## Handoffs

- Task 14 will consume `UIProvider` from `@agh/ui` and wrap the app in `web/src/main.tsx`. `reducedMotion="user"` is the default — override to `"always"` only for tests/demos.
