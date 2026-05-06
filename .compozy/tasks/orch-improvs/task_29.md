---
status: completed
title: "Site Docs for Review Gate, Bundled Skills, and Notification Cursors"
type: docs
complexity: high
dependencies:
  - task_19
  - task_25
  - task_27
---

# Task 29: Site Docs for Review Gate, Bundled Skills, and Notification Cursors

## Overview
This task authors narrative `packages/site` documentation for review gates, bundled orchestration skills, and notification cursors. It must explain post-terminal review authority, reviewer routing/binding, continuation runs, bundled skill expectations, bridge subscriptions, cursor diagnostics, and agent-operable surfaces.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST document review request, reviewer binding, verdict outcomes, rejected continuation, and ReviewRouter behavior.
- MUST document bundled skills and native tools without calling capabilities recipes or workflows.
- MUST document bridge notification subscriptions, cursor replay, diagnostics, and SSE resume behavior.
</requirements>

## Subtasks
- [x] Author review-gate concept and how-to docs with CLI/API/native examples.
- [x] Author bundled orchestration skill docs and tool usage boundaries.
- [x] Author notification cursor and bridge subscription lifecycle docs.
- [x] Update navigation/source metadata and generated reference links.
- [x] Run site source generation, typecheck, build, CLI docs if needed, and full verify.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `packages/site/content/runtime` - narrative docs location.
- `packages/site/content/runtime/cli-reference/task/review/` - generated review CLI docs.
- `packages/site/content/runtime/cli-reference/task/` - notification CLI docs.
- `internal/skills/bundled/skills` - bundled skill source.

### Dependent Files
- `COPY.md` - claim standards and naming.
- `docs/_memory/glossary.md` - capability vocabulary.
- `packages/site/CLAUDE.md` - site rules.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursors are monotonic and replay-safe.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Documents bundled skills, native tools, bridge subscriptions, and notification cursors as extensibility surfaces.
- Agent manageability: Docs must show CLI/HTTP/UDS/native/web paths for review and notification management.
- Config lifecycle: Documents review profile/config lifecycle where applicable.

### Web/Docs Impact
- `web/`: References review/notification UI from task 27.
- `packages/site`: Primary owner: new/updated `packages/site/content/runtime/**` MDX pages.

## Deliverables
- Task implementation or documentation matching the requirements above.
- Focused unit tests with 80%+ coverage where code changes.
- Integration, contract, e2e, or docs-build tests proportional to the touched behavior.
- Updated workflow memory, QA evidence, generated artifacts, or site docs when applicable.

## Tests
- Unit tests:
  - [x] Validate the primary success path for this task.
  - [x] Validate malformed input, missing dependency, or authorization failure paths.
  - [x] Validate boundary conditions named by the related TechSpec and ADRs.
- Integration tests:
  - [x] Exercise the task through the owning service/transport boundary when applicable.
  - [x] Compare persisted state, generated contract output, or rendered docs/UI with runtime truth.
  - [x] Run race, codegen, site, web, or full verify gates listed by the touched surface.
- Test coverage target: >=80% for changed code paths; docs-only tasks require 100% checklist evidence against authored pages.
- All tests must pass.

## Completion Evidence
- Implemented review-gate docs in `packages/site/content/runtime/core/autonomy/review-gate.mdx`, documenting review authority, reviewer binding, verdict outcomes, rejected continuations, native/API/CLI surfaces, web read-only surfaces, and the implemented review event taxonomy.
- Implemented notification cursor docs in `packages/site/content/runtime/core/autonomy/notification-cursors.mdx`, documenting bridge subscriptions, cursor lifecycle, diagnostics, SSE resume seeding, and the public reset boundary.
- Updated bundled skill docs in `packages/site/content/runtime/core/skills/bundled.mdx` for `agh-orchestrator`, `agh-task-worker`, and `agh-task-reviewer` as instructional-only bundled guidance, not capabilities/recipes/workflows.
- Updated Autonomy navigation in `packages/site/content/runtime/core/autonomy/index.mdx` and `meta.json`.
- Extended `packages/site/lib/runtime-autonomy-docs.test.ts` to cover the new pages and to reject previously-audited false claims.
- Split blog/changelog static metadata into lightweight modules so site metadata tests no longer import heavy route component/content trees.
- Validation:
  - `cd packages/site && bun run source:generate` PASS.
  - `cd packages/site && bun run content:generate` PASS.
  - `cd packages/site && bun run typecheck` PASS.
  - `cd packages/site && bunx vitest run lib/runtime-autonomy-docs.test.ts` PASS, `1` file / `15` tests.
  - `cd packages/site && bunx vitest run lib/static-route-metadata.test.ts` PASS, `1` file / `3` tests.
  - `cd packages/site && bun run test` PASS, `74` files / `263` tests.
  - `cd packages/site && bun run build` PASS, SSG generated `1086` static pages.
  - `compozy tasks validate --name orch-improvs --format json` PASS, `scanned: 32`.
  - `git diff --check` PASS.
  - `make verify` PASS: Vitest monorepo `339` files / `2206` tests, `golangci-lint` `0 issues`, Go race gate `DONE 8283 tests in 32.697s`, boundaries OK.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
