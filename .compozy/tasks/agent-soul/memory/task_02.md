# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add durable global DB persistence for immutable Soul runtime snapshots and append-only managed authoring revisions without changing prompt assembly or public API surfaces.
- Required completion evidence: migration v12, store methods, constraints/indexes/session provenance columns, focused tests, final `make verify`, tracking update, and one local commit after verification.
- Status: complete and locally committed as `538955da feat: persist soul snapshots`; post-commit `make verify` passed on 2026-05-02.

## Important Decisions
- `_techspec_soul.md` is the normative DDL source for this task.
- The snapshot table keeps deterministic query fields as columns (`workspace_id`, `agent_name`, `source_path`, `digest`, `created_at`, session references). Full read model, compact projection, validation state, config provenance, and redacted diagnostics are stored in the `profile_json` envelope because the Soul child TechSpec does not define extra authority columns for them.
- Heartbeat migration version 13 remains reserved for task_06; task_02 will only add migration version 12.

## Learnings
- Task 01 committed resolver/read-model/projection types in `internal/soul`; task 02 should persist those outputs instead of reparsing `SOUL.md`.
- The pre-change baseline had no `agent_soul_snapshots`, `agent_soul_revisions`, `SoulSnapshot`, `SoulRevision`, `add_agent_soul_snapshots`, `Version: 12`, or Heartbeat v13 symbols under `internal/store` or `internal/soul`.
- Focused non-race tests pass for `internal/soul` and `internal/store/globaldb` after adding v12 storage.
- The AGH test convention helper accepts the new Soul test file and the adjusted session scan test file; it reports pre-existing whole-file convention debt in `global_db_test.go`.
- Coverage evidence: `internal/soul` is 85.4%; `internal/store/globaldb` whole-package coverage is 77.9% because older unrelated surfaces in the same package remain below the 80% target. The new Soul storage paths are covered by focused fresh DB, reopen, constraint, cascade, rollback, resolver persistence, and error/default tests.
- Fresh pre-commit gate passed after staging: `make verify` completed with `DONE 7440 tests`, `0 issues`, and `OK: all package boundaries respected`.
- Post-commit gate passed after `538955da`: `make verify` completed with `DONE 7440 tests`, `0 issues`, and `OK: all package boundaries respected`.

## Files / Surfaces
- Expected production surfaces: `internal/soul/`, `internal/store/globaldb/`, possibly `internal/store/types.go` if session provenance structs need to expose v12 columns.
- Expected tests: `internal/store/globaldb/*_test.go` and possibly `internal/soul/*_test.go` for persistence-shape helpers.
- Touched production surfaces: `internal/soul/persistence.go`, `internal/store/types.go`, `internal/store/globaldb/global_db.go`, `internal/store/globaldb/migrate_soul.go`, `internal/store/globaldb/global_db_soul.go`, `internal/store/globaldb/global_db_session.go`.
- Touched test surfaces: `internal/store/globaldb/global_db_soul_test.go`, `internal/store/globaldb/global_db_session_test.go`, `internal/store/globaldb/global_db_test.go`.

## Errors / Corrections
- Initial compile found a helper name collision with task_01's `cloneDiagnostics`; renamed the new helper to `clonePersistenceDiagnostics`.
- Existing session scanner tests assumed the pre-v12 `sessions` scan width; updated them to include Soul provenance fields and assertions.
- `make lint` found a heavy value parameter, a long line, and a context argument ordering issue; fixed by passing `*ResolvedSoul`, wrapping the long error, and reordering the test helper args.

## Ready for Next Run
- Task 02 is complete. Tracking/memory files remain uncommitted by design; implementation commit is `538955da`.
