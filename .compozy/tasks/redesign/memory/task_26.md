# Task Memory: task_26.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Knowledge domain (`web/src/systems/knowledge/**` + `routes/_app/knowledge.tsx`) rewritten as a pure visual migration over `@agh/ui`. Domain adapters/hooks/types/query wiring untouched — `useKnowledgePage` keeps its exact surface.

## Important Decisions

- Domain testid contracts preserved: `knowledge-list-panel`, `knowledge-search-input`, `knowledge-list-empty`, `knowledge-list-loading`, `knowledge-list-error`, `knowledge-group-{scope}`, `knowledge-group-header-{scope}`, `memory-item-{filename}`, `memory-active-indicator`, `type-badge-{type}`, `scope-badge-{scope}`, `knowledge-detail-panel`, `knowledge-detail-loading`, `knowledge-detail-error`, `knowledge-detail-empty`, `detail-type-badge`, `detail-scope-badge`, `content-preview`, `metadata-table`, `metadata-row-{Type|Scope|Agent|Modified}`, `delete-memory-btn`, `view-in-cli-btn`, `dream-status`, `knowledge-shell`, `knowledge-shell-title`, `knowledge-shell-icon`, `knowledge-loading`, `knowledge-error`, `knowledge-split-pane`, tab pills `tab-{all|global|workspace}`.
- New testids introduced: `knowledge-delete-dialog`, `confirm-delete-memory-btn`, `cancel-delete-memory-btn`.
- Pills selection test assertion switched from class check (`text-[#e8572a]`) to `aria-pressed`.
- Delete button still emits `onDelete(filename)` but only after the `Dialog` confirm — `onDelete` not called on cancel.
- Delete dialog extracted into standalone component `knowledge-delete-dialog.tsx` (exported from domain barrel) so the snapshot baseline can render it with `open={true}` directly without a play fn.
- `CodeBlock` inside detail panel uses `copyable showPrompt={false}` — copy affordance kept, terminal `$ ` prefix suppressed for markdown prose.
- Scope badge on detail panel shows "GLOBAL" / "WORKSPACE" full label; list rows use compact "GLOBAL" / "WS".
- Dropped `view-full-content-link` testid entirely — `CodeBlock` owns horizontal scroll natively; no truncation needed.

## Learnings

- The repo has a story-source invariant in `src/storybook/web-storybook-stories-and-fixtures.test.tsx` that asserts specific absolute import paths for certain component stories — using relative `../` imports fails the test. Always use `@/systems/<domain>/components/<file>` in migrated stories.
- `Empty` primitive renders both `title` and `description` with the same copy if they overlap — scope test assertions with `{ selector: "h3" }` when the strings collide.

## Files / Surfaces

Rewritten:
- `web/src/systems/knowledge/components/knowledge-list-panel.tsx`
- `web/src/systems/knowledge/components/knowledge-detail-panel.tsx`
- `web/src/routes/_app/knowledge.tsx`
- `web/src/routes/_app/-knowledge.test.tsx`
- `web/src/systems/knowledge/components/stories/knowledge-list-panel.stories.tsx`
- `web/src/systems/knowledge/components/stories/knowledge-detail-panel.stories.tsx`
- `web/src/routes/_app/stories/-knowledge.stories.tsx`

Added:
- `web/src/systems/knowledge/components/knowledge-delete-dialog.tsx`
- `web/src/systems/knowledge/components/knowledge-delete-dialog.test.tsx`
- `web/src/systems/knowledge/components/knowledge-list-panel.test.tsx`
- `web/src/systems/knowledge/components/knowledge-detail-panel.test.tsx`
- `web/src/systems/knowledge/components/stories/knowledge-delete-dialog.stories.tsx`
- `web/src/systems/knowledge/lib/knowledge-formatters.ts`
- `web/src/systems/knowledge/lib/knowledge-formatters.test.ts`

Updated:
- `web/src/systems/knowledge/index.ts` (export new dialog + formatters)

Baselines committed (darwin):
- Route: `routes-app-stories-knowledge--{default,empty,content-loading,content-error}-chromium-darwin.png`
- List panel: `systems-knowledge-knowledgelistpanel--{default,empty,filtered-empty,loading,error,scope-global-only,scope-workspace-only}-chromium-darwin.png`
- Detail panel: `systems-knowledge-knowledgedetailpanel--{default,workspace-scope,no-content,loading,error-state,empty-selection}-chromium-darwin.png`
- Delete dialog: `systems-knowledge-knowledgedeletedialog--{default,pending-delete}-chromium-darwin.png`
- Removed orphan: `systems-knowledge-knowledgedetailpanel--empty-chromium-darwin.png`

## Errors / Corrections

- First story rewrite used `../knowledge-*` relative imports → the source-invariant test `web-storybook-stories-and-fixtures.test.tsx` demands absolute `@/systems/...` paths. Fixed to absolute.
- Two initial unit tests tripped on duplicate `Empty` copy (same text in `title` + `description`). Scoped with `{ selector: "h3" }` + `within()`.

## Ready for Next Run

- `make verify` is still blocked by the same pre-existing Go lint issues (gocyclo + gosec) flagged in MEMORY.md open-risks — nothing introduced by this task.
- Web-only pipeline (`web-fmt`, `web-lint`, `web-typecheck`, `web-test`, `test:visual`) is 100% green: 1387 unit tests, 272 visual baselines, 0 lint/typecheck issues.
