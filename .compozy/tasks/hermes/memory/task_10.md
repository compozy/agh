# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create Hermes hardening QA planning artifacts only. Do not execute live QA flows in this task.
- Required output root is `.compozy/tasks/hermes/qa/` with `qa-output-path=.compozy/tasks/hermes` for both planning and task_11 execution.

## Important Decisions
- Use one feature plan plus one regression suite: `qa/test-plans/hermes-hardening-test-plan.md` and `qa/test-plans/hermes-hardening-regression.md`.
- Keep manual cases P0/P1 heavy and invariant-focused rather than generic smoke checks.
- Preserve task_10's stronger added coverage notes for tasks 04, 06, 07, and 09 in the QA artifacts.

## Learnings
- `task_10.md` already contains extra explicit QA coverage for automation durability, process registry interrupts, memory visibility, and environment/extension/release hardening beyond the original prompt.
- Current repository state has existing unrelated dirty `packages/site` changes; task_10 should not edit those files, but QA planning must include site verification for the changed docs.

## Files / Surfaces
- Created QA plan and regression suite under `.compozy/tasks/hermes/qa/test-plans/`.
- Created manual cases under `.compozy/tasks/hermes/qa/test-cases/` for persistence, observability, ACP lifecycle, automation, MCP auth, symlink security, process registry, memory, CLI setup, environment/extensions, release, web, and site docs.
- Reserved task_11 evidence directories: `.compozy/tasks/hermes/qa/issues/`, `screenshots/`, and `logs/`.

## Errors / Corrections
- `make verify` passed before tracking updates.
- Final fresh `make verify` passed after tracking and memory updates: web lint had `Found 0 warnings and 0 errors`, Go lint had `0 issues`, Go tests reported `DONE 5851 tests in 5.984s`, and package boundaries were OK.
- Post-commit `make verify` passed after the pre-commit Markdown formatter wrote commit content: web lint had `Found 0 warnings and 0 errors`, Go lint had `0 issues`, Go tests reported `DONE 5851 tests in 6.086s`, and package boundaries were OK.

## Ready for Next Run
- QA artifacts, tracking updates, and memory updates are complete.
- QA artifact commit: `92adb526 test: add hermes hardening qa artifacts`.
