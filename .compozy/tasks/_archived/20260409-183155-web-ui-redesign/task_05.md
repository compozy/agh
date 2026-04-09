---
status: completed
title: Build skill frontend system
type: frontend
complexity: medium
dependencies:
    - task_04
---

# Task 05: Build skill frontend system

## Overview

Create the `systems/skill/` module following the app-renderer-systems pattern: types, API adapter, query keys, query options, hooks, and public barrel export. This data layer provides typed access to the Skills HTTP endpoints (task_04) and feeds the Skills page (task_06).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Core Interfaces" section for adapter type definitions
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `systems/skill/types.ts` with Zod schemas for `SkillPayload`, `SkillSource`, and filter types
- MUST create `systems/skill/adapters/skill-api.ts` with `listSkills`, `getSkill`, `enableSkill`, `disableSkill` functions
- MUST create `systems/skill/lib/query-keys.ts` with hierarchical query key factory
- MUST create `systems/skill/lib/query-options.ts` with `queryOptions` factories for list and detail
- MUST create `systems/skill/hooks/use-skills.ts` with `useSkills` (list) and `useSkill` (detail) hooks
- MUST create `systems/skill/hooks/use-skill-actions.ts` with `useEnableSkill` and `useDisableSkill` mutation hooks
- MUST create `systems/skill/index.ts` barrel with explicit named exports
- MUST pass `AbortSignal` from query context through to every API call
- MUST use `staleTime: 30000` and `refetchInterval: 60000` for list queries
- MUST invalidate skill queries after enable/disable mutations via `onSettled`
</requirements>

## Subtasks
- [x] 5.1 Create Zod schemas and TypeScript types for skill domain
- [x] 5.2 Create skill API adapter with typed fetch functions and error class
- [x] 5.3 Create query keys and query options factories
- [x] 5.4 Create query hooks (`useSkills`, `useSkill`) and mutation hooks (`useEnableSkill`, `useDisableSkill`)
- [x] 5.5 Create barrel export `index.ts`
- [x] 5.6 Write tests for adapter functions and hooks

## Implementation Details

See TechSpec "Core Interfaces" section for the SkillPayload type definition and adapter signatures.

Follow the exact pattern established by `systems/agent/` — same file structure, naming conventions, and query infrastructure patterns. The agent system is the reference implementation.

### Relevant Files
- `web/src/systems/agent/` — Reference system to follow as template (adapters, lib, hooks, types, index)
- `web/src/systems/agent/adapters/agent-api.ts` — Reference adapter pattern
- `web/src/systems/agent/lib/query-keys.ts` — Reference query key pattern
- `web/src/systems/agent/lib/query-options.ts` — Reference query options pattern
- `web/src/systems/agent/hooks/use-agents.ts` — Reference hook pattern
- `web/src/systems/agent/types.ts` — Reference Zod schema pattern

### Dependent Files
- `web/src/systems/skill/` — All files in this directory are new (created by this task)
- `web/src/routes/_app/skills.tsx` — Will import from this system (task_06)

### Related ADRs
- [ADR-003: Full Systems Architecture for Skills and Knowledge](../adrs/adr-003.md) — Mandates full data layer from day one

## Deliverables
- Complete `systems/skill/` directory with types, adapters, lib, hooks, and index
- Typed adapter functions for all 4 skill endpoints
- Query hooks with proper staleTime/refetchInterval configuration
- Mutation hooks with cache invalidation
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `listSkills` adapter calls `GET /api/skills?workspace=:id` and returns typed array
  - [ ] `getSkill` adapter calls `GET /api/skills/:name?workspace=:id` and returns typed object
  - [ ] `enableSkill` adapter calls `POST /api/skills/:name/enable` and returns `{ok: true}`
  - [ ] `disableSkill` adapter calls `POST /api/skills/:name/disable` and returns `{ok: true}`
  - [ ] Adapter throws typed `SkillApiError` on non-2xx responses
  - [ ] `useSkills` hook returns loading state then data
  - [ ] `useEnableSkill` mutation invalidates skill list cache on settle
  - [ ] Query options factory includes correct staleTime and refetchInterval
  - [ ] Query keys are hierarchically structured (skill.list, skill.detail)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make web-lint && make web-typecheck` passes
- System exports are importable from `@/systems/skill`
- Hooks return correct data when backend endpoints are available
