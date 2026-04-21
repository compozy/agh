# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Relocate DropdownMenu, Switch, Toggle, ToggleGroup, Accordion, Collapsible from `web/src/components/ui/` to `@agh/ui`, rewrite every importer, delete originals, land stories + tests with ≥80% coverage.

## Important Decisions

- Preserved existing CSS-driven animations (`animate-accordion-down/up` from `tw-animate-css`, `transition-all` hover/focus) — did **not** swap to motion. Matches memory guidance: simpler, no risk of double-animation.
- Rebuilt `ToggleGroup` in `@agh/ui` using sibling import of `toggleVariants` from `./toggle` (instead of `@/components/ui/toggle`).
- Accordion test uses Base UI's `multiple` prop — Base UI does not accept `type="single"/"multiple"` (that's Radix). Same for ToggleGroup (`multiple` boolean, not `toggleMultiple`).

## Learnings

- `web/src/lib/utils.ts` only re-exports `cn` from `@agh/ui`. When a test mocks `@agh/ui` as a whole, `cn` in the underlying module becomes undefined unless either (a) the mock also provides `cn`, or (b) the mock uses `vi.importActual` so the rest of the module stays intact. For narrow mocks, the `importActual` spread pattern is the cleanest fix.
- The `web-storybook-stories-and-fixtures.test.tsx` integration test dynamically imports every story module. Moving a story out of `web/src/components/ui/stories/` into `packages/ui/src/components/stories/` requires removing it from the `import(...)` array (tests can't resolve the `@/` alias across the package boundary) and switching its source-file assertion to a relative `../packages/ui/...` resolve.
- Pre-existing failures remain outside scope: `tasks-empty-state.test.tsx` queries `getByRole("heading")` against a `<div data-slot="empty-title">`, which is broken on the base branch (`git stash` confirms it fails there too). Do not fix in this task.

## Files / Surfaces

Added in `packages/ui/src/components/`:
- `dropdown-menu.tsx` + `.test.tsx` + `stories/dropdown-menu.stories.tsx`
- `switch.tsx` + `.test.tsx` + `stories/switch.stories.tsx`
- `toggle.tsx` + `.test.tsx` + `stories/toggle.stories.tsx`
- `toggle-group.tsx` + `.test.tsx` + `stories/toggle-group.stories.tsx`
- `accordion.tsx` + `.test.tsx` + `stories/accordion.stories.tsx`
- `collapsible.tsx` + `.test.tsx` + `stories/collapsible.stories.tsx`

Rewritten importers (tsx):
- `web/src/components/app-sidebar.tsx`
- `web/src/systems/session/components/thinking-block.tsx`
- `web/src/systems/agent/components/agent-sidebar-group.tsx`
- `web/src/systems/bridges/components/{bridge-create-dialog,bridge-edit-dialog}.tsx`
- `web/src/routes/_app/settings/{automation,hooks-extensions,memory,network,observability,skills}.tsx`

Rewritten test mocks:
- `web/src/components/app-sidebar.test.tsx`
- `web/src/systems/agent/components/agent-sidebar-group.test.tsx` (uses `importActual` to keep `cn`)
- `web/src/systems/session/components/{chat-view,permission-prompt}.integration.test.tsx`
- `web/src/systems/session/components/message-bubble.test.tsx`
- `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`

Deleted: six primitive sources + five stories in `web/src/components/ui/` (+ `stories/toggle.stories.tsx`).

## Errors / Corrections

- Initial DropdownMenu test used `DropdownMenuLabel` outside a `DropdownMenuGroup`, which Base UI rejects with "MenuGroupRootContext is missing". Wrapped label in `DropdownMenuGroup`.
- Initial `agent-sidebar-group.test.tsx` mock replaced all of `@agh/ui`, breaking `cn`. Switched to `importActual` spread.

## Ready for Next Run

- Verification:
  - `bun run --cwd packages/ui test` → 16 files, 74 tests passing.
  - `make web-typecheck` passing.
  - `make web-lint` clean (0 warnings / 0 errors).
  - `make web-build` succeeds.
  - `make web-test` → 1176/1178 passing; only pre-existing `tasks-empty-state.test.tsx` failure remains.
  - `rg "@/components/ui/(dropdown-menu|switch|toggle|toggle-group|accordion|collapsible)" web/src` → 0 matches.
- `make verify` currently fails end-to-end only on the same pre-existing failures already documented in shared MEMORY "Open Risks".
