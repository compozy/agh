# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Build the canonical raw desired-state persistence kernel under `internal/resources` for Task 01.
- Scope is limited to raw persistence, authority enforcement, SQLite schema, snapshot semantics, and required tests.
- Targeted validation and `make verify` now pass; only tracking and commit bookkeeping remain.

## Important Decisions

- Treat the approved PRD + TechSpec + ADR set as the design source of truth for this implementation run.
- Keep typed codecs, transport handlers, and projector wiring out of scope for this task.
- Implement `internal/resources` as a raw SQLite-backed kernel over `*sql.DB`, with `globaldb` responsible for schema/bootstrap only.
- Add explicit source-session activation/reset APIs now so nonce liveness and source reset semantics are available before the extension handshake task lands.
- Use per-source in-process locking plus SQLite `BEGIN IMMEDIATE` transactions for snapshot/session transitions.
- Reject direct CRUD from extension actors and require snapshot records to use `expected_version = 0`; extension sequencing flows through `source_version` instead.

## Learnings

- `internal/resources` does not exist yet; this task starts from a clean package boundary.
- `globaldb` already centralizes deterministic schema creation and migration helpers; the new tables should follow that pattern instead of introducing a parallel database bootstrap path.
- Existing `globaldb` code already uses dedicated connections plus `BEGIN IMMEDIATE` for serialized transactional paths, which fits snapshot/source-state updates well.
- Repo-wide verification also required a small lint-only constant extraction in `internal/daemon/task_runtime.go`; no task scope expansion was needed.
- `internal/resources` package coverage reached `82.2%`, and the full repository verification gate passed after the raw-kernel changes landed.

## Files / Surfaces

- `internal/resources/` (new package)
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_resources_test.go`
- `internal/store/globaldb/global_db_resources_integration_test.go`
- `internal/daemon/task_runtime.go`
- `.compozy/tasks/extensibility-parity/task_01.md`
- `.compozy/tasks/extensibility-parity/_tasks.md`
- `.compozy/tasks/extensibility-parity/_techspec.md`
- `.compozy/tasks/extensibility-parity/adrs/adr-001.md`
- `.compozy/tasks/extensibility-parity/adrs/adr-004.md`
- `.compozy/tasks/extensibility-parity/adrs/adr-005.md`
- `.compozy/tasks/extensibility-parity/adrs/adr-007.md`
- `.compozy/tasks/extensibility-parity/adrs/adr-008.md`

## Errors / Corrections

- Existing untracked files under `.compozy/tasks/extensibility-parity/` were present before implementation; avoid touching unrelated workflow artifacts.
- An interrupted lint refactor temporarily left `internal/resources/kernel.go` with missing helper functions; the helper split was restored without changing store semantics.

## Ready for Next Run

- Task 01 is complete. The next task can start from the persisted raw kernel, schema, and verified snapshot/session authority boundary.
