# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Built the message rendering surface on top of the task_13 shell: `Timeline`, `MessageRow` (full / collapsed / system variants), `DatePill`, `NewDivider`, `HoverToolbar`, hybrid right-rail `ThreadOverlay`, headerless `DirectRoom`, list views, activity feed, and supporting hooks.
- Author-group collapsing implemented in `web/src/systems/network/lib/group-messages.ts` with the 60s window from `_design.md` §5.3 and break rules on author / kind / gap.
- Date pills cross midnight + year boundaries via `format-timestamp.ts`; New divider positions at first message after `lastReadAt`.
- Thread overlay mounts in the existing `RightRail` slot on `>=1024px` (URL canonical) and as full-page in `<1024px` via `useThreadViewMode` (canonical `1024px` breakpoint via `matchMedia` + `useSyncExternalStore`).

## Important Decisions

- Right-rail overlay rendering is owned by `web/src/routes/_app/network.tsx` (reads `activeThreadId` from `useNetworkRouteShell`) so the L4 right rail is a true sibling of `<main>` per `_design.md` §3.1; the threads route still ships full-page overlay handling for `<1024px` via its own `<Outlet />`.
- Component complexity caps (`compozy-react(max-component-complexity)`) forced extraction of `useDirectRoom` and `useThreadOverlay` custom hooks to keep the rendering components thin. Mocks targeting `useNetworkDirectDetail` / `useNetworkThreadDetail` / `useNetworkMessages` still cover behavior because the new hooks compose those primitives.
- `useNetworkPresence` returns static `idle` per `_design.md` §5.6 and §11.3 — wired through but kept as a placeholder until presence telemetry exists.

## Learnings

- TanStack file-based nesting: `network.$channel.threads.$threadId.tsx` is a child of `network.$channel.threads.tsx`. The parent must render an `<Outlet />` for the child component to mount; without it the child is silent.
- `verbatimModuleSyntax: true` is enforced repo-wide — type-only re-exports must use `export type { ... }` and component imports of types need `import type`.
- `Link` `params` are typed per route — sharing `Record<string, string>` across thread/direct route entries triggers TS errors; use discriminated union types when collapsing them into one list.
- `oxfmt` reformats files automatically on `make web-lint` runs — keep stories and tests authored in oxfmt-friendly shape (single-line JSX where the linter prefers).
- Avatar gutter widths in code: 36px for channel timeline, 32px for the thread overlay; `MessageAvatar` accepts `sizePx: 36 | 32` to keep the choice explicit.

## Files / Surfaces

- `web/src/systems/network/lib/{format-timestamp,group-messages}.ts` (+ tests)
- `web/src/systems/network/components/timeline/*` (timeline, full / collapsed / system rows, avatar, body, hover toolbar, date pill, new divider, index, tests)
- `web/src/systems/network/components/thread-overlay/*` (overlay, header, root, replies, tests)
- `web/src/systems/network/components/threads/threads-list.tsx` (+ tests)
- `web/src/systems/network/components/directs/{directs-list,direct-room}.tsx` (+ tests)
- `web/src/systems/network/components/activity/activity-feed.tsx` (+ tests)
- `web/src/systems/network/hooks/{use-threads,use-directs,use-messages,use-network-presence,use-thread-view-mode,use-thread-overlay,use-direct-room}.ts` (+ tests)
- `web/src/routes/_app/network*.tsx` (network shell wires overlay into right rail; thread/direct/activity routes integrate the new components)
- `web/src/systems/network/components/stories/{timeline,message-row,thread-overlay}.stories.tsx` + `storybook.ts` exports + `web-storybook-stories-and-fixtures.test.tsx` registration

## Errors / Corrections

- Initial overlay placement was inside `network.$channel.threads.tsx` — refactored so the shell-level `RightRail` owns overlay rendering on `>=1024px` while the threads route only owns the full-page mode.
- Lint flagged `DirectRoom` (6 hooks) and `ThreadOverlay` (behavior score 8) — extracted to `useDirectRoom`/`useThreadOverlay`; tests still pass because the underlying primitives are still mocked at the same boundary.
- Initial typed `Link` params used `Record<string, string>` for the activity feed — switched to discriminated union (`ThreadEntry | DirectEntry`) to satisfy router `to`/`params` typing.

## Ready for Next Run

- Composer (`composer/`), `work-banner`, `work-chip`, work inspector, and empty-state polish are still owed by task_15.
- Hover toolbar handlers are no-op stubs awaiting task_15 wiring (Reply opens overlay; Pin/Fork/Kebab call mutations).
- Polling cadence is inherited from task_13 query options; SSE migration is out-of-scope.
- `useNetworkPresence` is the single place to update once presence telemetry ships (currently always `idle`).
