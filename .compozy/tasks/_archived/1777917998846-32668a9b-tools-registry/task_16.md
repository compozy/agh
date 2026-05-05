---
status: completed
title: Real-Scenario QA Execution
type: test
complexity: critical
dependencies:
    - task_15
---

# Task 16: Real-Scenario QA Execution

## Overview

Execute the Tool Registry QA plan against a fresh, isolated, production-like AGH lab. This task uses `qa-execution`, `real-scenario-qa`, and browser automation for the UI-bearing scope, fixes root causes for reproduced defects, reruns gates, and records machine-readable evidence.

<critical>
- ALWAYS READ `.compozy/tasks/tools-registry/qa/test-plans/*` before executing
- ACTIVATE `real-scenario-qa`, `qa-execution`, `agh-qa-bootstrap`, and `agh-worktree-isolation`
- USE unique `AGH_HOME`, daemon ports, provider homes, and tmux bridge/socket paths for this QA pass
- For UI coverage, drive Playwright via `browser-use:browser`; fall back to `agent-browser` only if `browser-use:browser` is unavailable
- DO NOT rely on mocks only; final validation must exercise real daemon, CLI, HTTP/UDS, SDK, MCP, browser, and docs/build flows
</critical>

<requirements>
1. MUST bootstrap a fresh isolated QA lab and persist the bootstrap manifest path in the verification report.
2. MUST execute task_15 P0/P1 cases for native tools, TypeScript extension tools, Go extension tools, external MCP call-through, hosted MCP, policy, approval, redaction, CLI/HTTP/UDS, web, docs, and config.
3. MUST run `make test-e2e-runtime` and `make test-e2e-web` for the UI-bearing feature scope.
4. MUST drive the highest-risk web diagnostics flow through `browser-use:browser` with `agent-browser` fallback only if needed.
5. MUST create `.compozy/tasks/tools-registry/qa/issues/BUG-NNN.md` for every reproduced defect and fix root causes before claiming completion.
6. MUST run final `make verify` and record evidence, lab root, runtime home, base URL, provider homes, and command outputs in `qa/verification-report.md`.
</requirements>

## Subtasks
- [ ] 16.1 Bootstrap a fresh isolated QA lab with `agh-qa-bootstrap` and record manifest/runtime paths
- [ ] 16.2 Execute smoke and P0/P1 backend/CLI/API/UDS/config cases from task_15
- [ ] 16.3 Execute real TypeScript and Go extension-host tool fixtures through the registry
- [ ] 16.4 Execute real external MCP/OAuth call-through and hosted MCP session exposure flows
- [ ] 16.5 Execute web diagnostics through Playwright and `browser-use:browser`, then validate site docs/build
- [ ] 16.6 File and fix every reproduced defect, rerun targeted checks, then run final `make verify`
- [ ] 16.7 Write `qa/verification-report.md` with machine-readable QA bootstrap evidence

## Implementation Details

Use task_15 artifacts as the execution contract. This task validates real behavior across the daemon, extension subprocesses, MCP servers, CLI, HTTP, UDS, web, generated docs, config lifecycle, and redaction boundaries.

Run `make test-e2e-runtime` (daemon harness) and `make test-e2e-web` (Playwright). Drive the highest-risk UI workflow through `browser-use:browser`; fall back to `agent-browser` only if `browser-use:browser` is unavailable. Do not silently substitute shell-only checks.

For CLI/API/agent-manageability coverage, exercise structured CLI output, HTTP/UDS routes, status/config discovery, deterministic errors, and compare persisted daemon state. For extensibility/config coverage, validate TypeScript and Go extension tool authoring, MCP auth/config lifecycle, hosted MCP, and config overlays end-to-end.

