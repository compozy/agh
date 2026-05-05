# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Generate task_18-ready QA planning artifacts under `.compozy/tasks/autonomous/qa/` using `qa-output-path=.compozy/tasks/autonomous`.
- Required artifact set: feature test plan, regression suite with smoke/targeted/full lanes, manual `TC-*.md` cases, stable issue/screenshot/log/report paths, and P0/P1 traceability across tasks 01-16/TechSpec/ADRs/resource lessons.
- This is planning only; live runtime/browser/docs execution remains task_18.

## Important Decisions
- QA root remains `.compozy/tasks/autonomous/qa/`; task_18 must use the same root and write `verification-report.md` there.
- Pre-change signal: QA root contained only `.gitkeep` placeholders and no feature plan, regression suite, or manual `TC-*.md` cases.
- Manual cases will focus on real autonomy invariants: token-fenced task ownership, channel binding at run enqueue, channels as conversation-only, scheduler as wake/sweep only, coordinator singleton bootstrap, safe spawn narrowing/reaper behavior, manual-first UI labels, and docs/contract parity.

## Learnings
- Shared workflow memory says tasks 01-16 are implemented locally; task_17 should plan around those committed/tracked surfaces rather than broad post-MVP memory/network/eval scope.
- ADR-012 is the central QA risk: every coordinated workspace run needs a stable channel at enqueue, but channel messages must never mutate task ownership/status or carry raw claim tokens.
- Hermes QA artifacts use a feature plan, one regression suite, `TC-*` manual case files, and reserved `issues/`, `screenshots/`, `logs/`, and `verification-report.md` paths.
- Generated artifacts cover all task_01 through task_16 invariants with 18 manual cases: P0 lease/channel/scheduler/spawn/coordinator/E2E cases and P1 config/contracts/hooks/context/identity/UI/docs/boundary cases.
- `make verify` passed after the final artifact whitespace cleanup: web lint/test/build, Go fmt/lint/test/build, and package boundary checks completed cleanly.

## Files / Surfaces
- Planning inputs read: `_techspec.md`, `_tasks.md`, ADRs 001-012, tasks 01-16, task_17, Hermes task_10 and QA artifacts, autonomy docs/CLI references, web Tasks E2E, and required Paperclip/Hermes/Multica references.
- Outputs: `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-test-plan.md`, `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-regression.md`, and `.compozy/tasks/autonomous/qa/test-cases/TC-AUTO-001.md` through `TC-AUTO-018.md`.

## Errors / Corrections
- No planning-discovered implementation discrepancies were filed as `BUG-*` issues.
- `git diff --cached --check` initially caught trailing Markdown spaces and extra EOF blank lines in generated artifacts; corrected mechanically and reran artifact checks plus full `make verify`.

## Ready for Next Run
- QA artifact commit: `386f8493` (`test: add autonomy mvp qa artifacts`).
- Task 18 should use `/qa-execution` with the same `qa-output-path=.compozy/tasks/autonomous`.
- Start execution from `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-regression.md`.
- Write runtime logs under `.compozy/tasks/autonomous/qa/logs/`, screenshots under `.compozy/tasks/autonomous/qa/screenshots/`, issue files under `.compozy/tasks/autonomous/qa/issues/`, and the final report at `.compozy/tasks/autonomous/qa/verification-report.md`.
