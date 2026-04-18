# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Close Phase 2 by rewriting `web/src/components/design-system/design-system-showcase.tsx` into a flat, `@agh/ui`-only consumer at `web/src/components/design-system-showcase.tsx`, deleting the `design-system/` folder, and updating the `/design-system` route to import the new location.

## Important Decisions

- **Flat location + export shape.** Showcase lives at `web/src/components/design-system-showcase.tsx` (sibling to `app-sidebar.tsx`). Default export is `{ DesignSystemShowcase, SECTIONS, TOKEN_GROUPS }` so the test file can assert grouping + anchor coverage without duplicating the table.
- **Section anchors point to the GitHub-hosted DESIGN.md.** Absolute `https://github.com/compozy/agh/blob/main/DESIGN.md#<slug>` URLs (git remote = `git@github.com:compozy/agh.git`), so the links resolve both in dev and in deployed builds instead of 404ing against the Vite server.
- **Token swatch contract.** `TOKEN_GROUPS` enumerates every AGH-owned custom property from `packages/ui/src/tokens.css` (excludes shadcn aliases like `--background` that resolve to `var(--color-*)`). The test regex-parses `:root` and asserts every `--color-*|--radius-*|--duration-*|--ease-*|--tracking-*` with a literal value is rendered as a `[data-token]` card. Shadcn aliases are covered implicitly by the primitives themselves.
- **No `asChild` on Button.** `@agh/ui` Button is a Base UI `ButtonPrimitive` — wrap with `render={<a .../>}` (or any element) instead of the Radix `asChild` pattern. Same applies to every other `@agh/ui` + Base UI primitive (`DialogTrigger`, `SheetTrigger`, `PopoverTrigger`, `TooltipTrigger`, `DropdownMenuTrigger`, `CollapsibleTrigger`).
- **Progress renders its own track.** `<Progress value>` auto-mounts `<ProgressTrack><ProgressIndicator /></ProgressTrack>` — passing them as additional children double-renders the bar. Consumers should only add `<ProgressLabel />` + `<ProgressValue />` as children.
- **ToggleGroup is array-valued.** Base UI's ToggleGroup reads/emits `readonly string[]`, not a single string — use `defaultValue={["tasks"]}` for the single-select case. My first pass with a string state variable compiled but failed at runtime because Base UI ignored the scalar.
- **Overlays stay closed by default.** Dialog/Sheet/Popover/Tooltip/DropdownMenu render only their trigger + Base UI root in the showcase; opening is an interaction, not the baseline paint. This keeps the route safe for eventual Playwright snapshots (task 16) and avoids testing-library portal collisions.
- **Showcase route is unchanged in path + file name.** Only the import line flipped from `@/components/design-system` → `@/components/design-system-showcase`.

## Learnings

- `getByText("Outline")` collides between `Button variant="outline"` text and `Badge variant="outline"` text — prefer `getByRole("button", { name: ... })` or `getAllByText` with a `length >= N` assertion whenever a demo renders both shapes.
- `ScrollArea` logs an `act(...)` warning on initial mount inside jsdom (it updates state after measuring). This is noise from the Base UI `ScrollAreaRoot`, not something the showcase can fix; other packages/ui tests absorb the same warning.
- Multi-line `import { ... } from "@agh/ui";` blocks require `source.matchAll(/from ["']([^"']+)["']/g)` for the import-source contract check. A line-by-line filter against `^\s*import\b` only sees the opening `import {` line and misses the module specifier on the closing line.
- The `/design-system` route lives at `web/src/routes/design-system.tsx` — it is _not_ under `_app/`, so it bypasses `AppSidebar` + the route-level motion wrapper. The showcase supplies its own `TooltipProvider` + `Toaster` inside the page so it renders standalone.
- `stories/story-frame.tsx` (the dead `StoryFrame`/`TexturedStoryFrame` pair) had no callers outside the deleted folder — safe to remove with the rest of the directory.

## Files / Surfaces

- `web/src/components/design-system-showcase.tsx` — new flat showcase, pure `@agh/ui` + lucide-react + react consumer.
- `web/src/components/design-system-showcase.test.tsx` — 18 tests (section coverage, token swatch parity vs tokens.css, DESIGN.md anchor contract, import-source contract, route-import contract, deleted-folder assertion).
- `web/src/routes/design-system.tsx` — import flipped to `@/components/design-system-showcase`.
- **Deleted:** `web/src/components/design-system/design-system-showcase.tsx`, `.../index.ts`, `.../stories/design-system-showcase.stories.tsx`, `.../stories/story-frame.tsx`.

## Errors / Corrections

- Initial showcase used `<Button asChild>` for the DESIGN.md shortcut — replaced with `render={<a .../>}` per Base UI's render-prop pattern.
- Initial ToggleGroup passed a string to `value` — replaced with uncontrolled `defaultValue={["tasks"]}` (array form).
- Initial Progress wrapped `<ProgressTrack><ProgressIndicator /></ProgressTrack>` as children, duplicating the auto-rendered track — simplified to `<ProgressLabel />` + `<ProgressValue />` only.
- Initial test scraped imports line-by-line and missed the `@agh/ui` multi-line block — switched to `matchAll(/from ["']([^"']+)["']/g)`.
- Initial test asserted `getByText("Outline")` — failed because Badge `variant="outline"` also renders literal text "Outline"; switched to role-based queries and `getAllByText` length checks.

## Ready for Next Run

- Task 16 (web Playwright visual harness) can snapshot `/design-system` directly. The showcase exposes stable `data-testid` selectors (`design-system-showcase`, `section-*`, `section-link-*`, `token-group-*`, `token-<name>`) and all overlays stay closed at paint time, so a single baseline shot covers the whole page. Forward-reference from the task_15 deliverable "Playwright snapshot baseline committed for the showcase page" remains a **follow-up for task 16** — not landed in this task because the web visual harness does not exist yet.
- No regressions introduced in other domains; the 42 pre-existing `internal/daemon` Go test failures and 2 pre-existing `golangci-lint` issues are unrelated to task 15 (Go packages untouched) and are flagged in shared workflow memory under "Open Risks".
