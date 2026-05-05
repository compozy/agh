# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Replace the flat `/network` web IA with a channel-pivot shell: 6 file-based routes, channel rail with cross-channel Recents, channel header with Threads/Directs/Activity tabs, right-rail slot, query keys namespaced by `[network, channel, surface, containerId]`, last-read tracking, and a hard-cut browser artifact rename from `network_selected_peer` to `network_selected_thread`/`network_selected_direct`.

## Important Decisions

- Kept `KindChip`, `NetworkCreateChannelDialog`, `createNetworkChannelDraft`, `toggleDraftAgent`, `sortAgentsForNetwork` since the dialog and design-system showcase still consume them. Pruned the rest of the legacy IA helpers from `lib/network-formatters.ts` and `types.ts` (room list/active room/details tab models).
- `NetworkChannelSummary` from the generated OpenAPI lacks `open_work_count`; the shell renders the `openWorkCount` slot at zero in task_13. Aggregating per-channel work lands in task_15 alongside the Work Inspector.
- Route-shell hook complexity hit the oxlint `compozy-react(max-component-complexity)` ceiling (max 5 hook calls). Extracted `useNetworkRouteShell` into `web/src/systems/network/hooks/use-network-route-shell.ts` so the route component stays a single-hook caller.
- Storybook stories use `parameters.router = { kind: "stub" as const }` for shell components; component unit tests mock `@tanstack/react-router` `Link` as an `<a>` tag rather than building per-test routers. Less brittle.
- `buildLastReadStorageKey` joins channel/surface/containerId with a literal `:` separator (not whitespace) — the original space-template produced null bytes when written through the editor pipeline.

## Learnings

- The shipped fixtures had `network-workspace-shell.stories.tsx` and `routes/_app/-network.test.tsx` (1000+ lines) hard-wired to the legacy IA; both were deleted. The `web-storybook-stories-and-fixtures.test.tsx` regression guard now imports `networkShellStories`, `networkChannelRailStories`, `networkChannelHeaderStories` and renamed `networkChannelMessagesFixture` → `networkThreadMessagesFixture`.
- MSW handlers for the new routes (`/threads`, `/threads/{id}`, `/threads/{id}/messages`, `/directs`, `/directs/{id}`, `/directs/{id}/messages`, `/directs/resolve`, `/work/{id}`) replace the removed `/channels/{channel}/messages` and `/peers/{peer_id}/messages` endpoints in `web/src/systems/network/mocks/handlers.ts`.
- `NetworkChannelSummary` requires `peer_count`; test channel literals must include it or typecheck fails.
- `Route.options.id` is not exposed publicly on `createFileRoute(...)` results in this TanStack version, so the route registration test asserts `route.options.component` and the existence of `routeTree`, not the id string.
- `network-create-channel-dialog.tsx` still depends on `NetworkCreateChannelDraft`, so that type and its helpers stay in the system index; the dialog's storybook regression assertion remains unchanged.

## Files / Surfaces

- New: `web/src/systems/network/components/shell/{network-shell,channel-rail,channel-rail-recents,channel-rail-row,channel-header,channel-tabs,right-rail,index}.tsx`, `web/src/systems/network/components/stories/{channel-rail,channel-header,network-shell}.stories.tsx`, `web/src/systems/network/hooks/{use-channels,use-last-read,use-recents,use-network-page,use-network-route-shell}.ts`, `web/src/systems/network/hooks/{use-channels,use-last-read,use-recents}.test.tsx?`, `web/src/systems/network/components/shell/{channel-rail,channel-tabs}.test.tsx`, `web/src/systems/network/lib/palette.ts`, `web/src/routes/_app/{network.$channel.threads,network.$channel.threads.$threadId,network.$channel.directs,network.$channel.directs.$directId,network.$channel.activity}.tsx`, `web/src/routes/_app/-network-routes.test.ts`.
- Rewritten: `web/src/routes/_app/network.tsx`, `web/src/systems/network/{adapters/network-api,lib/{query-keys,query-options,network-formatters},types,index,storybook,components/stories/network-create-channel-dialog.stories,mocks/{fixtures,handlers,index,network-mocks.test}}.ts(x)`, `web/src/routes/_app/stories/-network.stories.tsx`, `web/src/routes/_app/settings/-network.test.tsx`, `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `web/e2e/fixtures/{artifacts,artifacts.test,browser-artifact-session,browser-artifact-session.test}.ts`, `web/src/routeTree.gen.ts`.
- Deleted: `web/src/hooks/routes/use-network-page.ts`, `web/src/systems/network/hooks/use-network.ts`, `web/src/systems/network/components/network-workspace-shell.tsx`, `web/src/systems/network/components/network-workspace-shell.test.tsx`, `web/src/systems/network/components/stories/network-workspace-shell.stories.tsx`, `web/src/routes/_app/-network.test.tsx`.

## Errors / Corrections

- Initial `bun run build` flagged 8 JSX-in-`.ts` syntax errors → renamed `use-channels.test.ts`/`use-recents.test.ts` to `.tsx`.
- `open_work_count` was used on `NetworkChannelSummary` (not on the contract); removed the channel-rail-row open-work badge and shell openWorkCount aggregator and left them as task_15 follow-up.
- `storage` key template literals were emitted with U+0000 separators by the editor pipeline → switched to explicit `[…].join(":")`.
- Channel-tabs/rail unit tests using a hand-rolled `createRouter` rendered an empty body in jsdom; replaced with `vi.mock("@tanstack/react-router", () => ({ Link }))` shim that produces an `<a>`.
- Original recents merge test asserted activity ordering by hand-sorted timestamps (incorrect) — fixed expectation to match real desc sort by `last_activity_at`.

## Ready for Next Run

- task_14 timeline/thread overlay can mount inside the shell's `right-rail.tsx` slot (use `rightRailMode` prop) and read messages via `networkThreadMessagesOptions`/`networkDirectMessagesOptions` factories — both already namespaced and isolated.
- task_15 work surfacing should aggregate `open_work_count` from the active channel's threads + directs lists; the shell exposes the slot via the `openWorkCount` prop in `NetworkShell`.
- task_16 docs should reference the renamed browser artifact fields and the post-MVP scope guard added in `routes/_app/settings/-network.test.tsx`.
