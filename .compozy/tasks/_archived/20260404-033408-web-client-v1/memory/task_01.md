# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Foundation task: install deps, create domain types with Zod schemas, barrel exports, ThemeProvider, route integration tests.

## Important Decisions

- **No separate `routes/index.tsx` redirect**: The `_app` prefix is a pathless layout route in TanStack Router, so `_app/index.tsx` already serves `/` within the sidebar layout. A separate `routes/index.tsx` would create a route collision at `/`. The redirect requirement is satisfied implicitly.
- **Zod 4 `z.record()` requires explicit key type**: `z.record(z.string())` in Zod 3 becomes `z.record(z.string(), z.string())` in Zod 4.
- **Route test type casting**: TanStack Router's Route type doesn't expose `component` as a public property. Tests use `(Route as any).component` cast to access the mocked component function.

## Learnings

- Zod 4 (`zod@^4.3.0`) is installed — uses `z.infer<>` at type level, `safeParse()` at runtime same as v3, but `z.record()` API changed.
- Backend token usage fields use nullable pointers (`*int64`, `*float64`) — mapped as optional numbers in TS schemas.

## Files / Surfaces

- `web/package.json` — added 7 dependencies
- `web/src/systems/session/types.ts` — SessionPayload, UIMessage, AgentEventPayload, TokenUsagePayload, etc.
- `web/src/systems/agent/types.ts` — AgentPayload, AgentMCPServer
- `web/src/systems/daemon/types.ts` — HealthPayload
- `web/src/systems/{session,agent,daemon}/index.ts` — barrel exports
- `web/src/routes/__root.tsx` — added ThemeProvider wrapper
- `web/src/routes/__root.test.tsx` — integration tests
- `web/src/routes/_app.test.tsx` — integration tests
- `web/src/systems/{session,agent,daemon}/types.test.ts` — Zod schema unit tests

## Errors / Corrections

- First run: `z.record(z.string())` caused Zod 4 runtime error — fixed to `z.record(z.string(), z.string())`
- First typecheck: `Route.component` not accessible on TanStack Router types — fixed with `any` cast in test files

## Ready for Next Run

All subtasks complete. 53 tests passing, typecheck clean, lint clean.
