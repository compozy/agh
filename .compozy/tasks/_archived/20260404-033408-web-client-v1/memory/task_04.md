# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Streaming core & session store: streaming buffer, event mapper, Zustand store, useSessionChat hook, tool labels.

## Important Decisions

- AI SDK `useChat` with `DefaultChatTransport` handles SSE transport; custom `onData` callback intercepts `data-agh-permission` events for the permission flow.
- Buffer uses snapshot approach: on each AI SDK message update, reset buffer and re-fill from accumulated AI SDK parts, then schedule rAF flush. This avoids double-counting since AI SDK already accumulates deltas internally.
- `transformAIMessage` converts AI SDK's `UIMessage` (with `parts`) to our flat `UIMessage` format for non-streaming messages.
- `event-mapper.ts` returns empty partials for `agent_message`/`thought`/`permission`/`done`/`error` since those are handled by the streaming buffer or the store directly.

## Learnings

- AI SDK's `useChat` already handles `text-delta`, `reasoning-delta`, `text-start/end`, `reasoning-start/end` natively via parts. Custom events (`data-agh-permission`, `data-agh-event`) come through the `onData` callback.
- `session-sidebar-item.test.tsx` had a type error: the Link mock's spread `...props` needed explicit typing with `[key: string]: unknown` to accept `data-testid` and other HTML attributes passed by the SidebarMenuSubButton mock.

## Files / Surfaces

- `web/src/systems/session/lib/streaming-buffer.ts` — SimpleStreamingBuffer + mergeStreamingChunk
- `web/src/systems/session/lib/event-mapper.ts` — mapAgentEventToUIMessage + extractPermissionRequest
- `web/src/systems/session/stores/session-store.ts` — Zustand session store
- `web/src/systems/session/lib/tool-labels.ts` — tool icons, labels, compact summaries
- `web/src/systems/session/hooks/use-session-chat.ts` — useSessionChat hook
- `web/src/systems/session/index.ts` — barrel exports (updated)
- `web/src/systems/session/components/session-sidebar-item.test.tsx` — fixed type error in Link mock

## Errors / Corrections

- Fixed type error in `session-sidebar-item.test.tsx:9`: Link mock needed explicit typing for spread props to satisfy `children` type constraint on `<a>`.

## Ready for Next Run

Task complete. All tests pass (194/194), typecheck clean, lint clean. Coverage >=80% on all task_04 files.
