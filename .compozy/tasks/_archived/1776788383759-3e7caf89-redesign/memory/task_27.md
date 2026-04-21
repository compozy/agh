# Task Memory: task_27.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite `web/src/systems/skill/**` + `web/src/routes/_app/skills.tsx` as a pure visual migration over `@agh/ui`. Split-pane installed tab (list + detail) + marketplace tab (card grid). Domain adapters, query hooks, and `useSkillsPage` preserved unchanged.

## Important Decisions

- Page-level tab switch uses `@agh/ui` `Tabs` (TabsList + TabsTrigger) with labels `Installed` / `Marketplace`, testids `tab-installed` / `tab-marketplace`. Pills was replaced with Tabs per spec.
- Detail panel header is `@agh/ui` `PageHeader` with Wrench icon + title + meta containing (version | author | source) as `MonoBadge`s. `source-badge` + `detail-version-badge` + `detail-author-badge` testids.
- Enable/disable is a Base UI `Switch` (role=switch, not checkbox). Testid `skill-enabled-switch` wired to `onCheckedChange(next) → next ? onEnable : onDisable`. `disabled={isActionPending}` exposes `aria-disabled="true"` + `data-disabled`. `toBeDisabled()` does NOT match — tests assert attributes instead.
- `Capabilities` section pulls from `skill.metadata.capabilities: string[]` (render as `MonoBadge` row, Empty fallback). `Recent calls` pulls from `skill.metadata.recent_calls: Array<{ label, status?, timestamp? }>` (render as `Table`, Empty fallback).
- Marketplace view is a responsive `Card` grid (sm:2 cols, xl:3 cols). Each card uses `size="sm"` + `CardHeader` (icon well + name + author/version/downloads meta) + `CardContent` (description + tag `MonoBadge`s) + `CardFooter` (`Button` install OR `MonoBadge` INSTALLED). Category filter is `@agh/ui` `Pills` with `category-chip-{CAT}` testids.
- Legacy testid contracts preserved so existing route tests survive with minimal edits: `skill-list-panel`, `skill-item-{name}`, `skill-active-indicator`, `skill-status-dot-{name}`, `skill-search-input`, `skill-list-empty`, `skill-group-{source}`, `marketplace-view`, `marketplace-search-input`, `marketplace-row-{name}`, `installed-pill-{name}`, `install-btn-{name}`, `marketplace-empty`, `content-body/-empty/-loading/-error`, `view-full-content-btn`, `retry-view-content-btn`. New testids: `skill-group-header-{source}`, `skill-list-loading`, `skill-list-error`, `skill-source-badge-{name}`, `skill-enabled-switch`, `skill-enabled-toggle`, `skill-capabilities-list/-empty`, `skill-capability-{cap}`, `skill-recent-calls-table/-empty`, `skill-recent-call-row-{idx}`, `marketplace-grid`, `marketplace-category-pills`, `marketplace-tag-{name}-{tag}`, `skills-shell`, `detail-version-badge`, `detail-author-badge`, `skill-detail-icon`, `skill-detail-title`.
- Marketplace-grid route baseline uses a non-play-fn `MarketplaceTabAutoClick` wrapper that polls via requestAnimationFrame until `[data-testid='tab-marketplace']` mounts, then clicks it. Storybook's `StorybookProvidersBoundary` renders the story render fragment AS SIBLING of RouterProvider, so useEffect inside the story render fires before the /skills route mounts — polling avoids a first-render race. Separate play-fn `MarketplaceInteraction` story kept for interaction coverage.

## Learnings

- Base UI `Switch` from `@agh/ui` renders as `<span role="switch">` with `aria-disabled` + `data-disabled` — the `toBeDisabled()` matcher does NOT match (expects native `disabled` attribute). Use `toHaveAttribute("aria-disabled", "true")` + `toHaveAttribute("data-disabled")` instead.
- `make verify` invokes Go lint which fails on pre-existing base-branch issues (`gosec G202` in `internal/observe/tasks.go` + `gocyclo` in `internal/store/globaldb/global_db_task_aux.go`). These are already documented in shared MEMORY under Open Risks. Scoped verification for this task is `make web-lint web-typecheck web-test web-build` + `bun run --cwd web test:visual`.
- `git stash` + `make verify` is a brittle combination during a scoped task — Go lint auto-fixes modified the unrelated pre-existing uncommitted files on disk, which then blocked the subsequent `git stash pop` with "would be overwritten by merge". Recovery path: `git checkout stash@{0} -- <my-files>` to restore only the task's modifications; the Go files on disk already matched the stash content.
- Playwright visual harness excludes `play-fn`-tagged stories from the baseline suite. To snapshot a tab-driven state that has no URL or store hook, render a tiny non-play-fn React wrapper that polls `document.querySelector` in `useEffect` until the target element exists, then `.click()` it synchronously. The re-render commits before the screenshot is taken (verified against `MarketplaceGrid` route baseline).

