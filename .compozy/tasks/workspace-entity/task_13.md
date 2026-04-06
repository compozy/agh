---
status: completed
title: Web workspace UI and session grouping
type: frontend
complexity: high
dependencies:
  - task_10
---

# Task 13: Web workspace UI and session grouping

## Overview

Update the React SPA (`web/`) to add a workspace selector, group or filter sessions by workspace, and consume the new REST endpoints from task_10. Follow the `web/src/systems/` pattern (session system, adapters, hooks) per project conventions.

<critical>
- READ `_techspec.md` web impact and `web/CLAUDE.md` if present
- USE `app-renderer-systems` skill patterns for systems modules
- TESTS REQUIRED — Vitest for adapters/hooks; integration tests where project already uses them
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST add workspace API client functions (TanStack Query) alongside existing session API
- MUST add UI to pick or switch workspace before/during session creation flows
- MUST display `workspace_id` (and human name) in session list/detail per UX decision (minimal: filter + badge)
- MUST update TypeScript types in `web/src/systems/session/types.ts` for new session payloads
- MUST add or update tests: `session-api.test.ts`, hook tests, and any affected components
- MUST run `web` lint/test scripts as defined in repo Makefile or `web/package.json`
</requirements>

## Subtasks
- [x] 13.1 Add `systems/workspace/` (or extend session system) with queries for list/resolve
- [x] 13.2 Extend session create flow to send `workspace` / `workspace_path` fields
- [x] 13.3 Add workspace selector component and wire to router layout
- [x] 13.4 Group or filter session sidebar by workspace
- [x] 13.5 Update tests and storybook/stories only if required by project for touched components

## Implementation Details

See TechSpec "web/" impact. Reference `web/src/systems/session/adapters/session-api.ts` and hooks. Avoid generic AI UI — match existing Tailwind/shadcn tokens.

### Relevant Files
- `web/src/systems/session/adapters/session-api.ts` — HTTP client
- `web/src/systems/session/types.ts` — Shared types
- `web/src/systems/session/hooks/use-sessions.ts` — List queries
- `web/src/systems/session/components/session-sidebar-item.tsx` — May show workspace

### Dependent Files
- `internal/httpapi/` — REST contract (task_10)

## Deliverables
- Workspace-aware web UI with updated API consumption
- Vitest coverage for new adapters/hooks **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Workspace list query parses response shape from `GET /api/workspaces`
  - [x] Session create payload includes workspace field when selector chosen
  - [x] Type guards or zod schemas reject incomplete API responses
- Integration tests:
  - [x] Update `chat-view.integration.test.tsx` or similar if session creation is exercised
- Test coverage target: maintain or improve package coverage for touched `web/src/systems/*` modules
- All tests must pass (`pnpm test` or Makefile target used by CI)

## Success Criteria
- All frontend tests passing
- `make verify` passes including web checks if part of default verify target
- No TypeScript errors; linter clean
