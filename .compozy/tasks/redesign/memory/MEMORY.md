# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Phase 1 in progress. Tasks 01 + 02 + 03 + 04 landed: tokens/motion/UIProvider, Dialog/Popover/Sheet/Tooltip (motion), Combobox/Command/Select/ScrollArea/Tabs, plus DropdownMenu/Switch/Toggle/ToggleGroup/Accordion/Collapsible in `@agh/ui` (CSS animations kept — see decisions below).

## Shared Decisions

- **Package manager is Bun.** Treat all `pnpm --filter <pkg> <cmd>` references in task docs as `bun run --cwd <pkg> <cmd>`. Install with `bun add` (never edit `package.json` by hand).
- **`packages/ui` now has its own Vitest harness** (`packages/ui/vitest.config.ts` + `src/test-setup.ts`, included in root `vitest.config.ts` `projects`). New primitives should land with colocated `*.test.tsx` files rather than spinning up separate configs. The setup file also mocks `Element.prototype.scrollIntoView` for cmdk under jsdom.
- **Motion consumer hooks**: use `useReducedMotionConfig()` (context-aware) when asserting against a `MotionConfig` wrapper. Plain `useReducedMotion()` only reads `matchMedia` and ignores the provider.
- **Base UI + motion integration template**: for any Base UI primitive with a Portal/Popup lifecycle, wrap the Root in a controlled state component that passes `actionsRef={useRef()}` to Base UI and exposes `{ actionsRef, open }` via context. The Content renders `<AnimatePresence onExitComplete={() => actionsRef.current?.unmount()}>` around `{open && <Portal keepMounted>…</Portal>}`, and wires Backdrop/Popup via `render={<motion.div initial/animate/exit/>}`. This is the sanctioned pattern for motion-animated popups — do not re-introduce `data-open:animate-*` CSS keyframes alongside motion, it double-animates.
- **Select + Combobox kept CSS animations, not motion.** `SelectPortal` does not expose `keepMounted`, and Select's `alignItemWithTrigger` already suppresses the CSS animation; converting them to motion requires a deeper rewrite. They stay on `tw-animate-css` until a future task explicitly migrates them.
- **Base UI's `Input` is `Field.Control`.** Do not use it inside a `ComboboxPrimitive.Input render={}` — `Field.Control` intercepts the input state and the combobox stops receiving typed values in tests. Use a raw `<input>` with the primitive's classes instead.
- **Base UI Combobox input filtering under jsdom.** `userEvent.type` does not fire the input event the combobox listens to; use `fireEvent.change(input, { target: { value } })` to drive filter tests.
- **`web/src/lib/utils.ts` only re-exports `cn` from `@agh/ui`.** When a vitest test mocks `@agh/ui`, the factory must either (a) include a working `cn`, or (b) use `vi.mock("@agh/ui", async importActual => ({ ...(await importActual()), … }))`. Replacing the whole module leaves `cn` undefined and any downstream component that calls `cn(...)` crashes at render.
- **Base UI Menu group parts require `<Menu.Group>`.** `DropdownMenuLabel` (MenuGroupLabel) throws `MenuGroupRootContext is missing` if not wrapped in `DropdownMenuGroup`. Same rule applies to any group-scoped part.
- **Base UI's Accordion and ToggleGroup use `multiple` (boolean), not Radix's `type="single"/"multiple"`.** Tests and docs written against Radix conventions will not compile or will silently no-op.

## Shared Learnings

- `motion@12.38.0` is a thin re-export of `framer-motion`; querying `useReducedMotion` hits `window.matchMedia("(prefers-reduced-motion)")` (no `: reduce` value). Any jsdom setup intended to simulate "reduce" must match that exact query shape.

## Open Risks

- `make verify` cannot pass end-to-end on the base branch: pre-existing Go lint issues (`internal/observe/tasks.go` gosec, `internal/store/globaldb/global_db_task_aux.go` gocyclo) and ~6 flaky/broken test files in `web/`, `packages/site/`, and `sdk/create-extension/`. Task 01 did not introduce these; future tasks that rely on `make verify` will need them resolved or scoped around.

## Handoffs

- Task 14 will consume `UIProvider` from `@agh/ui` and wrap the app in `web/src/main.tsx`. `reducedMotion="user"` is the default — override to `"always"` only for tests/demos.
