---
status: completed
title: "Web Data Layer for Orchestration, Review, and Notifications"
type: frontend
complexity: high
dependencies:
  - task_18
  - task_19
  - task_23
  - task_24
  - task_25
---

# Task 26: Web Data Layer for Orchestration, Review, and Notifications

## Overview
This task builds the web data layer for task orchestration surfaces using generated OpenAPI types. It must add adapters, query keys/options, hooks, fixtures, and tests for execution profiles, review state, task context bundles, SSE resume data, and notification diagnostics.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST consume generated OpenAPI types instead of hand-written duplicate DTOs.
- MUST add TanStack Query hooks for profiles, reviews, context bundles, streams, and notifications.
- MUST include MSW fixtures and tests for loading, empty, error, and mutation states.
</requirements>

## Subtasks
- [x] Read `web/CLAUDE.md`, `DESIGN.md`, and generated OpenAPI types before implementation.
- [x] Add task-system adapters and query/mutation hooks for profile and review surfaces.
- [x] Add context bundle, latest event sequence, and notification diagnostics data access.
- [x] Add MSW fixtures and focused hook tests.
- [x] Run Bun lint, typecheck, tests, web build, and full verify.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `web/src/systems/tasks` - task system data layer and hooks.
- `web/src/generated/agh-openapi.d.ts` - generated contract types.
- `web/src/test` or task MSW fixtures - data-layer mocks.
- `internal/api/spec/spec.go` - source of generated contract.

### Dependent Files
- `web/CLAUDE.md` - web-specific rules.
- `DESIGN.md` - UI grammar for follow-up components.
- `COPY.md` - product-language rules for visible strings.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursor and replay semantics.
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run projection boundaries.
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - review request/verdict/continuation authority.
- [ADR-010: Typed Overlay](adrs/adr-010.md) - execution profile schema and config overlay shape.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Surfaces runtime task orchestration through the web client data system.
- Agent manageability: Web becomes an operator/agent-assist management surface backed by the same API contracts.
- Config lifecycle: No new config keys; display effective profile/config-derived state truthfully.

### Web/Docs Impact
- `web/`: Primary owner: `web/src/systems/tasks/**` data layer, hooks, fixtures, and tests.
- `packages/site`: Generated types and docs references feed tasks 28 and 29.

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
- New web orchestration surfaces are implemented under `web/src/systems/tasks/` and consume only generated OpenAPI types from `web/src/lib/api-contract.ts` and `web/src/generated/agh-openapi.d.ts`.
- Adapters added: `getTaskExecutionProfile`, `setTaskExecutionProfile`, `deleteTaskExecutionProfile`, `listTaskRunReviews`, `listTaskReviews`, `requestTaskRunReview`, `getTaskRunReview`, `submitTaskRunReviewVerdict`, `getAgentContext`, `getTaskContextBundle`, `listTaskBridgeNotificationSubscriptions`, `createTaskBridgeNotificationSubscription`, `getTaskBridgeNotificationSubscription`, `deleteTaskBridgeNotificationSubscription`. All thread `AbortSignal` and map 404/409 to typed `TasksApiError`.
- Query keys, query options, and React Query read/mutation hooks ship for every adapter; mutation invalidation reaches related task detail/timeline/run/dashboard/inbox/agent-context/context-bundle keys.
- Cursor-seeded SSE access ships as `buildTaskStreamUrl` (typed URL builder over `TaskStreamFilter.after_sequence`) and `useTaskStream` (cursor-seeded EventSource hook with optional `eventSourceFactory`, `onEvent`/`onError` callbacks, SSR/disabled/empty-id guards, parses generated `TaskStreamPayload`, invalidates task detail/timeline/runs/run-detail/lists/dashboard/inbox/context-bundle/agent-context/reviews/bridge-notification roots, closes the source on cleanup). Named-SSE correction: `useTaskStream` now also registers `addEventListener("<task event type>", ...)` for every task/orchestration/review/notification event emitted by `internal/api/core/sse.go` (`WriteTaskStreamEvent` sets `event: <task event type>`), since EventSource never routes named events to `onmessage`. Cleanup removes those named listeners when `removeEventListener` exists and always closes the source.
- MSW fixtures and handlers added for all new operations; fixtures cover populated and zero-state cursor diagnostics, and verdict continuation lineage.
- Tests added: `tasks-api-orchestration.test.ts` (adapter), extended `query-keys.test.ts` and `query-options.test.ts`, extended `fixtures.test.ts`, new hook tests `use-task-orchestration-hooks.test.tsx` (adapter mock) and `use-task-orchestration-msw.test.tsx` (real adapter+query stack against MSW handlers), and `use-task-stream.test.tsx` (URL builder + cursor-seeded EventSource lifecycle, named-SSE listener registration + parse path covering `task.run_started`/`task.run_review_requested`/`task.notification_delivered`, defensive unnamed-message fallback through `onmessage`, named-SSE malformed payload routes through `onError`, cleanup removes named listeners when `removeEventListener` exists and still closes the source when it does not, error event forwarding, factory reuse on identical inputs).
- Verification evidence: delegated web gates passed (`make web-lint`, `make web-typecheck`, `make web-build`, web Vitest). Local Codex audit then ran full `make verify` without reduced parallelism or environment overrides: Bun/Vitest `332 passed (332)` / `2150 passed (2150)`, web build PASS, `golangci-lint` `0 issues`, Go race gate `DONE 8283 tests in 75.399s`, package boundaries OK. Stream-hook correction reran web gates (`make web-lint` 0/0, `make web-typecheck` PASS, `make web-test` `206 passed (206)` / `1594 passed (1594)`, `make web-build` PASS) and full `make verify` (`333 passed (333)` / `2161 passed (2161)`, race gate `DONE 8283 tests in 61.850s`, boundaries OK). Named-SSE correction reran focused stream test (`vitest run src/systems/tasks/hooks/use-task-stream.test.tsx` `1 file / 13 tests passed`), `make web-lint` `0 warnings / 0 errors`, `make web-typecheck` PASS, `make web-test` `206 passed (206)` / `1596 passed (1596)`, `make web-build` PASS, full `make verify` `333 passed (333)` / `2163 passed (2163)`, race gate `DONE 8283 tests in 10.605s`, `OK: all package boundaries respected`. Final local Codex verification after audit normalization ran `make verify` on the current worktree: Bun/Vitest `333 passed (333)` / `2163 passed (2163)`, web build PASS, `golangci-lint` `0 issues`, Go race gate `DONE 8283 tests in 66.726s`, boundaries OK.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
