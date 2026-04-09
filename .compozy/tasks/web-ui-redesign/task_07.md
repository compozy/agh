---
status: done
title: Build knowledge frontend system
type: frontend
complexity: medium
dependencies: []
---

# Task 07: Build knowledge frontend system

## Overview

Create the `systems/knowledge/` module following the app-renderer-systems pattern: types, API adapter, query keys, query options, hooks, and public barrel export. This data layer wraps the existing Memory HTTP endpoints (`/api/memory`) and feeds the Knowledge page (task_08). No new backend work needed — the memory endpoints already exist.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Core Interfaces" section for knowledge adapter type definitions
- REFERENCE TECHSPEC "Existing Memory endpoints" table for API paths
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `systems/knowledge/types.ts` with Zod schemas for `MemoryHeader`, `MemoryScope`, `MemoryType`, and filter types
- MUST create `systems/knowledge/adapters/knowledge-api.ts` with `listMemories`, `readMemory`, `deleteMemory`, `writeMemory`, `consolidateMemory` functions
- MUST create `systems/knowledge/lib/query-keys.ts` with hierarchical query key factory
- MUST create `systems/knowledge/lib/query-options.ts` with `queryOptions` factories
- MUST create `systems/knowledge/hooks/use-knowledge.ts` with `useMemories` (list) and `useMemory` (detail) hooks
- MUST create `systems/knowledge/hooks/use-knowledge-actions.ts` with `useDeleteMemory` and `useConsolidateMemory` mutation hooks
- MUST create `systems/knowledge/index.ts` barrel with explicit named exports
- MUST pass `AbortSignal` from query context through to every API call
- MUST use `staleTime: 30000` and `refetchInterval: 60000` for list queries
- MUST invalidate knowledge queries after delete/write mutations via `onSettled`
</requirements>

## Subtasks
- [x] 7.1 Create Zod schemas and TypeScript types for knowledge/memory domain
- [x] 7.2 Create knowledge API adapter wrapping existing `/api/memory` endpoints
- [x] 7.3 Create query keys and query options factories
- [x] 7.4 Create query hooks (`useMemories`, `useMemory`) and mutation hooks (`useDeleteMemory`, `useConsolidateMemory`)
- [x] 7.5 Create barrel export `index.ts`
- [x] 7.6 Write tests for adapter functions and hooks

## Implementation Details

See TechSpec "Existing Memory endpoints" table for the complete API surface.

Follow the exact pattern established by `systems/agent/` — same file structure and conventions. The memory endpoints already exist in the backend (implemented in `internal/api/core/handlers.go`), so this task only creates the frontend data layer.

### Relevant Files
- `web/src/systems/agent/` — Reference system to follow as template
- `internal/api/contract/contract.go` — Existing `MemoryReadResponse`, `MemoryMutationResponse` types
- `internal/api/core/handlers.go` — Existing ListMemory, ReadMemory, WriteMemory, DeleteMemory, ConsolidateMemory handlers
- `internal/memory/store.go` — Memory types (MemoryHeader, MemoryType, Scope)

### Dependent Files
- `web/src/systems/knowledge/` — All files in this directory are new (created by this task)
- `web/src/routes/_app/knowledge.tsx` — Will import from this system (task_08)

### Related ADRs
- [ADR-003: Full Systems Architecture for Skills and Knowledge](../adrs/adr-003.md) — Mandates full data layer from day one

## Deliverables
- Complete `systems/knowledge/` directory with types, adapters, lib, hooks, and index
- Typed adapter functions for all 5 memory endpoints
- Query hooks with proper staleTime/refetchInterval configuration
- Mutation hooks with cache invalidation
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `listMemories` adapter calls `GET /api/memory?scope=:scope&workspace=:ws` and returns typed array
  - [x] `readMemory` adapter calls `GET /api/memory/:filename?scope=:scope` and returns content string
  - [x] `deleteMemory` adapter calls `DELETE /api/memory/:filename?scope=:scope`
  - [x] `writeMemory` adapter calls `PUT /api/memory/:filename` with body
  - [x] `consolidateMemory` adapter calls `POST /api/memory/consolidate` with workspace
  - [x] Adapter throws typed `KnowledgeApiError` on non-2xx responses
  - [x] `useMemories` hook returns loading state then data
  - [x] `useDeleteMemory` mutation invalidates memory list cache on settle
  - [x] Query options factory includes correct staleTime and refetchInterval
  - [x] Zod schema validates MemoryHeader with required name and type fields
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make web-lint && make web-typecheck` passes
- System exports are importable from `@/systems/knowledge`
- Hooks return correct data when connected to running daemon
