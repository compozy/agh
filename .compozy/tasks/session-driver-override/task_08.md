---
status: pending
title: Session Provider Override QA Execution and End-to-End Validation
type: test
complexity: critical
dependencies:
  - task_07
---

# Task 08: Session Provider Override QA Execution and End-to-End Validation

## Overview

Execute the full QA pass for session provider override using the artifacts from task_07, then commit durable regression coverage in the repository's real verification lanes. This task is the quality gate for the entire feature: it must validate create/resume behavior through backend, storage, API, CLI, and browser flows, fix root-cause regressions, and leave fresh evidence under the shared QA artifact layout.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the QA artifacts from task_07 before running any validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/session-driver-override` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- FOLLOW THE PROJECT QA CONTRACT - use repository-defined gates and real runtime/message flows as final proof, not one-off scripts
- DO NOT WEAKEN TESTS TO GET GREEN - fix production code or configuration at the source, then rerun the narrow reproduction and full gates
- E2E COVERAGE IS REQUIRED - the final pass must include browser-visible create/resume flows in addition to backend and transport verification
- GREENFIELD: a validacao final precisa provar o comportamento real de provider override, persistence, migration, e dialog UX, nao apenas snapshots de structs em testes isolados
</critical>

<requirements>
- MUST use the `/qa-execution` skill with `qa-output-path=.compozy/tasks/session-driver-override`
- MUST consume `.compozy/tasks/session-driver-override/qa/test-plans/` and `.compozy/tasks/session-driver-override/qa/test-cases/` from task_07 as the execution matrix seed
- MUST execute the repository verification contract plus real provider-override flows that exercise explicit create, persisted resume, invalid-provider failures, migration/repair, explicit API/CLI surfaces, workspace provider options, and web dialog/resume UX
- MUST capture fresh QA evidence in `.compozy/tasks/session-driver-override/qa/verification-report.md` and store issue files or screenshots under the same artifact root when applicable
- MUST fix root-cause regressions and add or update the narrowest durable regression coverage for every discovered bug
- MUST rerun the repository verification gates after the last fix, including `make verify`, `make codegen-check`, and the targeted web/API test lanes changed by this feature
- MUST include end-to-end coverage for browser-visible create and resume behavior, not only unit or integration tests
- SHOULD validate CLI, HTTP, and UDS parity when the same session/provider scenario is exposed on multiple explicit surfaces
</requirements>

## Subtasks
- [ ] 8.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/session-driver-override` and derive the execution matrix from task_07 artifacts
- [ ] 8.2 Run the baseline repository verification gate and establish the pre-execution health state
- [ ] 8.3 Execute backend, storage, transport, CLI, and browser-visible provider-override scenarios through supported repo surfaces
- [ ] 8.4 Fix root-cause regressions, add matching regression coverage, and rerun impacted scenarios
- [ ] 8.5 Rerun final verification gates and publish `.compozy/tasks/session-driver-override/qa/verification-report.md`

## Implementation Details

See the TechSpec "Testing Approach" plus the QA artifacts from task_07. The key constraint is end-to-end proof: the feature must work through the real seams that operators use, from create-time provider selection to persisted resume and error handling when that provider disappears later.

### Relevant Files
- `.agents/skills/qa-execution/SKILL.md` - required workflow for execution matrix discovery, evidence capture, and verification reporting
- `.agents/skills/qa-execution/scripts/discover-project-contract.py` - canonical project-contract discovery entrypoint required by `/qa-execution`
- `Makefile` - repository-defined verification gate that must be rerun after the last fix
- `internal/config/provider_test.go` - likely regression destination for provider resolution bugs found during QA
- `internal/session/manager_test.go` - likely regression destination for lifecycle ordering and persistence bugs
- `internal/session/manager_integration_test.go` - likely regression destination for create/resume runtime scenarios
- `internal/store/globaldb/global_db_session_test.go` - likely regression destination for index migration and repair bugs
- `internal/api/core/session_workspace_internal_test.go` - likely regression destination for HTTP/UDS contract and workspace detail issues
- `internal/cli/session_test.go` - likely regression destination for CLI provider surface issues
- `internal/extension/host_api_integration_test.go` - likely regression destination for host API create/read issues
- `web/src/routes/_app/-index.test.tsx` - likely regression destination for dialog create flow coverage
- `web/src/routes/_app/-session.$id.test.tsx` - likely regression destination for resume failure UX coverage

### Dependent Files
- `.compozy/tasks/session-driver-override/qa/verification-report.md` - final QA evidence produced by `/qa-execution`
- `.compozy/tasks/session-driver-override/qa/issues/BUG-*.md` - structured bug reports for failures discovered during execution
- `.compozy/tasks/session-driver-override/qa/screenshots/` - browser-visible evidence for dialog and resume failure scenarios
- `openapi/agh.json` - may need regeneration if execution finds contract drift
- `web/src/generated/agh-openapi.d.ts` - may need regeneration if execution finds codegen drift

### Related ADRs
- [ADR-002: Re-Resolve Provider-Owned Runtime Fields On Session Override](adrs/adr-002.md) - QA execution must prove provider-owned runtime fields are re-resolved coherently
- [ADR-003: Persist Effective Session Provider And Fail Explicitly On Mismatch](adrs/adr-003.md) - QA execution must prove persisted-provider and explicit-failure semantics end to end
- [ADR-004: Use Explicit Session Creation Surfaces For Provider Selection](adrs/adr-004.md) - QA execution must prove explicit create surfaces and dialog UX
- [ADR-005: Migrate Session Provider State In Place And Repair Legacy Metadata Once](adrs/adr-005.md) - QA execution must prove migration and one-time repair behavior

## Deliverables
- Fresh `.compozy/tasks/session-driver-override/qa/verification-report.md` produced by `/qa-execution`
- QA evidence covering explicit provider create, persisted-provider resume, invalid-provider create failures, unavailable-provider resume failures, migration/repair, API/CLI parity, workspace provider catalog, and web dialog/resume UX **(REQUIRED)**
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**
- Fresh issue files and supplementary evidence under `.compozy/tasks/session-driver-override/qa/` **(REQUIRED)**
- Passing repository verification gates after the final QA fix set **(REQUIRED)**
- Fresh browser-visible E2E evidence proving the dialog selection flow and resume failure UX **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Any bug found in provider resolution gains a narrow regression that proves the exact override invariant that failed
  - [ ] Any bug found in session persistence or resume ordering gains a regression that proves the specific lifecycle invariant that broke
  - [ ] Any bug found in migration or repair gains a regression that proves the exact schema or repair behavior
  - [ ] Any bug found in CLI, API, or generated contracts gains the narrowest regression that proves parity on `provider`
- Integration tests:
  - [ ] Real create-session flows prove an explicit provider override reaches runtime, persistence, and returned payloads coherently
  - [ ] Real resume flows prove the persisted provider wins even after an agent default changes
  - [ ] Real failure flows prove create and resume fail explicitly when the requested or persisted provider is unavailable
  - [ ] Real global DB and legacy metadata fixtures prove migration and one-time repair end to end
  - [ ] Real CLI, HTTP, UDS, and Host API flows stay aligned on the effective provider field
  - [ ] Real browser-visible flows prove the session creation dialog, provider picker, successful create path, and resume failure UX end to end
  - [ ] `make verify` and `make codegen-check` pass from a clean rerun after the final QA fix set
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-execution` has been run explicitly with artifacts stored under `.compozy/tasks/session-driver-override/qa/`
- The feature has fresh backend-to-browser evidence proving provider override works end to end
