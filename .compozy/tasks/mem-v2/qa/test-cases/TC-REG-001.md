# TC-REG-001: Codegen, CLI/API Docs, Runtime Docs, And QA Artifact Guard

**Priority:** P0
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify generated contracts, generated CLI/API references, runtime docs, manual shell examples, and this QA dossier stay aligned with the final Memory v2 Slice 1 surface.

## Preconditions

- [ ] Working tree contains the task_25 QA artifacts.
- [ ] Generated OpenAPI and CLI docs are current from tasks 15 and 24.
- [ ] Bun dependencies are installed.

## Test Steps

1. **Run codegen check**
   - Input: `make codegen-check`
   - **Expected:** No OpenAPI or generated TypeScript drift.

2. **Run CLI docs regeneration**
   - Input: `make cli-docs`
   - **Expected:** Generated CLI docs remain aligned with Cobra source. Any table formatting drift is normalized by the lint/format stage before final diff assessment.

3. **Run focused site truth tests**
   - Input: `cd packages/site && bun run test -- runtime-docs-truth runtime-docs-discovery runtime-manual-cli-examples runtime-manual-api-routes memory-v2-qa-artifacts`
   - **Expected:** Tests pass and assert Memory v2 docs/reference truth plus QA dossier completeness.

4. **Run site typecheck/build**
   - Input: `cd packages/site && bun run typecheck` and `cd packages/site && bun run build`
   - **Expected:** Site compiles and builds runtime docs.

5. **Run final repo gate**
   - Input: `make verify`
   - **Expected:** Full monorepo gate passes after all task_26 fixes.

## Evidence To Capture

- `make codegen-check` log.
- `make cli-docs` log and post-format diff assessment.
- Focused site test log.
- Site build log.
- Final `make verify` log.

## Negative Assertions

- No generated CLI page for `memory/read` or `memory/consolidate`.
- No current API reference for `GET /api/memory/search` or `PUT /api/memory/{filename}`.
- No `[memory.v2]` config namespace.
- QA dossier contains no empty-shell case.
