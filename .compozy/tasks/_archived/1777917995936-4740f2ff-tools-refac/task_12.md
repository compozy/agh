---
status: completed
title: QA Plan and Test Coverage
type: test
complexity: high
dependencies:
  - task_11
---

# Task 12: QA Plan and Test Coverage

## Overview

Produce the release-grade QA plan for `tools-refac`, grounded in the final TechSpec, ADRs, and implementation outputs from tasks 01-11. The plan must cover runtime truth across tool, CLI, HTTP, UDS, hosted MCP, generated contracts, docs, and the web task-run fallout from the autonomy hard cut.

<critical>
- ALWAYS READ `_techspec.md`, every ADR, and every completed task artifact before drafting test cases
- REFERENCE TECHSPEC invariants and delete targets instead of inventing ad hoc QA scope
- FOCUS ON WHAT: build the verification plan and regression matrix; execution belongs to task_13
- MINIMIZE implementation detail in this task body; the output is QA artifacts under `qa/`
- TESTS REQUIRED — planning must include negative, concurrency, parity, redaction, and generated-surface coverage
</critical>

<requirements>
1. MUST produce `qa/test-plans/`, `qa/test-cases/`, and a regression matrix covering tasks 01-11.
2. MUST include end-to-end coverage for tool, CLI, HTTP, UDS, and hosted MCP management paths touched by the redesign.
3. MUST include contract/codegen coverage for `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, and `web/src/systems/tasks/mocks/fixtures.ts`.
4. MUST include docs/config lifecycle coverage for the rewritten runtime docs and generated CLI references.
5. MUST explicitly plan redaction, approval, policy, and concurrency stress cases around autonomy, hooks, extensions, and hosted MCP.
</requirements>

## Subtasks
- [x] 12.1 Map each implementation task to concrete QA scenarios and regression hot spots
- [x] 12.2 Produce negative and boundary cases for policy, approval, auth, and redaction behavior
- [x] 12.3 Produce transport-parity scenarios for tool, CLI, HTTP, UDS, and hosted MCP paths
- [x] 12.4 Produce docs/codegen/web-consumer verification scenarios for generated and published artifacts
- [x] 12.5 Save the plan under `qa/` with clear evidence and execution prerequisites for task_13

## Implementation Details

Use the QA tail pattern from the Hermes template and the TechSpec sections "Test Strategy", "E2E Tests", and "Monitoring and Observability". The output of this task is a concrete QA dossier that task_13 can execute without re-scoping the feature.

### Relevant Files
- `.compozy/tasks/tools-refac/_techspec.md` — canonical implementation and verification contract
- `.compozy/tasks/tools-refac/_tasks.md` — approved task dependency graph
- `.compozy/tasks/tools-refac/adrs/adr-001-agent-tool-surface.md` — tool-first and default-discovery decisions
- `.compozy/tasks/tools-refac/adrs/adr-005-session-bound-autonomy-surface.md` — autonomy hard-cut invariants
- `.compozy/tasks/tools-refac/analysis/competitor-tool-surface-notes.md` — external references that motivated the design

### Dependent Files
- `.compozy/tasks/tools-refac/qa/test-plans/` — planned scenario outputs
- `.compozy/tasks/tools-refac/qa/test-cases/` — enumerated test cases and regression matrix

### Related ADRs
- [ADR-001: Agent Tool Surface Is Tool-First With Default Discovery](adrs/adr-001-agent-tool-surface.md)
- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md)
- [ADR-003: Identity-Bound Task Execution Uses Dedicated Agent Tools](adrs/adr-003-identity-bound-autonomy-tools.md)
- [ADR-004: MCP Auth Exposes Agent Status Only; Login And Logout Stay On Management Surfaces](adrs/adr-004-mcp-auth-status-tool.md)
- [ADR-005: Autonomy Tool Surfaces Are Session-Bound And Never Expose Raw Claim Tokens](adrs/adr-005-session-bound-autonomy-surface.md)
- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md)

### Web/Docs Impact
- `web/`: QA scope must include `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, `web/src/systems/tasks/mocks/fixtures.ts`, and any affected system tests that depend on those shapes.
- `packages/site`: QA scope must include updated runtime core pages plus generated CLI references for task, config, hooks, automation, extension, memory, network, session, workspace, observe, bridge, and MCP auth surfaces.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: the QA plan must cover built-in tools, hosted MCP projections, hook ownership, extension lifecycle, and automation-triggered flows.
- Agent manageability: the QA plan must compare tool, CLI, HTTP, UDS, and hosted MCP behaviors for the same persisted state.
- Config lifecycle: the QA plan must include default discovery overlays, config mutation boundaries, and docs/examples that describe the new lifecycle.

## Deliverables
- `qa/test-plans/` scenario plans for tasks 01-11
- `qa/test-cases/` regression matrix with negative and concurrency cases
- Explicit verification checklist for codegen, site docs, and web downstream artifacts
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for QA artifact completeness **(REQUIRED)**

## Tests
- Unit tests:
  - [x] every implementation task from 01-11 maps to at least one planned scenario and one regression hotspot (`qa/test-plans/tools-refac-traceability.md` § "Coverage Audit")
  - [x] autonomy, MCP auth, policy, approval, and redaction have explicit negative cases in the plan (`qa/test-cases/TC-AUDIT-001.md` § "Negative-Case Coverage")
- Integration tests:
  - [x] the QA dossier covers tool, CLI, HTTP, UDS, hosted MCP, docs, and downstream web artifact verification (`qa/test-plans/tools-refac-traceability.md` § "Coverage Audit" + `qa/test-plans/tools-refac-codegen-and-docs.md`)
  - [x] the planned execution steps are sufficient for task_13 to run without re-scoping the feature (`qa/test-plans/tools-refac-regression.md` smoke/targeted/full lanes + `qa/verification-report.md` scaffold)
- Test coverage target: >=80%
- All tests must pass

## Verification Evidence

- `qa/test-plans/tools-refac-test-plan.md` — feature QA plan, environment matrix, traceability summary, exit criteria.
- `qa/test-plans/tools-refac-regression.md` — smoke / targeted / full lanes with stop conditions.
- `qa/test-plans/tools-refac-traceability.md` — task → TC mapping, hot spots, surface-coverage audit.
- `qa/test-plans/tools-refac-codegen-and-docs.md` — codegen + CLI docs + site build + web consumer steps.
- `qa/test-plans/tools-refac-redaction-suite.md` — cross-channel raw-`claim_token` and MCP-secret sweep procedure.
- `qa/test-cases/TC-FUNC-{001..008}.md`, `TC-INT-{001..006}.md`, `TC-SEC-{001..006}.md`, `TC-AUT-{001..006}.md`, `TC-REG-{001..005}.md`, `TC-UI-001.md`, `TC-AUDIT-001.md` — execution-ready manual cases referenced by the regression suite.
- `qa/verification-report.md` scaffold reserved for task_13.

## Success Criteria
- All tests passing
- Test coverage >=80%
- The QA dossier completely covers tasks 01-11 and their shared regression risks
- Task_13 can execute the feature verification from the saved QA artifacts without re-deriving scope
