---
status: completed
title: "Durable Lessons and Glossary Alignment"
type: docs
complexity: medium
dependencies:
  - task_28
  - task_29
---

# Task 30: Durable Lessons and Glossary Alignment

## Overview
This task captures durable institutional lessons from the orchestration-improvements workstream. It must author numbered `docs/_memory/lessons/L-NNN-*.md` entries with concrete evidence and update the lessons index and glossary only where a canonical term or rule changed.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST create one or more numbered lesson files with confirmed root cause, fix, and evidence.
- MUST update `docs/_memory/lessons/README.md` with new lesson entries.
- MUST update `docs/_memory/glossary.md` only for real terminology alignment.
</requirements>

## Subtasks
- [x] Read existing lessons index, glossary, standing directives, and workflow memory.
- [x] Select durable lessons backed by evidence from specs, slices, reviews, or QA.
- [x] Author numbered lesson files with concrete paths and outcomes.
- [x] Update lessons index and glossary alignment if needed.
- [x] Run docs/site checks and full verify.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `docs/_memory/lessons` - durable lesson files.
- `docs/_memory/lessons/README.md` - lesson index.
- `docs/_memory/glossary.md` - canonical vocabulary.
- `.compozy/tasks/orch-improvs/memory` - evidence source.

### Dependent Files
- `docs/_memory/standing_directives.md` - rules not to duplicate.
- `packages/site` docs from tasks 28 and 29 - public wording alignment.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursor and replay semantics.
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run projection boundaries.
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - review request/verdict/continuation authority.
- [ADR-010: Typed Overlay](adrs/adr-010.md) - execution profile schema and config overlay shape.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Lessons preserve architectural decisions for future extensibility work.
- Agent manageability: Documents why every capability needs CLI/HTTP/UDS/native/web manageability when relevant.
- Config lifecycle: Captures config lifecycle lessons only if evidence supports a durable lesson.

### Web/Docs Impact
- `web/`: May cite task 27 UI truthfulness evidence.
- `packages/site`: Keeps public docs terminology aligned with memory/glossary.

## Deliverables
- Task implementation or documentation matching the requirements above.
- Focused unit tests with 80%+ coverage where code changes.
- Integration, contract, e2e, or docs-build tests proportional to the touched behavior.
- Updated workflow memory, QA evidence, generated artifacts, or site docs when applicable.

## Tests
- Unit tests:
  - [ ] Validate the primary success path for this task.
  - [ ] Validate malformed input, missing dependency, or authorization failure paths.
  - [ ] Validate boundary conditions named by the related TechSpec and ADRs.
- Integration tests:
  - [ ] Exercise the task through the owning service/transport boundary when applicable.
  - [ ] Compare persisted state, generated contract output, or rendered docs/UI with runtime truth.
  - [ ] Run race, codegen, site, web, or full verify gates listed by the touched surface.
- Test coverage target: >=80% for changed code paths; docs-only tasks require 100% checklist evidence against authored pages.
- All tests must pass.

## Completion Evidence
- Authored four numbered lessons under `docs/_memory/lessons/`:
  - `L-017-named-sse-listener-registration.md` (Frontend / SSE) — evidence: `web/src/systems/tasks/hooks/use-task-stream.ts`, `internal/api/core/sse.go:54-60`, `web/src/systems/tasks/hooks/use-task-stream.test.tsx`, `.compozy/tasks/orch-improvs/memory/task_26.md`.
  - `L-018-delegated-docs-runtime-truth-audit.md` (Documentation) — evidence: `.compozy/tasks/orch-improvs/memory/task_29.md`, `packages/site/content/runtime/core/autonomy/review-gate.mdx`, `packages/site/content/runtime/core/autonomy/notification-cursors.mdx`, `packages/site/lib/runtime-autonomy-docs.test.ts`, `packages/site/lib/static-route-metadata.test.ts`, blog/changelog metadata modules.
  - `L-019-diagnostic-data-outlives-primary-record.md` (Architecture / Persistence) — evidence: `.compozy/tasks/orch-improvs/adrs/adr-003-shared-durable-notification-cursors.md`, `internal/store/globaldb/global_db_bridge.go:985-1014`, `internal/store/globaldb/global_db_bridge_task_subscription_test.go:147-210`, `.compozy/tasks/orch-improvs/memory/task_25.md`.
  - `L-020-dense-typed-records-need-pointer-boundaries.md` (Architecture / Code style) — evidence: recurring `gocritic hugeParam` corrections across `.compozy/tasks/orch-improvs/memory/free-iter-020.md`, `free-iter-026.md`, `free-iter-030.md`, `free-iter-032.md`, `free-iter-036.md`, `.compozy/tasks/orch-improvs/memory/task_22.md`, and shared memory; `internal/task/lease.go` `Run.Review *RunReviewLineage`; profile pointer boundaries in `internal/daemon/native_profile_tools.go`, `internal/cli/client.go`, `internal/cli/task.go`, and `internal/api/contract/tasks.go`.
- Updated `docs/_memory/lessons/README.md` index with L-017 through L-020.
- Updated `docs/_memory/glossary.md` Autonomy section with seven canonical terms now load-bearing across runtime/contract/CLI/web/docs surfaces: Task Execution Profile, Notification Cursor, Bridge Task Subscription, Run Review, Continuation Run, Task Context Bundle, Current Run ID. Each entry restates the implemented authority/scope boundary.
- No race/full-gate flake notes were promoted to lessons because no confirmed root cause exists for them.
- The "truthful UI" posture from task 27 was rejected as a new lesson because Standing Directive `SD-007 — Truthful UI > Plausible UI` already covers it.

## Verification Evidence
- `compozy tasks validate --name orch-improvs --format json` PASS.
- `git diff --check` clean.
- `cd packages/site && bun run source:generate` PASS.
- `cd packages/site && bun run content:generate` PASS.
- `cd packages/site && bun run typecheck` PASS.
- `cd packages/site && bun run test` PASS.
- `cd packages/site && bun run build` PASS.
- `make verify` PASS.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
