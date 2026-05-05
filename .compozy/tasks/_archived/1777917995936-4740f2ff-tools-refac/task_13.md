---
status: completed
title: Real-Scenario QA Execution
type: test
complexity: critical
dependencies:
  - task_12
---

# Task 13: Real-Scenario QA Execution

## Overview

Execute the `tools-refac` QA dossier against a real daemon-served runtime and prove the final surface works end to end across the managed AGH paths. This task should validate the CLI/HTTP/UDS/tool/hosted-MCP contract, the autonomy hard cut, the config and mutation families, and the generated web/docs fallout with fresh evidence.

<critical>
- ALWAYS READ `qa/test-plans/*` and `qa/test-cases/*` before executing
- REFERENCE the saved QA dossier; do not shrink scope during execution
- FOCUS ON WHAT: run real-scenario verification, capture evidence, and fix root-cause defects if found
- MINIMIZE mock-driven confidence — use a fresh isolated QA lab and real daemon-served flows
- TESTS REQUIRED — run runtime end-to-end checks plus the contract/codegen/site/web validations required by the plan
</critical>

<requirements>
1. MUST execute the saved QA plan from task_12 against an isolated AGH runtime with unique `AGH_HOME`, ports, and provider homes.
2. MUST run `make test-e2e-runtime` and exercise the affected CLI verbs, HTTP/UDS routes, hosted MCP surface, and tool-call flows end to end.
3. MUST verify the autonomy hard cut, redaction rules, policy overlays, approvals, config mutation, hooks, automation, extensions, and MCP auth status behavior against persisted runtime state.
4. MUST verify downstream generated/web/docs artifacts with the checks required by the plan, including codegen and site build validation.
5. MUST record defects under `qa/issues/BUG-NNN.md`, fix root causes, and update `qa/verification-report.md` with final evidence.
</requirements>

## Subtasks
- [x] 13.1 Bootstrap a fresh isolated QA lab and capture the manifest and runtime coordinates
- [x] 13.2 Execute CLI, HTTP, UDS, tool, and hosted MCP scenarios from the QA dossier
- [x] 13.3 Execute autonomy hard-cut, redaction, policy, approval, and mutation boundary scenarios
- [x] 13.4 Run codegen, site, and downstream web verification required by the dossier
- [x] 13.5 Capture defects, fix root causes, re-run affected checks, and publish the verification report

## Implementation Details

Use the `agh-qa-bootstrap`, `real-scenario-qa`, and `qa-execution` workflows. Because this feature is agent-manageability and contract heavy rather than a new web UI, the execution emphasis is `make test-e2e-runtime` plus structured CLI/API/hosted-MCP parity, with web validation focused on generated types, task mocks, and affected tests rather than Playwright-first UI coverage.

### Relevant Files
- `.compozy/tasks/tools-refac/qa/test-plans/` — saved scenario plan to execute
- `.compozy/tasks/tools-refac/qa/test-cases/` — regression and negative test matrix
- `.compozy/tasks/tools-refac/qa/issues/` — defect records discovered during execution
- `.compozy/tasks/tools-refac/qa/verification-report.md` — final execution evidence and outcome
- `web/src/systems/tasks/types.ts` — downstream consumer that must stay aligned with the autonomy hard cut

### Dependent Files
- `web/src/generated/agh-openapi.d.ts` — regenerated contract output that must verify cleanly
- `packages/site/content/runtime/` — updated docs and CLI references that must build without drift

### Related ADRs
- [ADR-003: Identity-Bound Task Execution Uses Dedicated Agent Tools](adrs/adr-003-identity-bound-autonomy-tools.md)
- [ADR-004: MCP Auth Exposes Agent Status Only; Login And Logout Stay On Management Surfaces](adrs/adr-004-mcp-auth-status-tool.md)
- [ADR-005: Autonomy Tool Surfaces Are Session-Bound And Never Expose Raw Claim Tokens](adrs/adr-005-session-bound-autonomy-surface.md)
- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md)

### Web/Docs Impact
- `web/`: execute downstream verification for `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, `web/src/systems/tasks/mocks/fixtures.ts`, and any affected unit/integration tests identified in task_12.
- `packages/site`: build and spot-check the updated runtime core and CLI reference pages that describe the canonical surface.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: verification must cover built-in tools, hosted MCP, hooks, automation, extensions, and any projection changes visible to extension or sidecar workflows.
- Agent manageability: verification must compare tool, CLI, HTTP, UDS, and hosted MCP paths for the same underlying state and deny conditions.
- Config lifecycle: verification must prove default discovery, config mutation boundaries, and docs/examples line up with runtime truth.

## Deliverables
- Fresh QA bootstrap manifest and execution evidence
- `qa/issues/BUG-NNN.md` files for every reproduced defect
- Updated `qa/verification-report.md` with final pass/fail evidence
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests and runtime E2E evidence **(REQUIRED)**

## Tests
- Unit tests:
  - [x] execute the planned downstream web and generated-type regression checks for the autonomy hard cut
  - [x] execute the planned negative checks for policy, approval, auth status, and redaction behavior
- Integration tests:
  - [x] run `make test-e2e-runtime` and the saved CLI/HTTP/UDS/tool/hosted-MCP parity scenarios against the isolated QA lab
  - [x] run the codegen, docs, and downstream verification commands required by the QA dossier and record their evidence
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Real-scenario QA proves the canonical surface works across runtime, docs, and downstream generated artifacts
- Any discovered defect is captured, fixed at the root cause, and re-verified with fresh evidence
