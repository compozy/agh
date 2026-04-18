# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add an observer-owned dashboard read model in `internal/observe` for task totals, queue/backlog state, filtered health summary, active-run cards, and explicit freshness metadata without expanding manager point reads.
- Implemented via `Observer.QueryTaskDashboard`, reusing snapshot-backed summary/metrics/health logic and shaping a dashboard-specific payload for later transport/UI tasks.

## Important Decisions
- Keep the dashboard payload dashboard-specific rather than embedding raw `Summary`, `TaskMetrics`, or `TaskHealth` structs, so later transports/UI can consume stable card-oriented fields without leaking observer internals.
- Reuse observer snapshot loading plus existing summary/metrics/health helpers; refactor task-health assembly so dashboard queries can compute filtered health from the same snapshot path.
- Treat freshness as snapshot metadata (`observed_at`, latest activity, age, stale threshold/state) and treat backlog warning as queue-age thresholding, both configured inside `observe` instead of pushed to the frontend.
- Add observer-owned defaults for dashboard shaping in `observe.Observer`: active-run limit `4`, backlog warning threshold `10m`, and stale threshold `2m`.

## Learnings
- `internal/observe` reads durable task state directly from the registry, so manager-enriched task summaries such as `ActiveRun` are not available here; dashboard active-run cards must be shaped from persisted runs/tasks on the read side.
- Existing audit-derived metrics are only precisely filterable by channel/origin, so the dashboard payload should favor metrics derived from the filtered task snapshot for workspace/scope-safe aggregates.
- Real persisted lifecycle tests need the observer clock aligned with the task-manager clock, otherwise freshness and queue-age assertions drift during integration coverage.

## Files / Surfaces
- `internal/observe/observer.go`
- `internal/observe/tasks.go`
- `internal/observe/tasks_test.go`
- `internal/observe/tasks_integration_test.go`
- `.codex/ledger/2026-04-17-MEMORY-task-05-dashboard.md`

## Errors / Corrections
- Seed the unit-test observer health session registry for live-session fixtures so dashboard health coverage does not report false orphan-run warnings.
- Set `MaxAttempts: 1` on the failed integration fixture so lifecycle reconciliation persists a terminal failed status instead of recycling the task to `ready`.

## Ready for Next Run
- Verification evidence:
  - `go test ./internal/observe`
  - `go test -tags integration ./internal/observe`
  - `go test ./internal/observe -coverprofile=/tmp/task_05.observe.unit.cover`
  - `go test -tags integration ./internal/observe -coverprofile=/tmp/task_05.observe.cover`
  - `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./internal/observe/...`
  - `make verify`
- Coverage achieved:
  - unit package coverage `84.4%`
  - integration-tag package coverage `85.7%`
