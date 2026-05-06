---
status: completed
title: "Task Context Bundle and Review Continuation Redaction"
type: backend
complexity: high
dependencies:
  - task_10
  - task_12
  - task_22
---

# Task 23: Task Context Bundle and Review Continuation Redaction

## Overview
This task creates the bounded `TaskContextBundle` used by task sessions and continuation runs. It must deliver profile, current run, review lineage, redacted prior-turn context, and next-round guidance without leaking raw claim tokens or unrestricted transcript state.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST define and populate a bounded task context bundle at session start.
- MUST redact review continuation context and raw claim/lease tokens.
- MUST include review lineage and next-round guidance for continuation runs.
</requirements>

## Subtasks
- [x] Define task context bundle domain/contract boundaries.
- [x] Populate bundle data from task, run, profile, review, and notification read models.
- [x] Apply redaction rules for continuation and reviewer context.
- [x] Thread bundle into worker/reviewer/coordinator session starts.
- [x] Add bundle size, redaction, continuation, and session-start tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/task` - task/session start authority and domain types.
- `internal/daemon/task_runtime.go` - session bridge context handoff.
- `internal/session` - session creation metadata.
- `internal/store/globaldb` - task/run/review read models.

### Dependent Files
- `internal/skills/bundled/skills/agh-task-worker/SKILL.md` - worker context expectations.
- `internal/skills/bundled/skills/agh-task-reviewer/SKILL.md` - reviewer context expectations.
- `internal/api/core/task_reviews.go` - review context surfaces.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Creates a stable context bundle primitive for skills, tools, and web read models.
- Agent manageability: Agents receive bounded context through runtime sessions and can inspect associated state through API/CLI surfaces.
- Config lifecycle: Bundle behavior uses existing profile/review config; no new config keys expected.

### Web/Docs Impact
- `web/`: Bundle summary/read model feeds task detail and review queue UI in tasks 26 and 27.
- `packages/site`: Context/redaction behavior documented in tasks 28 and 29.

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
- Memory: `memory/task_23.md`.
- Focused validation:
  - `go test ./internal/situation -run 'Test(ContextForSession|ContextBundle)' -count=1`
  - `go test ./internal/daemon -run 'Test(TaskSessionBridgeStartTaskSessionInjectsTaskContextOverlay|ReviewRouterRoutesRunReviewRequests|CoordinatorRuntimeBootstrapsWithTaskContextOverlay)' -count=1`
  - `go test ./internal/situation -count=1`
  - `go test ./internal/daemon -count=1`
- Contract co-ship validation:
  - `make codegen`
  - `make codegen-check`
  - `make web-typecheck`
  - `make web-test`
  - `cd packages/site && bun run typecheck`
  - `cd packages/site && bun run test`
  - `cd packages/site && bun run build`
- Final gate: `make verify` passed with Bun lint/typecheck/test, web build, `golangci-lint` 0 issues, Go race gate `DONE 8276 tests in 130.651s`, and package boundaries OK.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
