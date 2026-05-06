# Task Memory: task_24

## Objective Snapshot

- Implement latest-event-sequence read models and cursor-seeded task SSE for `.compozy/tasks/orch-improvs/task_24.md`.
- Success requires `latest_event_seq` on task/run context read models, deterministic `after_sequence` vs `Last-Event-ID` precedence, read-then-stream replay coverage, regenerated contracts, and `make verify` PASS.

## Important Decisions

- `latest_event_seq` is a read projection from durable `task_events.event_seq` (`MAX(event_seq)` per task, `0` when no events), not a new mutable task column.
- Task stream replay keeps the existing subscribe-before-backlog ordering; task 24 only changes seed/cursor correctness and the read models that provide the seed.
- `Last-Event-ID` precedence is based on header presence after trimming, so `Last-Event-ID: 0` intentionally overrides `?after_sequence=N`.
- Web fixture builders now provide `latest_event_seq` defaults so generated contract types stay strict without weakening tests.

## Learnings

- `task.ContextBundle.RecentEvents` previously came from `ListTaskEvents`, which lost stream sequence. It now reads `ListTaskEventRecords(..., Descending: true)` and reverses the bounded window to keep chronological prompt order with sequence populated.
- Task dashboard/inbox surfaces consume task references and active-run cards; they need `latest_event_seq` because the web can open a task stream from those payloads.
- Running several `make` targets that invoke Mage in parallel can race on `mage_output_file.go`; rerun gate targets sequentially when validating.

## Files / Surfaces

- `internal/task`
- `internal/store/globaldb`
- `internal/situation`
- `internal/api/core`
- `internal/api/contract`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`
- `web/src/systems/tasks/mocks/fixtures.ts`
- `web/src/systems/tasks/components/tasks-multi-agent-panel.test.tsx`

## Errors / Corrections

- Initial parallel `make bun-typecheck` failed with `open ./mage_output_file.go: no such file or directory`; sequential rerun exposed the real TypeScript fixture gaps instead.
- First `make bun-test` timed out once in `packages/site/lib/static-route-metadata.test.ts`; focused rerun passed in 1.34s and full `make bun-test` passed after the fixture updates.
- `make bun-typecheck` failed after codegen because generated task reference/dashboard types require `latest_event_seq`; corrected fixture builders and task tree test helpers instead of weakening generated types.

## Ready for Next Run

- Implementation, task frontmatter, and shared workflow memory are complete.
- Passing evidence: focused Go tests for `internal/task`, `internal/store/globaldb`, `internal/api/core`, `internal/situation`; package Go tests for `internal/task`, `internal/store/globaldb`, `internal/api/core`, `internal/situation`, `internal/observe`; `make codegen`; `make codegen-check`; `make lint`; `make bun-typecheck`; `make bun-test`; `make bun-lint`.
- Final gate passed: `make verify` completed successfully with Go race `DONE 8279 tests in 135.206s` and `OK: all package boundaries respected`.
- Next task is task 25; do not rework task 24 unless a downstream diagnostic or QA task finds a regression.
