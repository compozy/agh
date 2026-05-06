# Task Memory: task_20.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Hard-cut the web Knowledge surface to the Memory v2 contract. Update list/detail/search/edit/delete/decisions UI through generated OpenAPI types only, drop legacy filename-prefix scope derivation, and add scope/tier-aware controls plus controller-backed mutations and decision context.

## Important Decisions

- KnowledgeSelector is the canonical web-side identity for a memory: `{ scope, workspaceId?, agentName?, agentTier? }`. All adapters, query options, hooks, and components accept it instead of positional args.
- Adapters route through `apiClient.PATCH("/api/memory/{filename}")`, `apiClient.POST("/api/memory/search")`, and `apiClient.GET("/api/memory/decisions")` to consume the final Memory v2 routes.
- Hook surface adds `useMemorySearch` (server-backed recall, only enabled when query trimmed and selector present) and `useMemoryDecisions` (selector-scoped). Mutations expose `useEditMemory` plus the existing delete/dream-trigger mutations.
- `use-knowledge-page` uses three scope tabs (`global | workspace | agent`); `agent` requires an explicit name and exposes a tier toggle (`workspace | global`). When `workspace`/`agent-workspace`, the active workspace id is required and the route shows a guard message until provided.
- When a search query is non-empty AND a selector is present, the page swaps the list source from `useMemories` to `useMemorySearch`. Search recall info appears under the search input; cleared search returns to list mode.
- Decisions are filtered client-side to the selected memory's filename (via `target_filename` or `frontmatter.filename`) so the panel only shows controller decisions relevant to the open memory.
- Action target key is intentionally not cleared on mutation failure; failed delete/edit messages stay visible until the user changes selection, scope, or search input.

## Learnings

- `editMemory`'s generated request body does not include `expected_hash` (TechSpec note left over from a draft); the UI must rely on `idempotency_key` if it ever needs optimistic-concurrency and not invent a missing field.
- `groupKnowledgeMemoriesByScope` already used `memory.scope`; removing `deriveScopeFromFilename` was safe because the generated header carries scope. The derivation helper would have masked daemon truth in agent-scoped fixtures (filenames are bare slugs, not prefixed paths).
- `memoryHeadersFixture` had legacy `global/`, `workspace/` filename prefixes that never matched the daemon contract. New fixtures use bare slugs (e.g. `operator-style.md`); MSW selector filtering handles scope routing.
- The `make verify` gate runs the full Bun + Go monorepo and finished in ~17s on cache; web tests alone covered 1550 tests in ~80s.

## Files / Surfaces

- `web/src/systems/knowledge/types.ts` — added `KnowledgeSelector`, `KnowledgeAgentTier`, `MemoryDecision*`, `MemorySearchRequest/Response`, edit/delete response types via `OperationResponse`/`OperationRequestBody` only.
- `web/src/systems/knowledge/adapters/knowledge-api.ts` — selector-shaped `listMemories`/`readMemory`/`deleteMemory`; new `editMemory`, `searchMemory`, `listMemoryDecisions`. Returns `summary + content` shape from `readMemory` (single object).
- `web/src/systems/knowledge/lib/query-keys.ts` + `query-options.ts` — selector-tuple keys, new search/decision query options with proper `enabled` gating.
- `web/src/systems/knowledge/hooks/use-knowledge.ts` + `use-knowledge-actions.ts` — new selector-shaped hooks plus `useMemorySearch`, `useMemoryDecisions`, `useEditMemory`.
- `web/src/systems/knowledge/components/knowledge-list-panel.tsx` — renders agent tier/agent name/recall count/staleness/system badges; `searchMode` toggles placeholder + empty copy.
- `web/src/systems/knowledge/components/knowledge-detail-panel.tsx` — full Memory v2 metadata table (agent tier, workspace, recall count, last recalled, staleness, supersession, injection, system_managed), edit dialog wiring, decisions section, supersession chip.
- `web/src/systems/knowledge/components/knowledge-edit-dialog.tsx` — new edit dialog wired to the controller edit body.
- `web/src/systems/knowledge/components/knowledge-decisions-section.tsx` — new section rendering controller decisions with op/source/confidence/applied chips.
- `web/src/systems/knowledge/components/knowledge-pill-tone.ts` — added decision op/source tone helpers and the `agent` scope tone.
- `web/src/systems/knowledge/components/knowledge-delete-dialog.tsx` — types tightened to `KnowledgeScope`; copy mentions controller decision flow.
- `web/src/systems/knowledge/lib/knowledge-formatters.ts` + `lib/knowledge-list.ts` — removed `deriveScopeFromFilename`/`resolveKnowledgeScope`; added agent tier labels and decision op/source labels.
- `web/src/systems/knowledge/mocks/fixtures.ts` + `handlers.ts` — bare-slug filenames, agent-scoped fixtures, search/edit/decisions fixtures and MSW routes (`PATCH /api/memory/:filename`, `POST /api/memory/search`, `GET /api/memory/decisions`).
- `web/src/hooks/routes/use-knowledge-page.ts` + `web/src/routes/_app/knowledge.tsx` — scope tabs (`global|workspace|agent`), agent name + tier inputs, server-backed search, decisions panel wiring, edit + delete handlers using full selector.
- `web/src/routes/_app/-knowledge.test.tsx`, `web/src/hooks/routes/use-knowledge-page.test.tsx`, system tests, plus new component/edit/decisions stories and tests; `Should ...` naming applied to refreshed tests.

