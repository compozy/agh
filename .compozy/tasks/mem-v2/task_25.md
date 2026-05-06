---
status: completed
title: QA Plan and Test Coverage
type: test
complexity: high
dependencies:
  - task_24
---

# Task 25: QA Plan and Test Coverage

## Overview

Produce the release-grade QA plan for the full Memory v2 Slice 1 program. This task translates the approved TechSpec, ADRs, task graph, and implemented surfaces into executable test plans, regression matrices, and scenario coverage for runtime, CLI, HTTP/UDS, web, docs, and extensibility behavior.

<critical>
- ALWAYS READ `_techspec.md`, every ADR, and every completed implementation task before drafting test artifacts.
- REFERENCE the TechSpec invariants, delete targets, public interfaces, and QA proofs instead of inventing new scope.
- ACTIVATE `qa-report` before writing QA plans or test matrices.
- MINIMIZE speculation: base the plan on actual implemented surfaces and the final task outputs.
- TESTS REQUIRED: plan coverage must include happy path, failure path, concurrency, redaction, transport parity, and real operator flows.
- NO WORKAROUNDS: do not produce a thin or “fraco” test plan for a cross-cutting runtime change.
</critical>

<requirements>
- MUST generate QA plans and test cases that cover every public Memory v2 surface touched by tasks 01-24.
- MUST include CLI, HTTP, UDS, native-tool, extension-host, web, docs, and config lifecycle verification.
- MUST identify regression hot spots from controller, recall, extractor, dreaming, ledger, and workspace-identity invariants.
- MUST define the real-scenario test data, workspace setup, and isolation requirements for execution.
- MUST call out any required codegen/build/doc-generation steps in the QA plan.
</requirements>

## Subtasks
- [x] 25.1 Produce the cross-surface QA plan and scenario matrix for Memory v2.
- [x] 25.2 Produce detailed test cases covering runtime, transports, UI, docs, and config lifecycle.
- [x] 25.3 Define regression hot spots and negative/concurrency/redaction checks.
- [x] 25.4 Prepare the execution prerequisites for isolated real-scenario QA.

## Implementation Details

Use the canonical QA tail pattern for AGH task packs. The output should prepare `qa/test-plans/`, `qa/test-cases/`, and any supporting scenario artifacts needed by `task_26`.

### Relevant Files
- `.compozy/tasks/mem-v2/_techspec.md` — normative behavior and invariants.
- `.compozy/tasks/mem-v2/_tasks.md` — final execution graph to cover in QA.
- `internal/api/httpapi/transport_parity_integration_test.go` — transport parity hotspot.
- `internal/api/udsapi/transport_parity_integration_test.go` — transport parity hotspot.
- `web/src/routes/_app/-knowledge.test.tsx` — web knowledge verification hotspot.
- `web/src/routes/_app/settings/-memory.test.tsx` — web settings verification hotspot.

### Dependent Files
- `.compozy/tasks/mem-v2/task_26.md` — execution task consumes the plans and cases produced here.
- `qa/test-plans/**` — expected output location for QA planning artifacts.
- `qa/test-cases/**` — expected output location for executable QA cases.

### Related ADRs
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md)
- [ADR-006: Session Ledger Hybrid (events.db Live + ledger.jsonl Forensic)](adrs/adr-006.md)
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md)
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md)

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: the QA plan must cover provider registry, extension host, and builtin-tool memory surfaces.
- Agent manageability: the QA plan must cover CLI, HTTP, UDS, structured outputs, deterministic errors, and parity across these surfaces.
- Config lifecycle: the QA plan must cover memory config defaults, validation, settings UI/backend parity, and docs/reference consistency.

### Web/Docs Impact

- `web/`: QA planning must cover knowledge, settings/memory, and session inspector flows.
- `packages/site`: QA planning must cover runtime docs truth, API/CLI references, and discoverability checks.

## Deliverables

- `qa/test-plans/` artifacts covering the full Memory v2 program.
- `qa/test-cases/` artifacts for runtime, transport, UI, docs, and config scenarios.
- Regression hotspot list covering concurrency, redaction, replay, parity, and failure handling.
- Execution prerequisites for isolated real-scenario QA.

## Tests

- Unit tests:
  - [x] QA artifacts enumerate every implemented public memory surface and major invariant.
- Integration tests:
  - [x] QA plan includes runtime, CLI, HTTP/UDS, native-tool, extension-host, web, docs, and config lifecycle coverage.
  - [x] QA plan includes negative, concurrency, redaction, and restart/replay scenarios.
- Test coverage target: complete behavioral coverage planning for tasks 01-24.
- All tests must pass.

## References

- `.agents/skills/qa-report/SKILL.md`
- `.agents/skills/real-scenario-qa/SKILL.md`
- `.agents/skills/agh-worktree-isolation/SKILL.md`

## Success Criteria

- QA planning artifacts exist and cover every Memory v2 surface implemented in tasks 01-24.
- The plan is detailed enough for execution without reopening architecture questions.

## Completion Notes

- Added `.compozy/tasks/mem-v2/qa/test-plans/memory-v2-test-plan.md`, `memory-v2-regression.md`, and `memory-v2-traceability.md`.
- Added twelve executable QA cases under `.compozy/tasks/mem-v2/qa/test-cases/`, including P0 TC-SCEN-001 for controller-backed write/search visibility without undocumented reindex.
- Added `packages/site/lib/memory-v2-qa-artifacts.test.ts` to guard task-to-scenario coverage, public-surface coverage, search-visibility risk mapping, and execution-ready case structure.
- Validation passed: focused site QA artifact test, focused site docs/reference/QA artifact tests, site typecheck, site build, `make codegen-check`, `git diff --check`, and no-workarounds scan over task_25 artifacts/test.
