# Task Memory: task_18.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute the autonomy MVP QA pass from task_18 using `qa-execution` with `qa-output-path=.compozy/tasks/autonomous`.
- Consume Task 17 test plans/cases, prove runtime/CLI/UDS/store/hooks/scheduler/spawn/coordinator/web/docs invariants, fix any discovered regressions at root cause, and publish fresh QA evidence under `.compozy/tasks/autonomous/qa/`.

## Important Decisions
- Use the Task 17 regression suite ordering: smoke P0 cases first (`TC-AUTO-009`, `007`, `008`, `014`, `010`, `012`, `013`, `017`), then targeted P1 lanes, then final repository/generated-contract/web/site gates.
- Treat existing dirty PRD/task/memory files as pre-existing work; do not revert or overwrite unrelated changes.

## Learnings
- Shared workflow memory says tasks 01-17 are implemented locally and Task 18 must write logs, screenshots, issues, and final report under `.compozy/tasks/autonomous/qa/`.
- Task 17 artifacts define 18 manual cases and the exact artifact layout for this QA execution.
- Baseline `make deps`, `make verify`, and `make codegen-check` passed before smoke execution.
- P0 smoke lanes passed for task execution boundary, claim/lease store/service, agent task CLI/UDS, coordination channel redaction, scheduler recovery, safe spawn, coordinator bootstrap/recovery, and combined CLI/coordinator/scheduler flow.
- Two first-pass targeted filters were too narrow (`internal/scheduler` unit and `internal/coordinator` unit); corrected by running full/exact package tests and keeping both original and corrected logs as evidence.
- P1 backend lanes passed for config, contract/OpenAPI/codegen, hooks, situation/identity, channel UX, and lineage.
- `TC-AUTO-015` exposed a Playwright harness regression: daemon-served specs branched on onboarding visibility before either onboarding or shell was ready, and the handoff spec also used a stale manual session start flow that skipped the session-create dialog.
- Fixed the web E2E regression with `web/e2e/fixtures/workspace.ts`, updated daemon-served specs to use the shared workspace helper, updated the handoff spec to submit the real session-create dialog, filed `BUG-001`, and reran the handoff spec successfully.
- `TC-AUTO-016` site docs lane passed source generation, typecheck, tests, and production build.
- `TC-AUTO-018` scope-boundary scans found no new route/system scope expansion and no implemented broad memory/network/eval/dashboard behavior; matches are explicit post-MVP boundaries or unrelated existing permission/shutdown wording.
- Final full web E2E initially exposed two additional harness regressions: `acpmock` exact user-text matching did not tolerate Task 04 situation-context prompt augmentation (`BUG-002`), and `tasks.spec.ts` expected only fallback Agents-panel states even though the current autonomy flow correctly renders an active task-bound run (`BUG-003`).
- Fixed `BUG-002` in `internal/testutil/acpmock` by canonicalizing fixture user-text matching after stripping the daemon-owned situation-context prefix; focused package test passed.
- Fixed `BUG-003` by adding Tasks multi-agent selectors and asserting the active Agents state/run link in the browser E2E; targeted automation/session/tasks Playwright rerun passed 3/3.
- Final post-fix gates passed: full daemon-served web E2E 19/19, `make verify`, `make codegen-check`, and `packages/site` source generation/typecheck/test/build.
- Added explicit `TC-AUTO-006` evidence after noticing the schema/redaction case lacked dedicated logs; schema/indexes, capability rows/channel filtering, integration reopen, and redaction boundary tests passed.
- Published `.compozy/tasks/autonomous/qa/verification-report.md` and marked task_18 plus the master task row completed.
- Local commit created: `dcb89534 test: complete autonomy qa execution`.

## Files / Surfaces
- `.compozy/tasks/autonomous/qa/test-plans/`
- `.compozy/tasks/autonomous/qa/test-cases/`
- `.compozy/tasks/autonomous/qa/logs/`
- `.compozy/tasks/autonomous/qa/screenshots/`
- `.compozy/tasks/autonomous/qa/issues/`
- `.compozy/tasks/autonomous/qa/verification-report.md`
- `internal/testutil/acpmock/fixture.go`
- `internal/testutil/acpmock/fixture_test.go`
- `web/e2e/fixtures/workspace.ts`
- `web/e2e/fixtures/selectors.ts`
- `web/e2e/automation.spec.ts`
- `web/e2e/bridges.spec.ts`
- `web/e2e/combined-flows.spec.ts`
- `web/e2e/tasks-coordinator-handoff.spec.ts`
- `web/e2e/tasks.spec.ts`

## Errors / Corrections
- Corrected smoke evidence filters that produced `[no tests to run]` for scheduler/coordinator package portions; no production defect found.
- Fixed `BUG-001` from `TC-AUTO-015`; root cause was E2E synchronization/stale flow, not autonomy production runtime behavior.
- Fixed `BUG-002`; root cause was the deterministic ACP mock driver's prompt matching contract lagging behind the live situation augmenter.
- Fixed `BUG-003`; root cause was stale browser E2E expectation after manual-first publish now creates a valid active run visible in the Agents panel.

## Ready for Next Run
- Task 18 is complete. Tracking/memory files remain local tracking artifacts; the commit intentionally includes only code fixes and required QA evidence.
