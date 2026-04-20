# Task Memory: task_28.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite `web/src/systems/workspace/**` (onboarding + setup dialog + selector + page shell) and `web/src/systems/agent/**` (sidebar group + icon) on `@agh/ui` primitives only. These screens are NOT in the mock — derived per ADR-004 non-mocked rule.

## Important Decisions

- Extracted all onboarding + dialog copy into `web/src/systems/workspace/lib/workspace-setup-copy.ts` so the components stay presentational.
- Built a local `SetupOptionCard` helper inside `workspace-setup.tsx` that composes `@agh/ui` `Section` (eyebrow label + right slot) + an icon well + body. Keeps both onboarding and dialog variants consistent and satisfies the "must compose on Section" rule without fighting Section's mono-eyebrow label styling.
- `WorkspaceSelector` API changed from native-select (`value` + `onValueChange`) to list-of-buttons (`activeWorkspaceId` + `onSelectWorkspace` + `globalWorkspaceId`). Each row is Avatar + name + HOME/PATH `Pill` + root_dir + StatusDot. Empty state renders `@agh/ui` `Empty`. No production consumer uses it yet — safe to break the API.
- `WorkspacePageShell` now takes `icon: ComponentType` (matching `PageHeader`'s contract) instead of `ReactNode`. No production route uses it; only a story + one mocked test file (`-network.test.tsx`).
- `AgentIcon` gained an explicit `tone` prop (`default | muted | accent`) + `data-slot="agent-icon"` + `data-provider` for deterministic targeting; size still `size-4` per DESIGN.md inline-icon convention.
- `AgentSidebarGroup` gained `defaultOpen` + `sessionCount` props. When `sessionCount > 0`, a `MonoBadge` count appears in the trigger right slot. Preserves existing test contract (new testids use the `agent-sidebar-group-*-${agent.name}` pattern).
- Dialog tests render inside `UIProvider reducedMotion="always"` so Base UI `AnimatePresence` resolves synchronously. The "closes after success" test is driven through the global-workspace button because that path calls `onSuccessClose` which invokes `onOpenChange(false)`.

## Learnings

- `@agh/ui` `Section.label` is a fixed `<h2>` with mono-eyebrow styling (`font-mono text-[11px] uppercase tracking-[0.06em]`). Overriding to regular-case title is awkward — cleaner to use the label as a short mono eyebrow (e.g. "Global" / "Path") and render the actual card title inside the body.
- `@agh/ui` Dialog automatically unmounts its portal when `open` is `false` (via AnimatePresence + `{open ? <Portal keepMounted>...</Portal> : null}`), so jsdom tests with `reducedMotion="always"` can assert `queryByTestId` without waiting for motion exit.
- `@agh/ui` `Empty.icon` accepts both Lucide component refs and ReactNodes; prefer the component ref so the primitive handles the stroke/size contract.
- `WorkspaceSelector` was never wired into a production consumer — `app-sidebar.tsx` has its own inline `RailSlot`. The rewrite is a standalone primitive that mirrors the rail's HOME/PATH semantics for future consumers.

## Files / Surfaces

- `web/src/systems/workspace/components/workspace-setup.tsx` — full rewrite; shared `SetupOptionCard` + `WORKSPACE_SETUP_COPY`.
- `web/src/systems/workspace/components/workspace-selector.tsx` — full rewrite; Avatar + Pill + StatusDot list.
- `web/src/systems/workspace/components/workspace-page-shell.tsx` — `PageHeader` + `Section` composition.
- `web/src/systems/agent/components/agent-icon.tsx` — added `tone` prop + `data-slot`/`data-provider`.
- `web/src/systems/agent/components/agent-sidebar-group.tsx` — added `defaultOpen`/`sessionCount`, stable testids.
- `web/src/systems/workspace/lib/workspace-setup-copy.ts` — new copy module.
- Tests: `workspace-setup.test.tsx`, `workspace-selector.test.tsx`, `agent-icon.test.tsx`, `agent-sidebar-group.test.tsx` — rewritten for new API / testids, UIProvider wrapper, `userEvent` where applicable.
- Stories: `workspace-setup.stories.tsx` (OnboardingDefault/OnboardingPathError/OnboardingGlobalUnavailable/SetupDialogOpen + play-fn SubmitManualPath/UseGlobalWorkspace), `workspace-selector.stories.tsx` (Empty/Single/Many/Active), `workspace-page-shell.stories.tsx` (icon → ComponentType), `agent-sidebar-group.stories.tsx` (ExpandedWithSessions/Collapsed/NoSessions/DisabledNewSession + play-fn ExpandCollapseInteraction/NewSessionAction).
- Visual baselines: regenerated 12 baselines (4 workspace selector + 4 agent sidebar group + 4 workspace setup); deleted 5 orphans (Default/EmptyGroup/ValidationError/Default selector/setup). `routes-app-stories-index--onboarding` baseline refreshed because the onboarding visual changed.

## Errors / Corrections

- First pass used Section's label slot for the card titles ("Use global workspace" / "Register workspace"). Section forces mono-eyebrow styling on labels — reworked to put titles in the body and use Section's label for the eyebrow ("Global" / "Path").
- First pass passed `icon: ReactNode` through `WorkspacePageShell`; `PageHeader` expects a `ComponentType`. Fixed the prop type and updated the only story that consumed it to pass `icon={Book}`.

## Ready for Next Run

- `make web-lint`, `make web-typecheck`, `make web-test` (1444 tests), `make web-build`, `bun run test:visual` (288 baselines), `packages/ui` vitest (232 tests) — all green.
- `make verify` not run here: base branch has pre-existing Go lint failures (`internal/observe/tasks.go`, `internal/store/globaldb/...`) and unrelated uncommitted Go edits. Per shared memory, scoped web verification is the preferred gate for UI-only tasks.
- No production consumers of `WorkspaceSelector` or `WorkspacePageShell` yet; any future consumer should adopt the new prop shapes (`activeWorkspaceId` + `onSelectWorkspace` + optional `globalWorkspaceId`; `icon: ComponentType`).
