# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 01 (Foundation) complete: types, schemas, barrels, ThemeProvider, route tests all in place.
- Task 02 (Daemon & Agent Systems) complete: API adapters, query hooks, connection status, agent sidebar groups all wired.
- Task 03 (Session System: API, CRUD & Sidebar) complete: session-api adapter, query hooks, mutations, sidebar items wired.
- Task 04 (Streaming Core & Session Store) complete: streaming buffer, event mapper, Zustand store, useSessionChat hook, tool labels all in place with tests.
- Task 05 (Chat View, Messages & Composer) complete: virtualized chat-view, message-bubble with markdown/syntax highlighting, thinking-block, message-composer, chat-header, processing-indicator, session.$id route fully wired, all tests passing.
- Task 06 (Tool Cards & Renderers) complete: collapsible tool-call-card with 3 status modes (executing/success/error), auto-expand/collapse with localStorage persistence, 6 specialized renderers (bash, read, write, edit, search, generic), integrated into chat-view tool_group rows, all tests passing.

## Shared Decisions

- Zod 4 is installed — `z.record(key, value)` requires explicit key type (not just value like v3).
- No separate `routes/index.tsx` — `_app/index.tsx` already serves `/` since `_app` is pathless layout. Downstream tasks should not create root index route.
- Route component tests use `(Route as any).component` cast due to TanStack Router type constraints.

## Shared Learnings

- Backend API payload structs are in `internal/httpapi/{sessions,agents,prompt,observe,daemon}.go` and `internal/observe/health.go`.
- All backend token/cost fields are nullable pointers — map as optional in TS.
- AI SDK `useChat` with `DefaultChatTransport` handles standard SSE events natively (text/reasoning deltas); custom daemon events (`data-agh-permission`, `data-agh-event`) arrive via the `onData` callback.
- Streaming buffer uses snapshot approach (reset + re-fill from AI SDK accumulated parts) since AI SDK already accumulates deltas internally — avoids double-counting.

## Open Risks

## Handoffs
