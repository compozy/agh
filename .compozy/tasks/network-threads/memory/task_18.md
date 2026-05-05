# Task Memory: task_18.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Generate behavior-first QA planning artifacts for Network Threads under `.compozy/tasks/network-threads/qa/`, without executing live QA flows in this task.

## Important Decisions
- Treat smoke/readiness as entry criteria only; P0 release-grade evidence must come from public-thread, direct-room, summarize-back, and direct-resolve-race operator journeys.
- Keep the execution path fixed to `qa-output-path=.compozy/tasks/network-threads` so task_19 can consume the artifacts directly.
- Live provider-backed behavior is required when reachable; if unavailable, task_19 must record the exact provider/tool/credential boundary instead of claiming live agent proof.

## Learnings
- `detect-phase.py` classified `task_18` as Phase B because the generated task uses `type: docs` and a "QA plan" title rather than a `qa-report` type/title; state still needs `qa.report_done=true` after this task because the marker and body are the QA report task.
- Task 17 memory provides deterministic harness commands for task_19: `make test-e2e-runtime`, `make test-e2e-web`, and `make verify`.

## Files / Surfaces
- `.compozy/tasks/network-threads/qa/test-plans/network-threads-test-plan.md`
- `.compozy/tasks/network-threads/qa/test-plans/network-threads-regression.md`
- `.compozy/tasks/network-threads/qa/test-cases/SMOKE-001.md`
- `.compozy/tasks/network-threads/qa/test-cases/TC-SCEN-001.md`
- `.compozy/tasks/network-threads/qa/test-cases/TC-SCEN-002.md`
- `.compozy/tasks/network-threads/qa/test-cases/TC-SCEN-003.md`
- `.compozy/tasks/network-threads/qa/test-cases/TC-INT-001.md`
- `.compozy/tasks/network-threads/qa/test-cases/TC-UI-001.md`
- `.compozy/tasks/network-threads/qa/test-cases/TC-REG-001.md`

## Errors / Corrections
- Initial structural validation found the three real-scenario cases lacked an explicit `Behavioral Evidence` section, and the UI case lacked an explicit `Disruption Probes` heading. Added those sections before verification.

## Ready for Next Run
- Task 19 should run `/qa-execution` with `qa-output-path=.compozy/tasks/network-threads`.
- Start with the generated regression order: SMOKE-001, TC-SCEN-001, TC-SCEN-002, TC-SCEN-003, TC-INT-001, TC-UI-001, TC-REG-001, then `make test-e2e-runtime`, `make test-e2e-web`, and final `make verify`.
- Task 18 verification evidence: structural artifact checks passed; `make verify` passed with Bun tests `2217`, Go lint `0 issues`, Go tests `8400`, and boundaries OK.
