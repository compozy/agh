# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the SQLite-backed extension registry for task_05: `extensions` schema in global DB, `internal/extension/registry.go`, CRUD operations, checksum verification, and required tests.
- Finish only after clean targeted verification, >=80% `internal/extension` coverage, integration-tag lifecycle coverage, and `make verify`.

## Important Decisions
- Registry checksum verification hashes the full extension directory artifact via `ComputeDirectoryChecksum`; `manifest_path` stores the resolved manifest file inside that directory.
- `Registry.Install` defaults persisted source to `SourceUser` because the required public signature has no source argument; package-local `installWithSource` covers non-user tiers for internal callers/tests.
- Capabilities and actions persist as JSON-encoded `CapabilitiesConfig` / `ActionsConfig` values and are normalized on both write and read.

## Learnings
- `internal/store/globaldb` had no raw DB accessor, so registry tests use a temp SQLite database with the registry schema applied locally, while `globaldb` tests assert the real daemon schema and idempotent reopen path.
- The first coverage pass landed at 75.8%; targeted helper and error-path tests raised `internal/extension` package coverage to 81.7%.

## Files / Surfaces
- `internal/extension/capability.go`
- `internal/extension/registry.go`
- `internal/extension/registry_test.go`
- `internal/extension/registry_integration_test.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_test.go`
- `.compozy/tasks/ext-architecture/task_05.md`
- `.compozy/tasks/ext-architecture/_tasks.md`

## Errors / Corrections
- Coverage initially missed the task target; fixed by adding direct tests for registry helper/error branches instead of weakening the requirement.

## Ready for Next Run
- Verification evidence:
- `go test ./internal/extension ./internal/store/globaldb -count=1`
- `go test -tags integration ./internal/extension -count=1`
- `go test ./internal/extension -coverprofile=/tmp/internal-extension-task05.cover.out -covermode=count -count=1` → `81.7%`
- `go vet ./internal/extension ./internal/store/globaldb`
- `make verify`
