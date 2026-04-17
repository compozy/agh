---
status: pending
title: Settings QA plan and regression artifacts
type: docs
complexity: high
dependencies:
  - task_10
  - task_11
  - task_12
  - task_13
  - task_14
---

# Task 15: Settings QA plan and regression artifacts

## Overview

Generate the reusable QA planning artifacts for the full settings feature before live execution begins. This task must leave the feature with a concrete test plan, route-by-route manual test cases, and regression-suite definitions that the follow-up execution task can consume without re-deciding scope or output paths.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_10.md` through `task_14.md` before planning coverage (`_prd.md` is absent; requirements come from the TechSpec and the generated task set)
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/settings-ui` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` — all planning and execution artifacts must live under `.compozy/tasks/settings-ui/qa/`
- FOCUS ON "WHAT" — define the coverage model, risks, and operator flows; do not execute the product or invent implementation fixes in this task
- USE THE SETTINGS DESIGN REFERENCES — derive UI and visual cases from `docs/design/paper/settings/` and use Figma only if it is actually configured
- GREENFIELD: cobrir todas as superfícies de settings de forma explícita; não aceitar smoke genérico que deixe páginas ou fluxos críticos sem rastreabilidade
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/settings-ui`
- MUST generate a feature-level test plan under `.compozy/tasks/settings-ui/qa/test-plans/`
- MUST generate manual test cases that cover `general`, `memory`, `observability`, `skills`, `automation`, `network`, `providers`, `environments`, `mcp-servers`, and `hooks-extensions`
- MUST cover restart-required versus applied-now messaging, restart-operation polling, collection CRUD, workspace-scoped MCP behavior, and extension immediate-action behavior
- MUST produce at least one regression-suite document that defines smoke, targeted, and full execution priorities for the follow-up `/qa-execution` task
- SHOULD create UI-focused test cases from the local Paper exports when Figma MCP is unavailable
</requirements>

## Design References

All 10 Settings Paper artboards are in scope for this task. Use the local exports under `docs/design/paper/settings/` and the `_techspec.md` "Design References" section to ensure route-by-route visual and behavioral coverage stays explicit.

## Subtasks

- [ ] 15.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/settings-ui`
- [ ] 15.2 Write the feature-level settings test plan with scope, risks, environments, and entry/exit criteria
- [ ] 15.3 Generate route-by-route manual test cases with explicit expected results and edge cases
- [ ] 15.4 Build the regression suite definitions and identify the P0/P1 flows that `/qa-execution` must run first
- [ ] 15.5 Validate artifact completeness, traceability, and handoff readiness for `task_16`

## Implementation Details

See TechSpec sections "Testing Approach", "Web route coverage", "Known Risks", and "Development Sequencing". This task is the formal handoff from implementation tasks to QA execution: it should capture what must be proven, where evidence will live, and which flows are too risky to leave to ad hoc exploratory testing.

### Relevant Files

- `.agents/skills/qa-report/SKILL.md` — required workflow, output structure, and artifact naming rules for the planning pass
- `.compozy/tasks/settings-ui/_techspec.md` — source of truth for settings routes, restart semantics, and verification expectations
- `.compozy/tasks/settings-ui/task_10.md` — defines the `general`, `memory`, and `observability` surfaces that QA must trace explicitly
- `.compozy/tasks/settings-ui/task_11.md` — defines the `skills`, `automation`, and `network` settings surfaces and linked operational behavior
- `.compozy/tasks/settings-ui/task_12.md` — defines `providers` and `environments` collection CRUD expectations
- `.compozy/tasks/settings-ui/task_13.md` — defines workspace-scoped MCP semantics, precedence, and target selection behavior
- `.compozy/tasks/settings-ui/task_14.md` — defines the hybrid hooks/extensions page and its immediate-action versus restart-aware split
- `docs/design/paper/settings/` — local Paper exports used to derive UI/visual test coverage when Figma is not available

### Dependent Files

- `.compozy/tasks/settings-ui/qa/test-plans/settings-ui-test-plan.md` — feature-level QA plan created by this task
- `.compozy/tasks/settings-ui/qa/test-plans/*-regression.md` — regression suite document(s) consumed by the execution task
- `.compozy/tasks/settings-ui/qa/test-cases/TC-*.md` — manual test cases with priorities and expected results
- `.compozy/tasks/settings-ui/qa/issues/BUG-*.md` — only created if planning uncovers a concrete documented discrepancy
- `.compozy/tasks/settings-ui/task_16.md` — execution task that must consume this artifact set unchanged

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — QA planning must treat settings as one nested product surface, not unrelated screens
- [ADR-002: Persist settings by writing canonical config overlays instead of creating a new settings store](adrs/adr-002.md) — MCP target and precedence behavior require explicit manual coverage
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Test planning must distinguish applied-now, restart-required, and operational-action flows
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — HTTP mutation restrictions and operator messaging must be represented in QA coverage

## Deliverables

- `.compozy/tasks/settings-ui/qa/test-plans/settings-ui-test-plan.md`
- One or more `.compozy/tasks/settings-ui/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`
- Route-by-route manual test cases under `.compozy/tasks/settings-ui/qa/test-cases/` **(REQUIRED)**
- P0/P1 coverage for restart flows, collection CRUD, MCP scope/precedence, and hooks/extensions hybrid behavior **(REQUIRED)**
- A stable artifact layout under `.compozy/tasks/settings-ui/qa/` that the execution task can consume without path changes **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] `settings-ui-test-plan.md` includes objectives, scope, environment matrix, entry/exit criteria, and risk assessment
  - [ ] Manual test cases exist for every settings route and include setup, explicit steps, expected results, and cleanup notes where needed
  - [ ] Regression suite documents identify smoke, targeted, and full coverage lanes plus the execution order for P0/P1 flows
  - [ ] Route-to-task traceability maps cases back to the relevant settings routes or task files clearly
  - [ ] All generated artifacts land under `.compozy/tasks/settings-ui/qa/` with stable names and no missing referenced files
- Integration tests:
  - [ ] Restart, restart-status, workspace-scope, collection CRUD, and operator-action flows are represented coherently across plan, cases, and regression docs
  - [ ] Manual cases and regression suites align on priorities, environment prerequisites, and expected test data setup
  - [ ] Generated artifacts can be consumed directly by `/qa-execution` without manual reformatting or missing dependencies
  - [ ] Any bug report or open-risk note created during planning references the originating test case or design discrepancy clearly

## Success Criteria

- The `/qa-report` workflow has been executed explicitly and its artifacts are stored under `.compozy/tasks/settings-ui/qa/`
- Every settings screen and critical operator flow has at least one traceable QA artifact
- `task_16` can start execution without redefining scope, output paths, or risk priorities
- The settings feature has a concrete regression plan instead of ad hoc manual testing notes
