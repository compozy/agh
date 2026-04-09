# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Build Skills page with three-panel layout (sidebar + skill list + detail), INSTALLED/MARKETPLACE tabs, grouped skill list, detail panel with actions, marketplace view with search/filters.

## Important Decisions

- Active workspace derived from `useWorkspaces()` first entry (same pattern as sidebar's fallback) — no global workspace store exists yet
- Skill components placed in `systems/skill/components/` per app-renderer-systems pattern
- Marketplace tab reuses the same skill list data from `useSkills` (no separate marketplace API endpoint exists yet)
- Auto-selects first skill when no selection is made for better UX

## Learnings

- oxfmt reformats arrow function parens — write `s =>` not `(s) =>` for single params
- Test file naming convention: `-skills.test.tsx` (dash prefix) for co-located route tests
- `vi.importActual` combined with mock overrides preserves real component exports while mocking hooks

## Files / Surfaces

- `web/src/routes/_app/skills.tsx` — Main route file (replaced placeholder)
- `web/src/systems/skill/components/skill-list-panel.tsx` — NEW: Grouped skill list with search
- `web/src/systems/skill/components/skill-detail-panel.tsx` — NEW: Detail panel with badges, actions
- `web/src/systems/skill/components/marketplace-view.tsx` — NEW: Marketplace tab with filters
- `web/src/systems/skill/index.ts` — Updated barrel with component exports
- `web/src/routes/_app/-skills.test.tsx` — NEW: 24 tests, >96% coverage

## Errors / Corrections

- Initial test had `screen.getByText("alpha-skill")` which matched multiple elements (list + detail panel) — fixed with `within(detailPanel)` scoping

## Ready for Next Run