## Files / Surfaces

**Rewritten (tracked as M):**
- `web/src/routes/_app/skills.tsx` — route shell: PageHeader + Tabs + SplitPane (installed) / MarketplaceView (marketplace). Preserves loading/error early-returns via `Empty` + `skills-loading`/`skills-error` testids.
- `web/src/routes/_app/-skills.test.tsx` — switched to Switch testid + aria-disabled assertions + capabilities/recent-calls assertions. Metadata table assertions removed.
- `web/src/routes/_app/stories/-skills.stories.tsx` — renamed stories: InstalledPopulated, InstalledEmpty, DetailOpen, MarketplaceGrid (+ MarketplaceInteraction play-fn). `MarketplaceTabAutoClick` helper polls RAF for the tab element.
- `web/src/systems/skill/components/skill-list-panel.tsx` — SearchInput + grouped rows (bundled → workspace → marketplace → user → additional) + StatusDot + MonoBadge source chip + Empty states.
- `web/src/systems/skill/components/skill-detail-panel.tsx` — PageHeader + Switch-powered enable/disable + Section blocks (Overview/Capabilities/Recent Calls) + Table for recent calls + content preview card.
- `web/src/systems/skill/components/marketplace-view.tsx` — Card grid + Pills category filter + install Button or Installed MonoBadge + Empty state.
- `web/src/systems/skill/components/stories/{skill-list-panel,skill-detail-panel,marketplace-view}.stories.tsx` — covers default/loading/error/empty/disabled-install/filter-to-empty + toggle-switch play-fn.

**Added (untracked):**
- `web/src/systems/skill/lib/skill-formatters.ts` + `.test.ts` — source ordering/tone/short label, author derivation, tags/capabilities/recent_calls parsers, category matching, `filterSkillsByQuery`, relative time formatter.
- `web/src/systems/skill/components/{skill-list-panel,skill-detail-panel,marketplace-view}.test.tsx` — new unit suites covering grouping, filtering, switch toggling, install actions, category/search filters, empty states.
- `web/tests/visual/__snapshots__/routes-app-stories-skills--{installed-empty,installed-populated,detail-open,marketplace-grid}-chromium-darwin.png` — required route baselines.
- `web/tests/visual/__snapshots__/systems-skill-{marketplaceview,skilldetailpanel,skilllistpanel}--{…}-chromium-darwin.png` — component baselines for loading/error/empty/default/disabled-skill/disabled-install/all-installed.

**Removed (orphan baselines from old story names):**
- `routes-app-stories-skills--default-chromium-darwin.png` (renamed → installed-populated)
- `routes-app-stories-skills--empty-chromium-darwin.png` (renamed → installed-empty)
- `systems-skill-marketplaceview--error-chromium-darwin.png` (renamed → error-state)

## Errors / Corrections

- Initial `MarketplaceTabAutoClick` fired `querySelector` once in useEffect; baseline captured the installed tab because RouterProvider had not yet mounted `/skills` at the moment the effect ran. Fix: recursive `requestAnimationFrame` poll until the tab button exists, then click. Baseline now shows the marketplace grid with all-installed MonoBadges.
- Initial tests asserted `toBeDisabled()` against the Switch — failed because Base UI Switch uses `aria-disabled` not the native `disabled` attribute. Switched assertions to `toHaveAttribute("aria-disabled", "true")` + `toHaveAttribute("data-disabled")`.
- Initial test asserted `tab-installed` had text "INSTALLED" (uppercase) — tab label is "Installed" (mixed-case from mock). Fixed assertion.
- Running `make verify` inside a temporary `git stash` corrupted the uncommitted pre-existing Go file state (lint --fix writes alongside stashed content) and blocked `git stash pop`. Recovery via `git checkout stash@{0} -- <scoped files>`.

## Ready for Next Run

Task complete. Web pipeline green, 282 visual baselines green. No follow-up debt for this domain.

Skills route no longer imports from `@/components/ui/*` or `@/components/design-system/*`. `useSkillsPage` hook + adapters untouched. Marketplace install path still gated by `installUnavailableReason` — daemon does not yet support install.
