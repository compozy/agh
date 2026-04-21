---
status: completed
title: Harness QA execution and daemon/runtime E2E
type: test
complexity: critical
dependencies:
  - task_09
---

# Task 10: Harness QA execution and daemon/runtime E2E

## Overview

Execute the full QA pass for the harness architecture using the artifacts from `task_09`, then commit durable regression coverage in the repo's existing daemon/runtime verification lanes. This task is the quality gate for the whole harness slice: it must validate real startup, prompt, detached-runtime, transcript, and observe flows, fix root-cause regressions, and leave behind repeatable evidence under the shared project contract.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the QA artifacts from `task_09` before running any validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/harness` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- FOLLOW THE PROJECT QA CONTRACT - use the repo's existing daemon, transport-parity, integration, and E2E lanes instead of one-off scripts as final proof
- FOCUS ON SHIPPED RUNTIME FLOWS - startup prompting, augmentation, synthetic turns, detached completion, transcript trust, and observability all need durable proof
- DO NOT WEAKEN TESTS TO GET GREEN - fix production code or configuration at the source, then rerun the affected scenarios and full gates
- GREENFIELD: a validacao do harness precisa entrar no fluxo normal do projeto, nao ficar como experimento manual isolado
</critical>

<requirements>
- MUST use the `/qa-execution` skill with `qa-output-path=.compozy/tasks/harness`
- MUST consume `.compozy/tasks/harness/qa/test-plans/` and `.compozy/tasks/harness/qa/test-cases/` from `task_09` as the execution matrix seed
- MUST execute real daemon/runtime, transport-parity, transcript, and detached task-runtime scenarios against the current repository state
- MUST capture fresh QA evidence in `.compozy/tasks/harness/qa/verification-report.md` and store screenshots or issue files under the same artifact root when applicable
- MUST fix root-cause regressions and add or update the narrowest durable regression coverage for any discovered bug
- MUST rerun the repository verification gates after the last fix, including the integration lanes that now prove harness behavior
- SHOULD include web/browser validation only if the changed branch exposes a user-visible surface directly impacted by the harness work; daemon/runtime proof remains mandatory regardless
</requirements>

