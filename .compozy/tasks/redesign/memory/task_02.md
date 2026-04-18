# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Completed. Moved `dialog`, `popover`, `sheet`, `tooltip` from `web/src/components/ui/` to `packages/ui/src/components/`, exported from `@agh/ui`, rewrote every web importer, and deleted the old files. Integrated motion via `AnimatePresence` + `actionsRef.unmount()` so Base UI's lifecycle plays cleanly with framer-motion exit animations. All 27 new unit tests pass, typecheck is clean, `make web-lint` is zero-warning, `make web-build` succeeds.

## Important Decisions

- **Motion + Base UI integration pattern**: Wrap the root in a thin controlled-state wrapper that creates a `useRef<*Root.Actions>()`, passes `actionsRef` to the Base UI Root, and exposes `{ actionsRef, open }` via context. The Content component uses `<AnimatePresence onExitComplete={() => actionsRef.current?.unmount()}>` around the conditional `open && <Portal>…`, and the Backdrop/Popup use `render={<motion.div initial/animate/exit transition/>}` so the UIProvider's `MotionConfig.reducedMotion` automatically suppresses transforms. This is the official "external animation library" pattern documented in `@base-ui/react/dialog` 1.4.0.
- **Popup keepMounted**: set `keepMounted` on the Portal so AnimatePresence controls the visibility, not Base UI's internal mount logic.
- **Sheet side animations**: variant map keyed by `side` with `x`/`y` translation. Motion drops translation automatically under reduced-motion.
- **Tooltip trigger hover/focus**: handled by Base UI natively; our wrapper only mirrors the open state. `delay` is still provider-level.
- **Test mocks in web**: updated `tool-call-card.test.tsx` and `-__root.test.tsx` to use `vi.mock("@agh/ui", async () => { const actual = await vi.importActual... })` so other primitives (Button, Input, etc.) continue to resolve to the real module while only the dialog/tooltip symbols are mocked.

## Learnings

- `DialogRoot.Props["onOpenChange"]` is `(open, eventDetails) => void` — must forward both args to preserve Base UI's reason metadata (for `outsidePress`, `escapeKey`, etc.).
- `DialogRoot.Props["children"]` is `ReactNode | PayloadChildRenderFunction`; when re-emitting children into our own Context.Provider we must narrow to `ReactNode` (we don't consume the payload variant in these primitives).
- Base UI's `*Primitive.Popup.Props` already includes `data-slot` — no need to pre-apply it in render-prop clones, though we still set `data-slot` on the host element for selector queries.
- `@testing-library/user-event` was not previously declared in `packages/ui`; added via `bun add -d` during this task.
- The pre-existing `tasks-empty-state.test.tsx` failures are unrelated to this task (confirmed by stashing the diff and re-running) and are already called out in shared memory.

## Files / Surfaces

- New: `packages/ui/src/components/{dialog,popover,sheet,tooltip}.tsx`, `{dialog,popover,sheet,tooltip}.test.tsx`, `stories/{dialog,popover,sheet,tooltip}.stories.tsx`.
- Modified public surface: `packages/ui/src/index.ts` (four new export groups).
- Modified web importers (16): routes/__root.tsx + test; components/ui/sidebar.tsx + command.tsx; systems/{tasks,settings,workspace,network,bridges,automation,session}/components/*.
- Deleted: `web/src/components/ui/{dialog,popover,sheet,tooltip}.tsx` and `web/src/components/ui/stories/{dialog,popover,sheet,tooltip}.stories.tsx`.
- Added dev dep: `@testing-library/user-event@14.6.1` in `packages/ui/package.json`.

## Errors / Corrections

- Initial draft of the context provider leaked the `PayloadChildRenderFunction` type to `Context.Provider.children`. Fixed by narrowing with `children as React.ReactNode` at the provider boundary. Not a runtime fix — the payload form is unsupported in these wrappers by design.
- First attempt tried to integrate motion while preserving Base UI's CSS-driven `data-open`/`data-closed` animations. Realized both systems fighting would double-animate; switched to `actionsRef.unmount` + `AnimatePresence` which is Base UI's documented external-animation escape hatch.

## Ready for Next Run

- Task 03 (Combobox/Command/Select/ScrollArea/Tabs) will consume `@agh/ui`'s Dialog via `CommandDialog`. The pattern above (actionsRef + AnimatePresence + render prop) is the template.
- Task 05 (Sidebar) currently depends on `web/src/components/ui/sheet.tsx`; sidebar.tsx now imports Sheet from `@agh/ui` directly so its migration will only need to touch the sidebar-specific compositions.
- Task 08 (close `web/src/components/ui/`) still needs to migrate `command.tsx`, `sidebar.tsx`, and a dozen other primitives; those files' imports of dialog/sheet/tooltip now point at `@agh/ui`.