### Relevant Files
- `.compozy/tasks/tools-registry/qa/test-plans/*` - execution plan and regression suites
- `.compozy/tasks/tools-registry/qa/test-cases/TC-*.md` - manual and scenario test cases
- `.agents/skills/agh-qa-bootstrap/SKILL.md` - deterministic QA lab bootstrap workflow
- `.agents/skills/real-scenario-qa/SKILL.md` - release-grade scenario QA workflow
- `.agents/skills/qa-execution/SKILL.md` - execution and reporting workflow
- `web/e2e/**` - browser-side E2E coverage if tests are added or extended
- `internal/testutil/acpmock/**` - ACP hosted MCP and approval bridge fixtures

### Dependent Files
- `.compozy/tasks/tools-registry/qa/verification-report.md` - final evidence report
- `.compozy/tasks/tools-registry/qa/issues/BUG-*.md` - reproduced defects and fixes
- `.compozy/tasks/tools-registry/qa/screenshots/**` - browser evidence
- `.compozy/tasks/tools-registry/qa/logs/**` - daemon, CLI, MCP, extension, and web logs
- `.compozy/tasks/tools-registry/qa/bootstrap-manifest.json` - copied or referenced QA bootstrap manifest

### Related ADRs
- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - execution must prove executable native and extension-host tools
- [ADR-002: Session Tool Exposure Path](adrs/adr-002-session-tool-exposure-path.md) - execution must prove hosted MCP exposure
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - execution must prove approval behavior
- [ADR-008: Manifest-Authoritative Extension Tool Descriptors](adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md) - execution must prove reconciliation
- [ADR-009: Public Go Extension Tool SDK](adrs/adr-009-public-go-extension-tool-sdk.md) - execution must prove Go SDK authoring
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - execution must prove remote MCP call-through and auth redaction

### Web/Docs Impact
- `web/`: execute `make test-e2e-web`, browser-use diagnostics flow, generated type checks, MSW-backed tests, and web build.
- `packages/site`: execute docs source generation, typecheck/build, generated CLI reference verification, and docs scenario checks from task_15.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: validates real TypeScript SDK, public Go SDK, extension manifests, MCP backend tools, hosted MCP, hooks, and registry dispatch.
- Agent manageability: validates CLI structured output, HTTP/UDS parity, session projections, invoke flows, deterministic errors, and approval paths.
- Config lifecycle: validates fresh config defaults, overlays, invalid values, MCP auth config, hosted MCP keys, policy keys, docs examples, and no legacy aliases.

## Deliverables
- Fresh QA bootstrap manifest and isolated lab evidence
- Executed QA cases with logs, screenshots, traces, and command evidence
- `BUG-NNN.md` reports for every reproduced defect **(REQUIRED when defects are found)**
- Root-cause fixes and rerun evidence for every resolved defect **(REQUIRED when defects are found)**
- Final `.compozy/tasks/tools-registry/qa/verification-report.md` **(REQUIRED)**
- Final `make verify` evidence **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Targeted package tests rerun for every defect fix
  - [ ] Redaction leak checks assert sentinel tokens are absent from logs, events, CLI JSON, HTTP JSON, UDS JSON, MCP responses, web payloads, and QA artifacts
  - [ ] Config validation tests cover invalid aliases, unsafe values, and overlay precedence
- Integration tests:
  - [ ] `make test-e2e-runtime` passes against an isolated lab
  - [ ] `make test-e2e-web` passes and the highest-risk UI workflow is driven through `browser-use:browser`
  - [ ] Real TypeScript and Go extension fixtures publish and execute tools through the registry
  - [ ] Real external MCP/OAuth fixture proves call-through and redacted auth diagnostics
  - [ ] Hosted MCP `tools/list` and `tools/call` match session projections and approval behavior
  - [ ] CLI, HTTP, and UDS outputs agree for the same persisted state
  - [ ] `make verify` passes after all fixes
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- QA verification report includes manifest path, lab root, runtime home, provider homes, base URL, command evidence, and final gate results
- No known P0/P1 QA defects remain open