## Subtasks
- [x] 10.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/harness` and derive the execution matrix from `task_09` artifacts
- [x] 10.2 Run the baseline repository verification gate and establish the runtime/integration starting point
- [x] 10.3 Execute real startup, augmentation, transcript, detached completion, and reentry scenarios through repo-supported surfaces
- [x] 10.4 Fix root-cause regressions, add matching regression coverage, and rerun impacted scenarios
- [x] 10.5 Rerun final verification gates and publish `.compozy/tasks/harness/qa/verification-report.md`

## Implementation Details

See TechSpec "Workstream 6: Storage, Observability, and Verification" and the QA artifacts from `task_09`. The main constraint is that harness QA must prove the runtime behavior end to end through existing project lanes such as daemon integration, transport parity, transcript replay, and task-runtime recovery instead of relying only on isolated unit tests.

### Relevant Files
- `.agents/skills/qa-execution/SKILL.md` - required workflow for execution matrix discovery, evidence capture, and verification reporting
- `scripts/discover-project-contract.py` - canonical project-contract discovery entrypoint required by `/qa-execution`
- `Makefile` - repository-defined `verify`, test, and integration entrypoints that must pass before completion
- `internal/daemon/daemon_automation_task_integration_test.go` - likely integration lane for detached task-runtime and completion behavior
- `internal/daemon/daemon_network_collaboration_integration_test.go` - useful runtime integration lane for startup and prompt-flow assertions
- `internal/api/httpapi/transport_parity_integration_test.go` - transport parity proof for persisted events and runtime flow visibility
- `internal/api/udsapi/transport_parity_integration_test.go` - UDS parity proof for the same harness-visible flows

### Dependent Files
- `.compozy/tasks/harness/qa/verification-report.md` - final QA evidence written by `/qa-execution`
- `.compozy/tasks/harness/qa/issues/BUG-*.md` - structured bug reports for failures discovered during execution
- `.compozy/tasks/harness/qa/screenshots/` - only when browser or visual evidence is relevant on the execution branch
- `internal/daemon/task_runtime_test.go` - natural place for narrow regression coverage if detached completion bugs are found
- `internal/session/manager_test.go` - likely regression destination for synthetic prompt and queueing bugs
- `internal/transcript/transcript_test.go` - likely regression destination for transcript trust-boundary bugs
- `internal/observe/observer_test.go` - likely regression destination for event-summary or observability bugs

### Related ADRs
- [ADR-001: Resolve Harness Behavior from Durable Session Context and Turn Origin](adrs/adr-001.md) - QA execution must validate the context-resolution matrix in runtime flows
- [ADR-002: Extend Existing Prompt Assembly and Turn Augmentation Seams with Staged Composition](adrs/adr-002.md) - QA execution must prove both startup and turn-time seams
- [ADR-003: Reuse the Task Runtime for Detached Harness Work and Policy-Based Synthetic Reentry](adrs/adr-003.md) - QA execution must prove detached completion and synthetic wake-up through the task-runtime substrate

### External References
- `.resources/openclaw/docs/concepts/qa-e2e-automation.md` - good reference for artifact discipline and realistic QA lanes
- `.resources/hermes/tests/integration/test_checkpoint_resumption.py` - strong precedent for resume/recovery regression scenarios
- `.resources/claude-code/utils/task/framework.ts` - useful inspiration for task lifecycle assertions worth proving during runtime QA
- `.resources/claude-code/tasks/LocalMainSessionTask.ts` - background completion plus foreground notification reference for runtime QA scenarios
- `.resources/openfang/docs/api-reference.md` - useful checklist source for externally inspectable eventful runtime behavior
- `.resources/openfang/crates/openfang-kernel/src/event_bus.rs` - helpful for thinking about observable internal event propagation during QA

## Deliverables
- Fresh `.compozy/tasks/harness/qa/verification-report.md` produced by `/qa-execution`
- Runtime QA evidence covering startup, augmentation, synthetic turns, detached completion, transcript trust, and observability **(REQUIRED)**
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**
- Fresh issue files and supplementary evidence under `.compozy/tasks/harness/qa/` **(REQUIRED)**
- Passing repository verification gates after the final QA fix set **(REQUIRED)**
- Fresh evidence that runtime-specific integration lanes, not only isolated unit tests, prove the harness flow **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Any new narrow regression helpers added during QA have stable deterministic assertions
  - [ ] Root-cause fixes discovered during QA gain the narrowest durable package-level regression coverage in `internal/session`, `internal/transcript`, `internal/task`, or `internal/observe` where appropriate
  - [ ] Unit-level regressions explicitly cover the bug that triggered the QA fix instead of only asserting a broader happy path
- Integration tests:
  - [ ] Startup prompt composition and section selection are proven through a real daemon/runtime scenario, not a mocked assembler-only test
  - [ ] Ordered augmentation and stored-input invariants are proven through a real prompt flow that persists events and dispatches to the driver
  - [ ] Synthetic prompt submission plus transcript and hook semantics are proven through an end-to-end runtime path
  - [ ] Detached task-run completion and synthetic reentry are proven through the task-runtime substrate and manager wake-up path
  - [ ] Event-summary visibility and HTTP/UDS transport parity remain correct after the full harness flow runs
  - [ ] `go test -tags integration ./internal/daemon ./internal/api/httpapi ./internal/api/udsapi -count=1` passes after the final QA fix set
  - [ ] `make verify` passes from a clean rerun after the final QA fix set
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- The `/qa-execution` workflow has been run explicitly with artifacts stored under `.compozy/tasks/harness/qa/`
- The harness architecture has fresh runtime evidence proving the main end-to-end flows
- Any QA failures were fixed at the source and documented with fresh evidence
- The normal repository verification gates pass with the new harness coverage in place

## Completion Notes

- Added `TestDetachedHarnessCompletionSilentPolicyRecordsDropEndToEnd` to `internal/daemon/daemon_integration_test.go` to close the silent/drop runtime coverage gap through the normal daemon integration lane.
- Published fresh QA evidence in `.compozy/tasks/harness/qa/verification-report.md`.
- Fresh final verification passed on 2026-04-18: targeted harness runtime bundles, `make test-integration`, and `make verify`.
- Documented the task/skill mismatch that still references `scripts/discover-project-contract.py`; the script is absent in this worktree, so QA used the repository-defined verification contract.
