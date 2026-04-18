# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Phase 1 in progress. Tasks 01 + 02 landed: tokens/motion/UIProvider foundation, plus Dialog/Popover/Sheet/Tooltip migrated into `@agh/ui` with motion-driven exit animations via Base UI's `actionsRef.unmount` + `AnimatePresence`.

## Shared Decisions

- **Package manager is Bun.** Treat all `pnpm --filter <pkg> <cmd>` references in task docs as `bun run --cwd <pkg> <cmd>`. Install with `bun add` (never edit `package.json` by hand).
- **`packages/ui` now has its own Vitest harness** (`packages/ui/vitest.config.ts` + `src/test-setup.ts`, included in root `vitest.config.ts` `projects`). New primitives should land with colocated `*.test.tsx` files rather than spinning up separate configs.
- **Motion consumer hooks**: use `useReducedMotionConfig()` (context-aware) when asserting against a `MotionConfig` wrapper. Plain `useReducedMotion()` only reads `matchMedia` and ignores the provider.
- **Base UI + motion integration template**: for any Base UI primitive with a Portal/Popup lifecycle (Dialog, Popover, Sheet, Tooltip, and coming Select/Combobox), wrap the Root in a controlled state component that passes `actionsRef={useRef()}` to Base UI and exposes `{ actionsRef, open }` via context. The Content renders `<AnimatePresence onExitComplete={() => actionsRef.current?.unmount()}>` around `{open && <Portal keepMounted>…</Portal>}`, and wires Backdrop/Popup via `render={<motion.div initial/animate/exit/>}`. This is the only sanctioned pattern in this repo — do not re-introduce `data-open:animate-*` CSS keyframes alongside motion, it double-animates.

## Shared Learnings

- `motion@12.38.0` is a thin re-export of `framer-motion`; querying `useReducedMotion` hits `window.matchMedia("(prefers-reduced-motion)")` (no `: reduce` value). Any jsdom setup intended to simulate "reduce" must match that exact query shape.

## Open Risks

- `make verify` cannot pass end-to-end on the base branch: pre-existing Go lint issues (`internal/observe/tasks.go` gosec, `internal/store/globaldb/global_db_task_aux.go` gocyclo) and ~6 flaky/broken test files in `web/`, `packages/site/`, and `sdk/create-extension/`. Task 01 did not introduce these; future tasks that rely on `make verify` will need them resolved or scoped around.

## Handoffs

- Task 14 will consume `UIProvider` from `@agh/ui` and wrap the app in `web/src/main.tsx`. `reducedMotion="user"` is the default — override to `"always"` only for tests/demos.
