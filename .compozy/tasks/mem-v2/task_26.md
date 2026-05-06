---
status: completed
title: Real-Scenario QA Execution
type: test
complexity: critical
dependencies:
  - task_25
---

# Task 26: Real-Scenario QA Execution

## Overview

Execute the release-grade Memory v2 QA pass against a realistic daemon/workspace/web environment. This task runs the scenario plans, captures bugs as artifacts, fixes root causes when needed, reruns the affected gates, and leaves a verification report showing the final system behavior across runtime, transports, UI, docs, and config.

<critical>
- ALWAYS READ `qa/test-plans/*` and `qa/test-cases/*` before executing.
- REFERENCE the TechSpec, ADRs, and QA plan artifacts instead of inventing ad-hoc scope.
- ACTIVATE `real-scenario-qa`, `qa-execution`, and `agh-worktree-isolation` before execution.
- USE `browser-use:browser` for the highest-risk UI checks; fall back to `agent-browser` only if necessary.
- TESTS REQUIRED: runtime E2E, web E2E, CLI/HTTP/UDS parity, docs truth checks, and config lifecycle scenarios must run.
- NO WORKAROUNDS: every reproduced defect must become a root-cause fix and a re-run, not a waived expectation.
</critical>

<requirements>
- MUST execute the Memory v2 QA plan in an isolated runtime environment with unique `AGH_HOME`, ports, and sockets as needed.
- MUST run runtime, transport, web, docs, and config lifecycle verification appropriate to the changed surfaces.
- MUST file `qa/issues/BUG-NNN.md` for every reproduced defect and rerun after fixes.
- MUST regenerate or rerun any required codegen/docs/build steps touched by fixes.
- MUST produce a final `qa/verification-report.md` with commands run, evidence, and outcomes.
</requirements>

## Subtasks
- [x] 26.1 Bootstrap the isolated QA environment and run the planned runtime/transport scenarios.
- [x] 26.2 Execute web and operator-facing scenarios, including browser-driven checks for the changed UI.
- [x] 26.3 File bugs for reproduced defects, fix root causes, and rerun affected scenarios/gates.
- [x] 26.4 Produce the final verification report with evidence and outcomes.

## Implementation Details

This task consumes the QA plan from `task_25` and is the final gate for the Memory v2 program. It should leave a machine-readable QA artifact set plus rerun evidence for any fixes required during execution.

### Relevant Files
- `.compozy/tasks/mem-v2/task_25.md` — planning task that defines execution scope.
- `qa/test-plans/**` — execution plan inputs.
- `qa/test-cases/**` — scenario definitions.
- `qa/issues/**` — required bug artifact location for any reproduced defect.
- `qa/verification-report.md` — final verification output.

### Dependent Files
- Any implementation task outputs fixed during QA reruns.
- `make verify` — final monorepo verification gate.
- `make test-e2e-runtime` and `make test-e2e-web` — required end-to-end lanes for this feature set.

### Related ADRs
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md)
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md)
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md)
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md)

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: execution must cover provider, extension host, and builtin-tool memory surfaces end-to-end.
- Agent manageability: execution must cover CLI, HTTP, UDS, structured outputs, deterministic errors, and parity scenarios.
- Config lifecycle: execution must cover config defaults, settings updates, validation, restart effects, and docs/reference truth checks.

### Web/Docs Impact

- `web/`: execution must cover knowledge, settings/memory, and session inspector scenarios through the real UI.
- `packages/site`: execution must cover runtime docs truth, API/CLI references, and discoverability checks after any fixes.

## Deliverables

- Executed QA scenarios with recorded evidence.
- `qa/issues/BUG-NNN.md` artifacts for any reproduced defect.
- Root-cause fixes and rerun evidence for any failed scenario.
- Final `qa/verification-report.md`.

## Tests

- Unit tests:
  - [x] N/A — this task executes integrated scenario coverage rather than adding isolated unit cases.
- Integration tests:
  - [x] `make test-e2e-runtime` passes for the Memory v2 runtime scenarios.
  - [x] `make test-e2e-web` passes for the changed UI surfaces.
  - [x] CLI/HTTP/UDS parity scenarios pass against the final runtime.
  - [x] Docs truth/discoverability checks pass after all changes and fixes.
  - [x] `make verify` passes at the end of the QA loop.
- Test coverage target: release-grade cross-surface verification.
- All tests must pass.

## References

- `.agents/skills/qa-execution/SKILL.md`
- `.agents/skills/real-scenario-qa/SKILL.md`
- `.agents/skills/agh-worktree-isolation/SKILL.md`
- `.agents/skills/agent-browser/SKILL.md`

## Success Criteria

- All planned Memory v2 scenarios execute with recorded evidence.
- Every reproduced defect is fixed at root cause and rerun.
- `qa/verification-report.md` shows the final release-grade verification state for Slice 1.
