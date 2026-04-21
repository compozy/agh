# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Make harness runtime decisions observable through existing `event_summaries` / `observe.QueryEvents` surfaces, then harden final-flow integration coverage.
- Completed: startup context resolution, startup section selection, prompt augmentation, detached completion, and synthetic reentry now leave durable harness summaries on the existing observe timeline, and transport/integration coverage proves the final flow stays visible.

## Important Decisions
- Reuse the existing global DB `EventSummary` write/list path instead of introducing new storage or read-side APIs.
- Keep summary payloads compact and stable so current SQL/query/SSE consumers can inspect them without schema changes.
- Queue startup summaries until `OnSessionCreated` instead of attempting pre-session writes, because `event_summaries.session_id` is validated against the global `sessions` index.

## Learnings
- Task 07 already emits `harness.detached_run_completed`, `harness.synthetic_reentry_emitted`, and `harness.synthetic_reentry_dropped`; task 08 should extend that model, not replace it.
- `observe.QueryEvents` is already a thin pass-through to `ListEventSummaries`, so visibility work mostly belongs in the writer side plus regression tests proving the current readers surface it correctly.
- The new durable startup/prompt summary types are `harness.context_resolved`, `harness.section_selected`, `harness.augmenter_applied`, and `harness.augmenter_failed`; HTTP and UDS observe streams surface them without transport-specific shaping.
- Verification evidence for this task ended green on `go test ./internal/daemon ./internal/observe ./internal/store/globaldb ./internal/api/httpapi ./internal/api/udsapi -count=1`, `go test -tags integration ./internal/daemon ./internal/api/udsapi -count=1`, `make test-integration`, and `make verify`.
- Fresh close-out verification reran `make test-integration` and `make verify` on the final staged tree before the local commit.
- Coverage for the modified read-side packages met the task bar with `internal/observe` at `82.0%` and `internal/store/globaldb` at `80.0%` when exercised with its integration lane.

## Files / Surfaces
- `internal/daemon/boot.go`
- `internal/daemon/harness_observability.go`
- `internal/daemon/section_selector.go`
- `internal/daemon/prompt_input_composite.go`
- `internal/daemon/harness_reentry_bridge.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/daemon/harness_observability_test.go`
- `internal/observe/observer_test.go`
- `internal/store/globaldb/global_db_extra_test.go`
- `internal/api/httpapi/stream_helpers_test.go`
- `internal/api/udsapi/stream_helpers_test.go`
- `internal/api/udsapi/transport_parity_integration_test.go`

## Errors / Corrections
- Initial lint/verify failures were corrected by removing unused startup-recorder context parameters, deduplicating repeated task-event strings behind constants, and splitting `promptInputComposite.Augment` into smaller helpers to satisfy `gocyclo`.

## Ready for Next Run
- No follow-up execution is required for task 08.
- Local code commit: `dd0eb036` (`feat: add harness lifecycle observability`).
- Task 09 can plan QA and regression artifacts against the existing harness summary types and current observe/query/http/uds surfaces rather than inventing new inspection endpoints.
- Unrelated local changes exist in `.agents/skills/compozy/references/config-reference.md`, `web/AGENTS.md`, `web/CLAUDE.md`, and prior task tracking files; leave them alone.
