# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 01 (Foundation) complete: types, schemas, barrels, ThemeProvider, route tests all in place.
- Task 02 (Daemon & Agent Systems) complete: API adapters, query hooks, connection status, agent sidebar groups all wired.

## Shared Decisions

- Zod 4 is installed — `z.record(key, value)` requires explicit key type (not just value like v3).
- No separate `routes/index.tsx` — `_app/index.tsx` already serves `/` since `_app` is pathless layout. Downstream tasks should not create root index route.
- Route component tests use `(Route as any).component` cast due to TanStack Router type constraints.

## Shared Learnings

- Backend API payload structs are in `internal/httpapi/{sessions,agents,prompt,observe,daemon}.go` and `internal/observe/health.go`.
- All backend token/cost fields are nullable pointers — map as optional in TS.

## Open Risks

## Handoffs