## Validation Evidence

- `bunx vitest run web/src/systems/knowledge web/src/hooks/routes/use-knowledge-page.test.tsx web/src/routes/_app/-knowledge.test.tsx` — 14 files / 141 tests passed.
- `bunx vitest run web/src/lib/memory-api-contract.test.ts web/src/storybook/web-storybook-stories-and-fixtures.test.tsx web/src/hooks/routes/use-settings-memory-page.test.tsx` — 8 files / 29 tests passed.
- `make web-lint` — 0 warnings / 0 errors (oxfmt + oxlint).
- `make web-typecheck` — clean (`tsgo --noEmit` passes after codegen-check).
- `make web-test` — 205 files / 1550 tests passed.
- `make web-build` — Vite + tsc bundle produced; only the standard chunk-size warning remains (unchanged from prior runs).
- `make verify` — DONE 8359 tests in 17.870s; "OK: all package boundaries respected".

### No-workarounds Remediation Validation (post-fix)

- `rg "as any|@ts-ignore|@ts-expect-error|eslint-disable" web/src/systems/knowledge web/src/hooks/routes/use-knowledge-page.ts web/src/hooks/routes/use-knowledge-page.test.tsx web/src/routes/_app/knowledge.tsx web/src/routes/_app/-knowledge.test.tsx web/src/routes/_app/stories/-knowledge.stories.tsx` — no matches.
- `bunx vitest run src/routes/_app/-knowledge.test.tsx src/hooks/routes/use-knowledge-page.test.tsx` — 2 files / 27 tests passed in 1.51s.
- `make web-lint` — 0 warnings / 0 errors (oxfmt + oxlint).
- `make web-typecheck` — clean (`tsgo --noEmit` after codegen-check).
- `make web-test` — 205 files / 1550 tests passed.
- `make verify` — DONE 8359 tests in 64.249s; "OK: all package boundaries respected".

## Errors / Corrections

- Initial list-panel test asserted `getByTestId("type-badge-user")` but the new fixtures expose two `user`-typed memories (global + agent). Switched to `getAllByTestId` and asserted the count + tone uniformly.
- Initial route test relied on default-selected memory carrying the decision target, but `sortKnowledgeMemories` orders alphabetically; selected memory was `project_migration.md`, not `user_role.md`. Test now clicks the matching list row before asserting decisions render.
- First version of `handleDelete`/`handleEdit` used `try/finally` to clear the action target, which wiped the failure marker before the next render and hid the inline delete-error banner. Restored the legacy "clear only on success" behavior.
- `use-knowledge-page` test mock initially returned `useMemoriesMock(scope, workspace, options)` shape; new selector signature broke it. Mock now reads from a single `selector` argument.
- Route test originally accessed the page component via `(Route as any).component as () => React.ReactNode` with an `eslint-disable` to bypass the typed `createFileRoute` return shape. That violated `$no-workarounds`. Removed the cast and the lint suppression by exporting `KnowledgePage` as a named export from `web/src/routes/_app/knowledge.tsx` so the test imports the real function reference directly. Kept the existing `vi.mock("@tanstack/react-router")` factory because the route module still calls `createFileRoute` at evaluation time.

## Ready for Next Run

Task 20 done. Next Phase B iteration should execute `task_21` (Web Memory Settings Surface).
