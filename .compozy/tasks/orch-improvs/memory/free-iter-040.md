# Free Iteration 040 - Remaining task decomposition and QA tail

## Slice

Generate `_tasks.md` acceptance cross-walk with QA pair tail and Web/Docs Impact coverage for remaining orch-improvs work.

## Completed

- Added `.compozy/tasks/orch-improvs/_tasks.md` as a traceability artifact for the remaining program scope while preserving the active `state.yaml mode=free` loop state.
- Added an "Already Completed Backend Cross-Walk" mapping completed free-mode slices to the aggregate TechSpec areas they closed.
- Added six valid Compozy task files for the remaining work:
  - `task_01.md`: context bundle, SSE seed, and bridge notification transport parity.
  - `task_02.md`: web task orchestration and review surfaces.
  - `task_03.md`: packages/site narrative docs and reference co-ship.
  - `task_04.md`: durable `docs/_memory` lessons and glossary alignment.
  - `task_05.md`: QA report and test coverage planning.
  - `task_06.md`: real-scenario QA execution.
- Wired the QA tail so `task_05` depends on the last implementation/docs-memory task and `task_06` depends on `task_05`.
- Kept CodeRabbit rounds out of the task table and documented them as the post-QA Phase D loop gate so the QA pair remains the task-list tail.

## Validation

- `compozy tasks validate --name orch-improvs --format json` passed with `"ok": true` and 6 scanned task files.
- Custom structural validation passed: every task has frontmatter, required sections, and a single QA report/execution tail row in `_tasks.md`.
- `PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/cy-codex-loop/scripts/detect-phase.py orch-improvs` continued to report `phase=B action=execute_free_slice`, as expected for the existing free-mode state.
- Final `make verify` passed: Bun lint/typecheck/test passed, Vitest reported 329 files and 2088 tests passed, web build passed, `golangci-lint` reported `0 issues`, Go race gate reported `DONE 8254 tests in 14.246s`, and package boundaries passed.

## Debugging notes

- Initial task validation failed because `type: qa-report` and `type: qa-execution` are not allowed by Compozy metadata v2. Fixed by using `type: test` and making the QA report task title explicitly contain `QA Report` so the loop's QA detector can still classify it if the workflow is ever switched to tasks mode.
- This slice did not hand-edit `state.yaml.tasks.*`; `state.yaml` remains managed by the cy-codex-loop helper scripts. The new task files close the missing decomposition artifact without changing the already-active free-mode execution contract.

## Remaining

- Continue Phase B with `task_01` scope: context bundle, cursor-seeded SSE, notification subscription/cursor transport surfaces, and generated contract/CLI docs if still required.
- Then execute the web, site docs, docs-memory lessons, QA report, QA execution, and three clean CodeRabbit rounds before `deliverables_complete` can be set.

