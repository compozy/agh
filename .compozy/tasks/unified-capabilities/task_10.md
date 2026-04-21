---
status: completed
title: Unified Capabilities QA Execution and End-to-End Validation
type: test
complexity: critical
dependencies:
  - task_09
---

# Task 10: Unified Capabilities QA Execution and End-to-End Validation

## Overview

Execute the full QA pass for unified capabilities using the artifacts from task_09, then commit durable regression coverage in the repository's verification lanes. This task is the quality gate for the entire unification effort: it must validate real backend, API, frontend, and documentation flows, fix root-cause regressions, and leave fresh evidence under the shared QA artifact layout.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the QA artifacts from task_09 before running any validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/unified-capabilities` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- FOLLOW THE PROJECT QA CONTRACT - use repository-defined gates and real runtime/API/UI/doc flows as final proof, not one-off scripts
- DO NOT WEAKEN TESTS TO GET GREEN - fix production code or configuration at the source, then rerun the narrow reproduction and full gates
- GREENFIELD: the final QA pass must prove the repo no longer behaves like a split capability/recipe system on any surfaced path
</critical>

<requirements>
- MUST use the `/qa-execution` skill with `qa-output-path=.compozy/tasks/unified-capabilities`
- MUST consume `.compozy/tasks/unified-capabilities/qa/test-plans/` and `.compozy/tasks/unified-capabilities/qa/test-cases/` from task_09 as the execution matrix seed
- MUST execute the repository verification contract plus real backend/runtime, API, frontend, and site-doc scenarios that cover the unification end to end
- MUST capture fresh QA evidence in `.compozy/tasks/unified-capabilities/qa/verification-report.md` and store issue files or screenshots under the same artifact root when applicable
- MUST fix root-cause regressions and add or update the narrowest durable regression coverage for every discovered bug
- MUST rerun the repository verification gates after the last fix, including the relevant web and integration lanes touched by this feature
- SHOULD use browser-based evidence for the `web/` network surface when a dev server can be started locally
</requirements>

## Subtasks
- [x] 10.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/unified-capabilities` and derive the execution matrix from task_09 artifacts
- [x] 10.2 Run the baseline repository verification gate and establish the pre-execution health state
- [x] 10.3 Execute real backend/API/frontend/docs scenarios for unified capabilities through supported repo surfaces
- [x] 10.4 Fix root-cause regressions, add matching regression coverage, and rerun impacted scenarios
- [x] 10.5 Rerun final verification gates and publish `.compozy/tasks/unified-capabilities/qa/verification-report.md`

## Implementation Details

See the TechSpec "Testing Approach" plus the QA artifacts from task_09. The key constraint is that unified capability QA must prove behavior through real seams: capability loading and digesting, transfer under `kind:"capability"`, discovery and API payloads, the network web UI, and the updated protocol/runtime docs.

### Relevant Files
- `.agents/skills/qa-execution/SKILL.md` - required workflow for execution matrix discovery, evidence capture, and verification reporting
- `.agents/skills/qa-execution/scripts/discover-project-contract.py` - canonical project-contract discovery entrypoint required by `/qa-execution`
- `Makefile` - repository-defined verification gate that must be rerun after the last fix
- `web/AGENTS.md` - frontend verification constraints for `make web-lint`, `make web-typecheck`, and route/system expectations
- `.compozy/tasks/unified-capabilities/qa/test-plans/` - task_09 artifacts that seed execution priorities and evidence expectations
- `.compozy/tasks/unified-capabilities/qa/test-cases/` - manual cases that define exact backend, frontend, and docs flows to run

### Dependent Files
- `.compozy/tasks/unified-capabilities/qa/verification-report.md` - final QA evidence produced by `/qa-execution`
- `.compozy/tasks/unified-capabilities/qa/issues/BUG-*.md` - structured bug reports for failures discovered during execution
- `.compozy/tasks/unified-capabilities/qa/screenshots/` - browser or visual evidence captured during execution
- `internal/network/router_integration_test.go` - likely regression destination for transfer and lifecycle bugs found during QA
- `internal/api/core/network_test.go` - likely regression destination for discovery/API contract bugs found during QA
- `web/src/routes/_app/-network.test.tsx` - likely regression destination for web-surface bugs found during QA
- `packages/site/content/protocol/*.mdx` - likely correction surface if protocol-doc execution reveals inconsistencies

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - QA execution must prove the repo now behaves around one concept
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) - QA execution must prove authored/runtime schema behavior remains correct
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - QA execution must prove transfer and lifecycle semantics under the new kind

## Deliverables
- Fresh `.compozy/tasks/unified-capabilities/qa/verification-report.md` produced by `/qa-execution`
- QA evidence covering backend schema/digesting, capability transfer, discovery/API alignment, web network UX, and `packages/site` consistency **(REQUIRED)**
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**
- Fresh issue files and supplementary evidence under `.compozy/tasks/unified-capabilities/qa/` **(REQUIRED)**
- Passing repository verification gates after the final QA fix set **(REQUIRED)**
- Fresh evidence that integration lanes and real surfaced flows, not only isolated unit tests, prove the unification **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Any bug found in schema/digest handling gains a narrow regression that proves the exact normalization or validation rule that failed
  - [x] Any bug found in transfer or lifecycle handling gains a regression that proves the specific `kind:"capability"` invariant that failed
  - [x] Any bug found in discovery/API contracts gains the narrowest regression in backend contract or handler tests that proves the exact payload issue
  - [x] Any bug found in the web network surface gains the narrowest route, hook, or adapter regression that proves the actual operator-facing failure
- Integration tests:
  - [x] Real runtime flows prove unified capabilities load, digest, and surface through discovery without recipe remnants
  - [x] Real transfer flows prove `kind:"capability"` delivery and lifecycle behavior end to end
  - [x] Real API and UDS flows prove peer details and discovery payloads remain coherent after the unification
  - [x] Real `web/` flows prove the network UI renders unified capabilities correctly when driven against the updated contract
  - [x] Final docs review proves the repository and site no longer teach recipe as a first-class steady-state concept
  - [x] `make verify` and the required web verification gates pass from a clean rerun after the final QA fix set
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-execution` has been run explicitly with artifacts stored under `.compozy/tasks/unified-capabilities/qa/`
- The feature has fresh end-to-end evidence proving AGH now exposes one unified capability model across runtime, protocol, web, and docs
