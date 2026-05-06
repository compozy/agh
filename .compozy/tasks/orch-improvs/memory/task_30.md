# Task Memory: task_30

## Status

Completed 2026-05-05 through the mandatory Compozy → Claude Opus docs delegation lane.

## Objective Snapshot

- Author durable institutional lessons from the orchestration-improvements workstream and align
  the glossary only where the implemented runtime introduced canonical terms.
- Required deliverables: numbered `docs/_memory/lessons/L-NNN-*.md` files with confirmed root
  cause + fix + evidence; updated `docs/_memory/lessons/README.md`; targeted `docs/_memory/glossary.md`
  alignment where canonical terms now exist in code, contracts, CLI, web, and docs.

## Important Decisions

- Authored four lessons (`L-017` through `L-020`) — all backed by confirmed evidence in
  workstream memory, ADRs, code paths, and tests. No speculative warnings.
- Glossary updated under `## Autonomy` with seven canonical terms now load-bearing across
  runtime/contract/CLI/web/docs surfaces: Task Execution Profile, Notification Cursor, Bridge
  Task Subscription, Run Review, Continuation Run, Task Context Bundle, Current Run ID. The
  additions reuse existing implementation surfaces and ADR-derived language; no new aspirational
  vocabulary was introduced.
- Race/full-gate flake notes from earlier tasks were considered but rejected as lesson-worthy
  because no confirmed root cause exists for them.
- The `truthful UI` posture from task 27 was rejected as a new lesson because it duplicates
  Standing Directive `SD-007 — Truthful UI > Plausible UI`. Lessons must not duplicate
  standing directives.

## Lesson Selection Rationale

| Lesson  | Class                            | Confirmed evidence                                                                                                                                                  |
| ------- | -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| L-017   | Frontend / SSE                   | `web/src/systems/tasks/hooks/use-task-stream.ts` named-listener loop; `internal/api/core/sse.go:54-60` named-event emit; task_26 audit + corrected hook tests.     |
| L-018   | Documentation / Spec authoring   | task_29 audit: invented review event names, broken `/runtime/core/agent/context` link, no-route outcome misnamed `error`, false UI placement claims, fixed tests.   |
| L-019   | Architecture / Persistence       | ADR-003; `internal/store/globaldb/global_db_bridge.go:985-1014`; `global_db_bridge_task_subscription_test.go:147-210` preserves stale cursor diagnostics on delete. |
| L-020   | Architecture / Code style        | Recurring `gocritic hugeParam` corrections across free slices 020/026/030/032/036 plus task_22; `Run.Review *RunReviewLineage` nested optional pointer pattern.       |

## Glossary Alignment Decision

Updated `docs/_memory/glossary.md` because the orch-improvs workstream shipped seven canonical
terms across runtime/contract/CLI/web/docs surfaces:

- **Task Execution Profile** — `task.ExecutionProfile`, `task_execution_profiles`,
  `[task.orchestration.profile]`, `agh task profile`, `/api/tasks/{id}/profile`,
  Orchestration tab, narrative docs.
- **Notification Cursor** — `internal/notifications`, `notification_cursors`, monotonic
  advance, identity `(consumer_id, stream_name, subject_id)`.
- **Bridge Task Subscription** — `bridge_task_subscriptions`, public
  `/api/tasks/{id}/notifications/bridges` route, web bridge-notifications card.
- **Run Review** — `task_run_reviews`, `task.Service.RecordRunReview`, status set
  `requested|routed|in_review|recorded|circuit_opened|canceled`, outcome set
  `approved|rejected|blocked|error|timeout|invalid_output`.
- **Continuation Run** — `task_runs.review_id`-linked rejected-review continuation.
- **Task Context Bundle** — `task.ContextBundle` / `/agent/context.task.bundle`.
- **Current Run ID** — `tasks.current_run_id` denormalized read projection (ADR-005).

Each entry restates the authoritative boundary for the term and matches existing implemented
behavior. No aspirational vocabulary added.

## Files / Surfaces Touched

- `docs/_memory/lessons/L-017-named-sse-listener-registration.md` (new)
- `docs/_memory/lessons/L-018-delegated-docs-runtime-truth-audit.md` (new)
- `docs/_memory/lessons/L-019-diagnostic-data-outlives-primary-record.md` (new)
- `docs/_memory/lessons/L-020-dense-typed-records-need-pointer-boundaries.md` (new)
- `docs/_memory/lessons/README.md` (index extended with L-017..L-020)
- `docs/_memory/glossary.md` (Autonomy section extended with seven canonical terms)
- `.compozy/tasks/orch-improvs/task_30.md` (subtasks/completion evidence/status)
- `.compozy/tasks/orch-improvs/_tasks.md` (task_30 row → completed)
- `.compozy/tasks/orch-improvs/memory/MEMORY.md` (cross-task durable record)
- `.compozy/tasks/orch-improvs/memory/task_30.md` (this file)

## Errors / Corrections

- Race/full-gate flake notes were considered but rejected as not lesson-worthy because the
  workstream memory does not confirm a root cause.
- The task 27 "truthful UI" lesson candidate was rejected because `docs/_memory/standing_directives.md`
  already contains `SD-007 — Truthful UI > Plausible UI`; this task should not duplicate standing rules.
- Local audit corrected the first delegated L-020 evidence wording: it mixed historical free-mode
  slice numbers with formal task numbers and cited a nonexistent `internal/task/manager_profile_native.go`
  file. The corrected evidence now names `free-iter-020`, `free-iter-026`, `free-iter-030`,
  `free-iter-032`, `free-iter-036`, `task_22`, `internal/daemon/native_profile_tools.go`,
  `internal/cli/client.go`, `internal/cli/task.go`, and `internal/api/contract/tasks.go`.
- A focused Claude correction attempt was blocked by provider rate limiting before edits:
  `ACP error -32603: Internal error: You've hit your limit · resets 8:20pm (America/Sao_Paulo)`.
  The local correction above was kept to audited wording only and is covered by fresh validation.

## Verification Evidence

- `compozy tasks validate --name orch-improvs --format json` PASS.
- `git diff --check` clean.
- `make verify` PASS — see Completion Evidence in `task_30.md`.

## Ready for Next Run

- Task 30 complete. Next loop step is `task_31` (QA Plan and Test Coverage).
- The lessons index is ready for any follow-on task to read before authoring new
  `docs/_memory/lessons/L-NNN-*.md` files.
