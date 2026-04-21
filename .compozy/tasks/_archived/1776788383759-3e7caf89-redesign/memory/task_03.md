# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Relocate shadcn batch 2 primitives (Combobox, Command, Select, ScrollArea, Tabs) from `web/src/components/ui/` to `packages/ui/src/components/`, with colocated unit tests and stories. Rewrite every importer and delete the originals. No domain-code refactor beyond import path changes.

## Important Decisions

- **Kept existing CSS open/close animations** (`data-open:animate-*` from `tw-animate-css`) on Select + Combobox popups rather than converting to motion's `AnimatePresence` pattern. Reason: task scope is refactor; Select's `alignItemWithTrigger` and its lack of `keepMounted` on `SelectPortal` make a motion migration non-trivial. Shared memory's motion template applies when motion is introduced — do not mix.
- **Inlined minimal InputGroup helpers** (`ComboboxInputGroup`, `ComboboxInputGroupAddon`, `ComboboxInputControl`) inside `combobox.tsx`, and a local search-input wrapper inside `command.tsx`, to avoid depending on `web/src/components/ui/input-group.tsx` (scheduled for task_08). Helpers are not exported from `@agh/ui`.
- **`ComboboxInputControl` uses a raw `<input>`** instead of `@agh/ui`'s `Input` (which is Base UI's `Field.Control`) — `Field.Control` inside a Combobox.Input `render={}` caused the combobox to ignore value/typing state in tests.
- **`cmdk` added as a `packages/ui` runtime dependency** via `bun add --cwd packages/ui cmdk`; command primitive requires it.
- **`Tabs` now forwards `orientation` to Base UI's TabsRoot.** Original destructured it without forwarding (Base UI never received it); the task's vertical-orientation test required the fix.
- **`Element.prototype.scrollIntoView` mocked in `packages/ui/src/test-setup.ts`** to unblock cmdk-driven tests under jsdom.
- **Scroll-area unit test asserts scrollbar markup only when `keepMounted` is set**, because Base UI's `ScrollArea.Scrollbar` returns `null` until it measures overflow — impossible under jsdom.
- **Pre-existing failures** in `web/src/systems/tasks/components/tasks-empty-state.test.tsx` (`EmptyTitle` rendered as `<div>` not `<h2>`) and Go lint (gosec, gocyclo) are **not in this task's scope**. Confirmed by `git stash` + re-run showing the same failures without my changes.

## Learnings

- Base UI Combobox + `user.type` in jsdom does not fire the `input` event that Base UI's combobox listens to. Use `fireEvent.change(input, { target: { value: "…" } })` to exercise filtering.
- In multi-select mode, chips are rendered by the consumer from controlled state; nothing renders chips automatically. Tests/stories that want to see chips must map selected values to `<ComboboxChip>` elements.

## Files / Surfaces

- Added: `packages/ui/src/components/{tabs,scroll-area,select,combobox,command}.tsx` + `.test.tsx` + `stories/*.stories.tsx`.
- Modified: `packages/ui/src/index.ts` (25 new exports), `packages/ui/src/test-setup.ts` (scrollIntoView mock), `packages/ui/package.json` (cmdk dep), `bun.lock`.
- Deleted: the five sources and their stories plus `web/src/components/ui/hooks/use-combobox-anchor.ts`.

## Errors / Corrections

- Initial `combobox.test.tsx` used `userEvent.type` and expected Base UI to fill the input after `{Enter}` selection; switched to `fireEvent.change` to drive filtering and removed the fill-on-enter assertion (that assertion never held in jsdom regardless of the underlying Base UI default).
- First multi-select chip test rendered no `<ComboboxChip>` children; fixed by controlling `value` and mapping `selected.map(... <ComboboxChip> ...)`.

## Ready for Next Run

- Task 08 is the next consumer of the inlined InputGroup pattern; when `input-group.tsx` moves to `@agh/ui`, consider replacing the inline helpers in `combobox.tsx`/`command.tsx` with the shared primitive (or leave inline — both are fine).
- If a follow-up wants motion-driven open/close on Select/Combobox popups, the template in shared memory applies; beware `SelectPortal` does not expose `keepMounted`.
