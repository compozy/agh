---
status: completed
title: "QA Plan and Test Coverage"
type: test
complexity: high
dependencies:
  - task_11
---

# Task 12: QA Plan and Test Coverage

## Overview
This task creates the release-grade QA plan for the provider model catalog program. It enumerates concrete test cases for every public surface, config lifecycle path, extension path, stale/failure mode, and ACP session config behavior implemented by Tasks 01-11.

<critical>
- ALWAYS READ `_techspec.md`, every ADR, and every completed task before drafting test cases.
- REFERENCE TECHSPEC for implementation details - do not duplicate here.
- FOCUS ON "WHAT" - describe what needs to be validated, not how to implement missing code.
- MINIMIZE CODE - do not write production code in QA planning artifacts.
- TESTS REQUIRED - QA plans must cover unit, integration, e2e, and regression behaviors proportionally.
</critical>

<requirements>
- MUST activate the `qa-report` skill before writing QA artifacts.
- MUST create QA artifacts under `.compozy/tasks/provider-model-catalog/qa/`.
- MUST cover config hard cut, global DB migrations, catalog merge/stale/availability behavior, live sources, ACP config options, HTTP, UDS, CLI, `/api/openai/v1/models`, Host API, web, generated docs, and extension source contracts.
- MUST include negative tests, edge cases, refresh stress/SQLite write-contention, redaction checks, and cross-surface parity checks.
- MUST include UI-bearing E2E coverage and CLI/API/agent-manageability E2E coverage.
- MUST define bug report format and verification evidence requirements for Task 13.
</requirements>

## Subtasks
- [x] 12.1 Build a QA coverage matrix mapping TechSpec invariants and tasks to concrete tests.
- [x] 12.2 Write operator and agent scenarios for config, catalog, CLI, HTTP, UDS, Host API, and web.
- [x] 12.3 Write failure-mode scenarios for stale sources, timeouts, redaction, unavailable providers, extension denial, refresh coalescing, and SQLite write contention.
- [x] 12.4 Write browser E2E scenarios for provider settings and new session model selection.
- [x] 12.5 Write verification commands, isolated QA environment requirements, and bug-report templates.

## Implementation Details
Use the `qa-report` skill and the TechSpec `Testing Approach` / `Safety Invariants` sections. Include the required QA pair scope from `.agents/skills/cy-tasks-tail-qa-pair/references/hermes-tail-template.md`.

### Relevant Files
- `.compozy/tasks/provider-model-catalog/_techspec.md` - source of invariants and test obligations.
- `.compozy/tasks/provider-model-catalog/adrs/` - decision context for QA coverage.
- `.compozy/tasks/provider-model-catalog/task_01.md` through `task_11.md` - implementation task scope.
- `.agents/skills/qa-report/SKILL.md` - QA plan skill instructions.
- `.agents/skills/cy-tasks-tail-qa-pair/references/hermes-tail-template.md` - required QA tail scope.

### Dependent Files
- `.compozy/tasks/provider-model-catalog/qa/test-plans/` - QA plan output directory.
- `.compozy/tasks/provider-model-catalog/qa/test-cases/` - concrete test cases.
- `.compozy/tasks/provider-model-catalog/qa/issues/` - bug report output directory used by Task 13.
- `.compozy/tasks/provider-model-catalog/qa/verification-report.md` - execution report target for Task 13.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - authority and stale/source behavior.
- [ADR-002: Provider Model Config Hard Cut](adrs/adr-002-provider-model-config-hard-cut.md) - no-compat config QA.
- [ADR-003: Extension Model Source Contract](adrs/adr-003-extension-model-source-contract.md) - extension QA.

## Deliverables
- QA coverage matrix for Tasks 01-11.
- QA test plan and concrete test cases under `qa/test-plans/` and `qa/test-cases/`.
- Bug report template and verification report template.
- Explicit commands for unit, integration, E2E runtime, E2E web, codegen, docs, and full verify **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] QA plan maps every TechSpec safety invariant to at least one test case.
  - [x] QA plan maps every public surface to at least one parity or contract test.
  - [x] QA plan includes old-key rejection and nested config lifecycle cases.
  - [x] QA plan includes redaction checks for source errors in logs/API/web-visible payloads.
  - [x] QA plan includes concurrent `POST /api/providers/models/refresh` stress across providers and repeated same-provider refresh coalescing.
- Integration tests:
  - [x] QA plan includes daemon-served CLI/HTTP/UDS parity scenario.
  - [x] QA plan includes `/api/openai/v1/models` auth, HTTP-only registration, OpenAI-shaped error, and provider filter scenarios.
  - [x] QA plan includes extension `model.source` success and denial scenarios.
  - [x] QA plan includes browser workflow for Settings > Providers and new session model selection.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- QA plan covers every task and TechSpec invariant without placeholder or vague test cases.
- Task 13 can execute the plan without inventing missing scenarios.
