# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Build `systems/knowledge/` data layer wrapping existing `/api/memory` backend endpoints.

## Important Decisions

- Task spec had incorrect API paths (`:scope/:filename` in path, `POST` for write). Actual routes use query params for scope/workspace and `PUT` for write. Implemented against actual backend routes from `internal/api/httpapi/server.go`.
- ListMemory returns `[]MemoryHeader` directly (no envelope), unlike skills which wrap in `{skills: [...]}`. Adapter parses with `memoryHeaderSchema.array()`.
- Mutations invalidate `knowledgeKeys.all` (broader invalidation) since both list and detail queries could be stale after delete/consolidate.

## Learnings

- Memory API routes: `GET /api/memory`, `GET /api/memory/:filename`, `PUT /api/memory/:filename`, `DELETE /api/memory/:filename`, `POST /api/memory/consolidate`. Scope and workspace passed as query params.
- ConsolidateMemory returns `{triggered: bool, reason?: string}`, not `{ok: bool}`.

## Files / Surfaces

- `web/src/systems/knowledge/types.ts` — Zod schemas for MemoryHeader, MemoryType, MemoryScope, response types, KnowledgeFilter
- `web/src/systems/knowledge/adapters/knowledge-api.ts` — 5 adapter functions + KnowledgeApiError
- `web/src/systems/knowledge/lib/query-keys.ts` — hierarchical key factory
- `web/src/systems/knowledge/lib/query-options.ts` — queryOptions factories (30s stale, 60s refetch)
- `web/src/systems/knowledge/hooks/use-knowledge.ts` — useMemories, useMemory
- `web/src/systems/knowledge/hooks/use-knowledge-actions.ts` — useDeleteMemory, useConsolidateMemory
- `web/src/systems/knowledge/index.ts` — barrel export

## Errors / Corrections

- None.

## Ready for Next Run

Task 08 (Knowledge page) can now import from `@/systems/knowledge`.
