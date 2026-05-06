---
status: completed
title: "Site Docs for Orchestration Profiles and Configuration"
type: docs
complexity: high
dependencies:
  - task_18
  - task_27
---

# Task 28: Site Docs for Orchestration Profiles and Configuration

## Overview
This task authors narrative `packages/site` documentation for task orchestration profiles and configuration. It must complement generated CLI reference pages with concept, how-to, API/CLI examples, config lifecycle notes, and truthful web UI references.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST document `[task.orchestration]`, execution profiles, selector precedence, sandbox modes, and worker runtime selection.
- MUST include CLI, HTTP, UDS, native tool, and web UI management paths.
- MUST co-ship generated references and site build/typecheck evidence.
</requirements>

## Subtasks
- [x] Read `packages/site/CLAUDE.md`, `COPY.md`, `DESIGN.md`, and glossary before writing docs.
- [x] Author concept/how-to docs for execution profiles and profile-driven worker/runtime selection.
- [x] Add API/CLI/native tool examples that match generated contracts.
- [x] Update docs navigation/source metadata as required by Fumadocs.
- [x] Run site source generation, typecheck, build, and full verify.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `packages/site/content/runtime` - runtime documentation pages.
- `packages/site/content/runtime/cli-reference/task/profile/` - generated CLI reference.
- `openapi/agh.json` - HTTP contract source.
- `web/src/systems/tasks` - UI behavior to document.

### Dependent Files
- `packages/site/CLAUDE.md` - site-specific rules.
- `COPY.md` - product language.
- `docs/_memory/glossary.md` - canonical terms.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Documents profiles as an extensible runtime capability rather than an internal implementation detail.
- Agent manageability: Docs must show profile management through CLI/HTTP/UDS/native/web surfaces.
- Config lifecycle: Documents config keys, defaults, validation, and examples from task 01 and profile runtime tasks.

### Web/Docs Impact
- `web/`: References truthful profile UI from task 27.
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
- Authored `packages/site/content/runtime/core/autonomy/execution-profiles.mdx` covering profile shape, selector precedence, sandbox modes, worker runtime selection, CLI/HTTP/UDS/native/web management paths, config lifecycle, and authority boundary.
- Updated `packages/site/content/runtime/core/autonomy/index.mdx` and `meta.json` so the profiles page is listed in autonomy navigation.
- Extended `packages/site/content/runtime/core/configuration/config-toml.mdx` with `[task.orchestration]`, `[task.orchestration.profile]`, and `[task.orchestration.review]` reference tables, defaults, and validation timing.
- Extended `packages/site/lib/runtime-autonomy-docs.test.ts` with checklist evidence for the authored execution-profiles page (precedence, native tools, web UI, config lifecycle) and for the generated `agh task profile` CLI references.
- Verified docs against runtime truth in `internal/task/profile.go`, `internal/task/manager.go`, `internal/task/manager_profile.go`, `internal/daemon/task_runtime.go`, `internal/config/task_orchestration.go`, and `openapi/agh.json` (operations `getTaskExecutionProfile`, `setTaskExecutionProfile`, `deleteTaskExecutionProfile`).
- Site validation: `bun run source:generate` PASS, `bun run content:generate` PASS, `bun run typecheck` PASS, `bun run test` PASS (74 files / 256 tests), `bun run build` PASS.
- Repo validation: `compozy tasks validate --name orch-improvs --format json` PASS (32 valid task files), `git diff --check` PASS, `make verify` PASS — Go race gate `DONE 8283 tests in 19.010s`, `OK: all package boundaries respected`.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
