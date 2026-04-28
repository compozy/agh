# Task Memory: task_32.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Phase 6 batch 2 — rewrite 5 settings sub-routes (`/settings/mcp-servers`, `/hooks-extensions`, `/observability`, `/environments`, `/network`) on `@agh/ui` primitives + task-30 shells. Hooks, mutation payload shapes, data-testids preserved. Empty / Dirty / Loading / Error / Restart variants exercised in stories. Playwright baselines regenerated per sub-route.

## Important Decisions

- Kept the existing hook contracts unchanged in all 5 routes. Skipped any task-plan test items that reference fields not present in the current hooks (log-level, OTLP endpoint, TLS, allowed-origins, bind-port). Task's MUST requirement ("preserve every existing hook call unchanged") and precedent from task 31 both override the speculative test-plan items.
- `/sandbox` switched from a card-grid to a `@agh/ui` `Table` per the task spec, but preserved the `settings-page-environments-card-<name>-*` testid prefix. Reasoning: the Dirty + DeleteProfile stories + the 11 existing unit tests all index on "card-" and changing the prefix provides zero test value for a purely presentational rewrite.
- For multi-select chips (hooks-extensions allowed kinds), reused `pillToggleVariants` from `@agh/ui` directly as the className on raw `<button>` elements. Pills primitive is scalar-valued; ToggleGroup's styling doesn't match DESIGN.md §5 chip vocabulary. Keeping raw buttons + the canonical toggle style is the cleanest path.
- MCP servers row status is rendered via `StatusDot` with `data-tone="configured"` as a visual indicator — MCP entries don't carry a reachability/health signal in the data model, so "configured = present in catalog" is the only honest semantic. Row still exposes env + args counts as before.
- Observability overview metrics use `SettingsStatGrid` + `SettingsStatItem` (which compose `@agh/ui` `Metric`). Metric cards: Active sessions / Active agents / Storage used / Capacity %. Capacity detail = `"of soft cap"`; storage detail = `"of <cap>"`. Task spec's "sessions, events, rate" metric vocabulary doesn't match the current observability runtime (no events rate counter), so I used the real signals.
- MCP servers + hooks-extensions + environments action banners consume `@agh/ui` `Alert` + `AlertDescription` + `AlertAction` with `variant="success|info"` + `role="status"`. Consistent vocabulary across batch 1 + 2.
- MCP servers scope selector switched from bespoke chip buttons to `@agh/ui` `Pills<V>` with values `"global" | "ws:<workspace_id>"`. The `ws:` prefix is an encoded discriminator; the Pills `onChange` handler splits it back into `selectGlobal()` vs `selectWorkspace(id)`.
- Observability "Open stream" kept as a plain `<a>` (not Button render prop) because the testid needs to apply to an `<a>` element — test asserts `href` attribute on `getByTestId("settings-page-observability-log-tail-link")`.

## Learnings

- `@agh/ui` `StatusDot` spreads `{...props}` AFTER its internal `data-tone={tone}`, so callers can override the tone attribute freely (e.g. `data-tone="configured"` for semantic-label-driven test assertions). Useful when the visual tone maps to one thing but the test-driven label maps to another (DESIGN.md tone vs domain status vocabulary).
- `Pills<V>` accepts any string-literal union via its generic, so an encoded discriminator scheme like `"global" | \`ws:${string}\`` works cleanly without losing type safety.
- For dialog-gated dirty baselines, the 2-stage RAF pattern (`trigger.click()` → wait for input mount → `setValue`) from the task 31 providers story is directly reusable. Copy-paste the `StorybookProvidersDirtySetup` shell and swap the two testids.

## Files / Surfaces

**Rewritten routes**
- `web/src/routes/_app/settings/mcp-servers.tsx` — Pills scope, Table list, Empty + Alert primitives; editor dialog swapped `<input>`/`<select>` for `Input`/`NativeSelect`.
- `web/src/routes/_app/settings/hooks-extensions.tsx` — Table for hooks, Empty for both hooks + extensions, Alert banners, Input/NativeSelect in policy section, MonoBadge chips for hook events.
- `web/src/routes/_app/settings/observability.tsx` — `SettingsStatGrid` + `Metric` overview row, Input for number fields, MonoBadge cap indicator.
- `web/src/routes/_app/sandbox.tsx` — card grid → Table, Empty primitive, Alert action banner, Input/NativeSelect in editor dialog.
- `web/src/routes/_app/settings/network.tsx` — Input for numeric + text fields (no legacy `<input type="number">`).

**Tests**
- `-mcp-servers.test.tsx` — added Empty + StatusDot + Alert primitive assertions.
- `-hooks-extensions.test.tsx` — added hooks-empty + extensions-empty Empty + Alert assertions.
- `-observability.test.tsx` — added Metric grid assertion.
- `-environments.test.tsx` — updated empty + alert tests to assert primitive data-slots.

**Stories** (all 5 updated with `Dirty` + additional `Empty` where required)
- `-mcp-servers.stories.tsx` — new `Dirty` (2-stage editor open+dirty), new `Empty` (MSW empty).
- `-hooks-extensions.stories.tsx` — new `Dirty` (registry input).
- `-observability.stories.tsx` — new `Dirty` (retention days).
- `-environments.stories.tsx` — new `Dirty` (2-stage editor open+dirty), retained `Empty` (MSW empty).
- `-network.stories.tsx` — new `Dirty` (port input).

**Visual baselines** (regenerated in `web/tests/visual/__snapshots__/`)
- 21 regenerated baselines (existing states per sub-route).
- 5 new Dirty baselines (one per sub-route).
- 1 new Empty baseline (mcp-servers).
- Total batch 2: 26 baselines. Full web visual suite: 318 baselines, all passing.

## Errors / Corrections

Initial attempt used `Button render={<a />}` for the observability log-tail link; reverted to plain `<a>` because the testid contract asserts `getByTestId(...).toHaveAttribute("href", ...)` — simpler and avoids Base UI button render overhead.

## Ready for Next Run

Phase 6 end gate reached: every `/settings/*` sub-route now renders on @agh/ui + task-30 shell with zero legacy `@/components/ui/*` or `@/components/design-system/*` imports. `bun run web lint`, `typecheck:raw`, `test:raw` (1491 tests in 201 files), `build`, and `test:visual` (318 baselines) all green. No follow-up tasks carry over — redesign PRD tasks 01–32 are complete.
