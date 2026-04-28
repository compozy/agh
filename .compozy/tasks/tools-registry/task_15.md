---
status: pending
title: QA Plan and Test Coverage
type: test
complexity: high
dependencies:
  - task_14
---

# Task 15: QA Plan and Test Coverage

## Overview

Generate the release-grade QA planning artifacts for the Tool Registry after implementation and docs land. This task uses `qa-report` to create traceable plans, manual cases, regression suites, and artifact paths covering native tools, TypeScript and Go extension-host tools, MCP call-through, hosted MCP, CLI/HTTP/UDS, web diagnostics, docs, config, policy, approval, and redaction.

<critical>
- ALWAYS READ `_techspec.md`, every ADR, tasks 01-14, and every per-task memory file before drafting test cases
- ACTIVATE `qa-report` with `qa-output-path=.compozy/tasks/tools-registry`
- KEEP the same `qa-output-path` for `qa-execution`; all planning and execution artifacts must live under `.compozy/tasks/tools-registry/qa/`
- DO NOT execute the flows in this task; this is planning, prioritization, traceability, and artifact generation only
- EVERY public surface touched by tasks 01-14 must have explicit coverage
</critical>

<requirements>
1. MUST use the `qa-report` skill with `qa-output-path=.compozy/tasks/tools-registry`.
2. MUST create a feature-level QA plan under `.compozy/tasks/tools-registry/qa/test-plans/`.
3. MUST create manual test cases under `.compozy/tasks/tools-registry/qa/test-cases/`.
4. MUST create regression-suite documents covering smoke, targeted, full, and security/redaction priorities.
5. MUST trace every P0/P1 case to tasks 01-14, `_techspec.md`, or ADR-001 through ADR-010.
6. MUST reserve stable artifact directories for issues, logs, screenshots, traces, fixtures, and verification reports.
</requirements>

## Subtasks
- [ ] 15.1 Activate `qa-report` with `qa-output-path=.compozy/tasks/tools-registry`
- [ ] 15.2 Write feature-level QA plan with environment matrix, risks, entry/exit criteria, and artifact layout
- [ ] 15.3 Generate manual test cases for native, extension-host, MCP, hosted MCP, policy, approval, CLI/API/UDS, web, docs, and config
- [ ] 15.4 Build regression suites with P0/P1 ordering for task_16 execution
- [ ] 15.5 Map each P0/P1 case to tasks, TechSpec invariants, ADR decisions, or safety invariants
- [ ] 15.6 Include negative tests, edge cases, concurrency stress, and redaction leak assertions

## Implementation Details

Use the TechSpec "Test Strategy", "Safety Invariants", and all task files 01-14. The QA plan must prove real executable tools through all backend classes and must not rely on mocks alone for final confidence.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` - required QA planning workflow
- `.compozy/tasks/tools-registry/_techspec.md` - authoritative requirements and invariants
- `.compozy/tasks/tools-registry/adrs/` - accepted architecture decisions
- `.compozy/tasks/tools-registry/task_01.md` through `.compozy/tasks/tools-registry/task_14.md` - implementation and verification scope
- `web/CLAUDE.md` - web verification rules for UI-bearing coverage
- `packages/site/CLAUDE.md` - site verification rules for docs coverage

### Dependent Files
- `.compozy/tasks/tools-registry/qa/test-plans/tool-registry-test-plan.md` - feature-level plan
- `.compozy/tasks/tools-registry/qa/test-plans/*-regression.md` - regression suites consumed by task_16
- `.compozy/tasks/tools-registry/qa/test-cases/TC-*.md` - manual test cases
- `.compozy/tasks/tools-registry/qa/issues/BUG-*.md` - defects found during QA planning or execution
- `.compozy/tasks/tools-registry/qa/verification-report.md` - task_16 final report

### Related ADRs
- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - QA must prove executable native and extension-host tools
- [ADR-002: Session Tool Exposure Path](adrs/adr-002-session-tool-exposure-path.md) - QA must prove hosted MCP exposure
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - QA must prove policy and approval behavior
- [ADR-008: Manifest-Authoritative Extension Tool Descriptors](adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md) - QA must prove reconciliation
- [ADR-009: Public Go Extension Tool SDK](adrs/adr-009-public-go-extension-tool-sdk.md) - QA must prove Go SDK authoring
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - QA must prove MCP call-through and auth redaction

### Web/Docs Impact
- `web/`: QA plan must include web diagnostics, generated type parity, MSW fixtures, route/component behavior, and Playwright/browser coverage from task_13.
- `packages/site`: QA plan must include site docs, generated CLI reference, config docs, extension docs, MCP docs, and API reference coverage from task_14.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: QA scope includes extension manifests, TypeScript SDK, Go SDK, MCP backend tools, hosted MCP, hooks, tool resources, and registry backends.
- Agent manageability: QA scope includes CLI structured output, HTTP endpoints, UDS routes, session projections, deterministic errors, and approval paths.
- Config lifecycle: QA scope includes `[tools]`, `[tools.policy]`, `[tools.hosted_mcp]`, agent `tools`, `toolsets`, `deny_tools`, defaults, overlays, examples, and docs.

## Deliverables
- `.compozy/tasks/tools-registry/qa/test-plans/tool-registry-test-plan.md`
- One or more `.compozy/tasks/tools-registry/qa/test-plans/*-regression.md` documents reusable by task_16
- Manual test cases under `.compozy/tasks/tools-registry/qa/test-cases/` **(REQUIRED)**
- Traceability matrix from P0/P1 cases to tasks, TechSpec sections, ADRs, and safety invariants **(REQUIRED)**
- Stable artifact layout for task_16 logs, screenshots, traces, issues, and verification reporting **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] QA plan includes objectives, scope, risks, environment matrix, entry criteria, and exit criteria across backend, CLI, API, UDS, web, docs, SDKs, MCP, and config
  - [ ] Manual test cases exist for native tools, TypeScript extension tools, Go extension tools, MCP call-through, hosted MCP, policy, approval, redaction, and web/docs
  - [ ] Regression suites define smoke, targeted, full, and security/redaction lanes with explicit P0/P1 ordering
  - [ ] Every P0/P1 case names the exact task, TechSpec invariant, or ADR it proves
- Integration tests:
  - [ ] All generated QA artifacts live under `.compozy/tasks/tools-registry/qa/`
  - [ ] task_16 can consume the plan without redefining scope, priorities, output paths, or environment setup
  - [ ] Any bug report created during planning references the originating test case or documented discrepancy
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `qa-report` artifacts are complete, traceable, and stored under `.compozy/tasks/tools-registry/qa/`
- task_16 can begin execution without changing QA scope or artifact paths
