# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Split persistence ownership into `internal/store/sessiondb` and `internal/store/globaldb`, reduce `internal/store` to shared helpers/types/interfaces, and move runtime consumers to the new package boundaries.
- Required closeout evidence for this task is now captured with fresh `make verify`, `make test-integration`, and direct package coverage runs for the touched persistence/runtime packages.

## Important Decisions
- Kept `internal/store` as the shared primitives package only: narrow interfaces, validation/types, timestamp/SQL helpers, schema execution, SQLite open/recovery utilities, and session metadata helpers.
- Moved concrete SQLite ownership fully into `internal/store/sessiondb` and `internal/store/globaldb` without leaving compatibility wrappers in `internal/store`.
- Added focused package-local tests in `store`, `sessiondb`, and `globaldb` to meet the explicit `>=80%` coverage target instead of relying only on repo gates.

## Learnings
- `make test-integration` surfaced a real duplicate-test-symbol issue after the package move because `_test.go` and `_integration_test.go` compiled together under the integration tag.
- Staticcheck rejects literal `nil` contexts in tests even when the production code is intentionally guarding against nil contexts; indirect nil-context helpers are required to exercise those branches cleanly.
- The repo-wide gates do not emit Go coverage, so task coverage evidence has to be gathered explicitly with direct `go test -cover` commands on the touched packages.

## Files / Surfaces
- `internal/store/store.go`
- `internal/store/sql_helpers.go`
- `internal/store/sqlite.go`
- `internal/store/schema.go`
- `internal/store/store_helpers_test.go`
- `internal/store/store_extra_test.go`
- `internal/store/sessiondb/session_db.go`
- `internal/store/sessiondb/session_db_test.go`
- `internal/store/sessiondb/session_db_integration_test.go`
- `internal/store/sessiondb/session_db_extra_test.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_session.go`
- `internal/store/globaldb/global_db_workspace.go`
- `internal/store/globaldb/global_db_observe.go`
- `internal/store/globaldb/global_db_permission.go`
- `internal/store/globaldb/migrate_workspace.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/store/globaldb/global_db_extra_test.go`
- `internal/session/manager.go`
- `internal/observe/observer.go`
- `internal/daemon/daemon.go`
- `internal/cli/skill.go`
- `internal/session/manager_test.go`
- `internal/session/manager_integration_test.go`
- `internal/session/manager_stop_integration_test.go`
- `internal/observe/observer_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/cli/skill_test.go`
- `internal/cli/cli_integration_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/workspace/resolver_integration_test.go`

## Errors / Corrections
- Restored the missing `database/sql` import in `internal/store/sql_helpers.go` after the helper export refactor broke lint/typecheck.
- Removed duplicate test aliases/constants from `internal/store/sessiondb/session_db_integration_test.go` after `make test-integration` failed to build the package.
- Added focused helper/lifecycle tests after coverage initially landed below target in `internal/store`, `internal/store/sessiondb`, and `internal/store/globaldb`.
- Reworked nil-context tests to avoid staticcheck `SA1012` failures while preserving guard-clause coverage.

## Ready for Next Run
- Final validation state for this task:
  - `make verify` passed
  - `make test-integration` passed
  - `go test -cover ./internal/store ./internal/store/sessiondb ./internal/store/globaldb ./internal/session ./internal/observe ./internal/daemon ./internal/cli ./internal/workspace -count=1` passed
- Final package coverage evidence:
  - `internal/store`: `86.6%`
  - `internal/store/sessiondb`: `86.3%`
  - `internal/store/globaldb`: `80.1%`
  - `internal/session`: `81.6%`
  - `internal/observe`: `82.6%`
  - `internal/daemon`: `80.5%`
  - `internal/cli`: `80.0%`
  - `internal/workspace`: `80.1%`
