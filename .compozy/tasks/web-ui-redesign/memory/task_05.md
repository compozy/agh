# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Build `systems/skill/` module: types, adapter, query keys/options, hooks, barrel export, tests.

## Important Decisions

- `SkillApiError` is a typed error class with `status` property — follows task requirement for typed errors on non-2xx responses.
- Workspace is required for all skill API calls (matches backend contract: `workspace` query param is mandatory).
- `provenancePayloadSchema` uses `z.string()` for `installed_at` (not `z.coerce.date()`) to avoid serialization issues.
- `metadata` typed as `z.record(z.string(), z.unknown())` to match Go's `map[string]any`.

## Learnings

- Agent system adapter uses plain `Error` throws, but skill task requires typed `SkillApiError` — both patterns coexist fine.
- Query options `enabled` flag prevents fetches with empty workspace/name strings.

## Files / Surfaces

- `web/src/systems/skill/types.ts` — Zod schemas + TS types
- `web/src/systems/skill/adapters/skill-api.ts` — API adapter with SkillApiError
- `web/src/systems/skill/lib/query-keys.ts` — Hierarchical key factory
- `web/src/systems/skill/lib/query-options.ts` — queryOptions factories (staleTime 30s, refetchInterval 60s)
- `web/src/systems/skill/hooks/use-skills.ts` — useSkills, useSkill
- `web/src/systems/skill/hooks/use-skill-actions.ts` — useEnableSkill, useDisableSkill
- `web/src/systems/skill/index.ts` — Barrel export
- Tests: types.test.ts, skill-api.test.ts, query-keys.test.ts, query-options.test.ts, use-skills.test.tsx, use-skill-actions.test.tsx

## Errors / Corrections

- oxlint flagged unused `createWrapper` in `use-skill-actions.test.tsx` — removed (inline wrappers used instead for spy access).

## Ready for Next Run

Task complete. All verification gates pass (lint, typecheck, 448 tests).
