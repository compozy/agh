# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the transport-agnostic managed Soul authoring service for `SOUL.md`: validate, put, delete, history, rollback, CAS, atomic file mutation, snapshot/revision persistence, deterministic redacted errors, and tests.

## Important Decisions
- Keep this task scoped to `internal/soul` service/domain code plus focused tests; CLI/HTTP/UDS/Host API adapters remain later tasks.
- Use existing task 01 resolver/parser (`soul.Resolve` / `soul.Parse`) for every mutation path and task 02 global DB methods for revision/snapshot persistence.
- Treat the current `.compozy/tasks/agent-soul/*` tracking changes outside task_03 as pre-existing workspace state.
- Add only one shared filesystem helper, `fileutil.AtomicRemoveFile`, so delete uses the same durable parent-directory sync discipline as atomic writes.
- Keep new fileutil tests in `internal/fileutil/atomic_remove_test.go` rather than refactoring pre-existing `atomic_test.go` debt during this task.
- The final exported service API is idiomatic for package `soul`: `AuthoringService`, `AuthoringTarget`, `PutRequest`, `DeleteRequest`, `HistoryRequest`, `RollbackRequest`, and `MutationResult`.

## Learnings
- Task 02 already provides `AppendSoulRevision`, `ListSoulRevisions`, `FindSoulRevisionForRollback`, and `UpsertSoulSnapshot`; Task 03 should coordinate them rather than adding store APIs.
- No existing `SoulAuthoringService` is present before this task.
- Validation evidence is green after the final recheck change: service tests, store integration tests, race tests, test-shape checks, `internal/soul` coverage at 80.5%, `make lint`, pre-commit `make verify`, and post-commit `make verify`.

## Files / Surfaces
- Touched: `internal/soul/authoring.go`, `internal/soul/authoring_test.go`, `internal/fileutil/atomic.go`, `internal/fileutil/atomic_remove_test.go`.

## Errors / Corrections
- `make lint` rejected stuttered exported names and long helpers; corrected by renaming the API to package-idiomatic names and splitting target/path validation helpers.
- Self-review against `_techspec_soul.md` found a missing immediate pre-mutation digest recheck; corrected by adding `verifyUnchangedSoul` to put/delete/rollback.

## Ready for Next Run
- Task 03 implementation is committed as `e881f7de feat: add managed soul authoring`.
- Tracking is updated for Task 03 but intentionally not included in the code commit.
