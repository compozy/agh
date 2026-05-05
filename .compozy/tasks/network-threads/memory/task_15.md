# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Composer + work surfacing + empty/error/disabled states + realtime polling are wired on top of the task_13 shell and task_14 timeline. Final make verify is green: 0 lint warnings, 2213 bun tests, 8386 Go tests, web build OK.

## Important Decisions

- `useOpenWork` aggregates open work from the *already-loaded* messages of the active container; the `_design.md` §13.3 prohibition on "client-side aggregation of full message lists" applies to channel-level counters, not the active container which is forensic detail (the alternative — N+1 `getNetworkWork` calls — was worse). Channel-level summaries still come from `NetworkChannelSummary.open_work_count` server-side.
- Optimistic message ID uses `crypto.randomUUID()` (`_design.md` §14.6); the canonical replacement matches by `message_id` rather than full-row replacement so polling can still merge cleanly.
- Channel-level composer collision retry treats *any* send error as collision-eligible — one silent retry with a fresh UUID, then surface the toast. Picking the strict 409-only policy would have left UUID collisions unhandled if the server signals a generic 400. The toast still fires at the end, so the user always sees a single error.
- The Work Inspector occupies the right rail only when no thread overlay is open AND `open_work_count > 0` for the active container. When a thread is open the right rail keeps the thread mode (`_design.md` §5.8.3).
- Behavior was extracted into custom hooks (`useComposerState`, `useDirectRoomView`, `useThreadOverlayView`, `useNetworkChannelThreadsRoute`) to satisfy `compozy-react(max-component-complexity)`. Keep the pattern when adding more behavior.
- The `__collision__` channel name in MSW handlers is the test/storybook hook for thread collisions — return 409 unconditionally for `surface:"thread"` sends to that channel.

## Learnings

- The `compozy-react(max-component-complexity)` lint rule caps components at ~5 hooks / 7 behavior score. Always front-load extraction into a `use-*-view.ts` (or `-state.ts`) helper next to the component when the component does anything beyond pure rendering.
- `DropdownMenuTrigger` from `@agh/ui` is base-ui's `MenuPrimitive.Trigger`, NOT radix — it uses the `render={...}` slot pattern, not `asChild`. The lint catches misuse but the typecheck error is a clearer signal.
- `useMutation`'s `mutationFn` runs after a microtask flush; tests asserting optimistic cache state must `await Promise.resolve()` after `mutateAsync` before reading the cache, OR move the optimistic write to `onMutate` (we left it inline in `mutationFn` for retry symmetry).
- `vi.advanceTimersByTime(...)` does NOT trigger React state updates outside `act()`. Banner auto-hide (`setTimeout` → `setPhase("hidden")`) needs `act(() => vi.advanceTimersByTime(400))`.

## Files / Surfaces

- New components: `web/src/systems/network/components/{composer,work,empty-states}/*`, `directs/new-direct-dialog.tsx`, `directs/use-direct-room-view.ts`, `thread-overlay/use-thread-overlay-view.ts`, `composer/use-composer-state.ts`.
- New hooks: `hooks/use-network-actions.ts` (rewritten), `hooks/use-work.ts`, `hooks/use-active-session.ts`, `hooks/use-network-channel-threads-route.ts`.
- New lib: `lib/use-elapsed.ts`; work-state helpers added to `lib/network-formatters.ts`.
- Updated routes: `routes/_app/network.tsx`, `network.$channel.threads.tsx`, `network.$channel.directs.tsx`. Direct/Thread detail routes unchanged but their components now render the composer + work surfaces.
- Updated existing components: `MessageRow` (work chip, optimistic retry/discard), `Timeline` (passes optimistic + work-chip handlers), `ThreadOverlay`, `DirectRoom`, `ThreadsList`/`DirectsList` (use new empty states), `ChannelHeader` (kebab refresh menu).
- Storybook: `composer.stories.tsx`, `work.stories.tsx`, `empty-states.stories.tsx` registered in `storybook.ts`.
- MSW: `__collision__` channel name forces a 409 on `sendNetworkMessage` to exercise the collision toast path.

## Errors / Corrections

- First lint run flagged `Composer`, `DirectRoom`, `ThreadOverlay`, `NetworkChannelThreadsRoute` for hook-count overruns. Resolved by extracting view-model hooks; do not re-merge them.
- First typecheck flagged `refetchInterval: ({ state }) => …` — the v5 callback receives the full `query` object, not destructured `{ state }`. Use `query.state.data`.
- First test run failed because `vi.advanceTimersByTime` ran outside `act()` for the work banner auto-hide. Fixed by wrapping in `act`.

## Ready for Next Run

- Task 15 is complete in local commit (queued for commit) with verified `make verify` PASS evidence: bun-lint 0/0, bun-test 2213/2213, go test 8386 done, OK boundaries.
- Task 16 (docs) can lean on the new components: `Composer`, `ChannelThreadComposer`, `DetailComposer`, `WorkBanner`, `WorkChip`, `WorkInspector`, `NewDirectDialog`, plus the new state copy from `empty-states/*`.
- Task 18/19 (QA) should treat `__collision__` channel + the disabled-session reasons (`Pick a channel to start composing.`, `Loading channel…`, `Join this channel from another surface to compose here.`) as test scenarios.
