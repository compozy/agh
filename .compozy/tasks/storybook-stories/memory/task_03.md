# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

13 overlay + navigation stories authored under `web/src/components/ui/stories/`. `bun run typecheck:raw`, `bun run lint`, and `bun run build-storybook` all pass; the Storybook manifest indexes exactly these 13 new titles under `components/ui/<Name>`.

## Important Decisions

- Skipped `tags: ["autodocs"]` on every new story to match ADR-003 (primitives only).
- Combobox filtering story uses `ComboboxCollection` inside `ComboboxList` because `ComboboxList` children are either a `ReactNode` **or** a render function — mixing `ComboboxEmpty` with an inline render callback would drop the empty slot.
- Sidebar story uses `SidebarProvider` + `SidebarInset` + `SidebarTrigger` inside `layout: "fullscreen"` so reviewers can exercise the open/closed transition without a page shell.

## Learnings

- Base UI `Combobox` auto-filters when items are of shape `{ value, label }`; no `itemToStringLabel` needed.
- `CollapsibleTrigger` exposes `data-panel-open` on the rendered element; using a `group/<name>` class on the trigger plus `group-data-[panel-open]/<name>:*` on descendants is the idiomatic swap pattern.
- `bun run lint` (web) runs `oxfmt` before `oxlint`; the formatter will rewrite stories to its preferred style — author close to final format to avoid large diffs.

## Files / Surfaces

- `web/src/components/ui/stories/{accordion,breadcrumb,collapsible,combobox,command,dialog,dropdown-menu,popover,scroll-area,sheet,sidebar,tabs,tooltip}.stories.tsx`
- `.compozy/tasks/storybook-stories/_tasks.md` (task 03 → completed)
- `.compozy/tasks/storybook-stories/task_03.md` (status + subtasks)

## Errors / Corrections

- Initial Sidebar `navItems` used `as const` with a badge only on the first entry; narrowed typing caused `item.badge` access to fail. Fixed by declaring an explicit `NavItem` interface with optional `badge`.
- First Combobox story mixed `ComboboxEmpty` with an inline render-function child of `ComboboxList` — replaced with explicit `ComboboxCollection` wrapper so empty + collection coexist.

## Ready for Next Run

- Task 04 can reuse the same story shape (title prefix `components/ui/<Name>`, no autodocs, `args: {}`, compose with `@agh/ui` primitives) for the forms + misc batch.
