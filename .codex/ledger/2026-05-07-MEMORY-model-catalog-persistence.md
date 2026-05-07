Goal (incl. success criteria):

- Implement provider model catalog SQLite persistence for Task 02: append global DB migration, transactional store APIs, required tests, tracking updates, and one local commit after clean verification.

Constraints/Assumptions:

- Must use RTK prefix for shell commands.
- Must use workflow memory files for provider-model-catalog before edits and before finish.
- Must follow cy-execute-task, cy-final-verify, agh-schema-migration, agh-code-guidelines, golang-pro, agh-test-conventions, and testing-anti-patterns.
- Must read PRD docs, TechSpec, ADRs, AGENTS/CLAUDE guidance before implementation.
- No destructive git commands without explicit permission.
- `make verify` is required before completion and before/after commit.

Key decisions:

- Add a minimal `internal/modelcatalog` package now for store contract types because Task 02 APIs need shared row/status types and Task 03 can extend them.
- Append model catalog schema as global migration v23; preserve existing v1-v22 identities.
- Add catalog schema helpers to `globalSchemaStatements` and migration helper files, matching notification cursor/resource sidecar patterns.
- Store stale filtering treats `ListOptions.IncludeStale || ListOptions.IncludeAll` as "include stale rows".

State:

- Task 02 implementation, focused validation, full pre-commit verification, self-review, tracking updates, local commit, and post-commit verification are complete.

Done:

- Read RTK.
- Read workflow shared memory and task_02 memory.
- Loaded required skill entrypoints.
- Read root/internal AGENTS/CLAUDE guidance, Task 02, `_tasks.md`, `_techspec.md`, and all provider-model-catalog ADRs.
- Read migration/code/test skill references and relevant cross-agent ledgers.
- Captured pre-change signal: no ModelCatalog tests and no current catalog persistence tables/methods.
- Added minimal `internal/modelcatalog` types/store contract.
- Added global DB model catalog schema helper, migration helper, and v23 migration entry.
- Implemented `GlobalDB.ReplaceSourceRows`, `ListRows`, and `ListSourceStatus` with provider-scoped status rows and `BEGIN IMMEDIATE` replacement transactions.
- Added migration/store tests for schema/index presence, prior-prefix upgrade, reopen stability, append-only identity, filtering, nullable defaults, provider-scoped `models_dev`, status updates, atomic reasoning effort replacement, and deterministic ordering.
- Fixed first `make verify` failure from staticcheck nil-context literal in tests.
- Focused validation passed: `go test ./internal/modelcatalog ./internal/store/globaldb -run 'TestGlobalDBModelCatalog' -count=1`, `go vet ./internal/modelcatalog ./internal/store/globaldb`, `go test ./internal/store/globaldb -coverprofile=/tmp/agh-globaldb.cover -count=1`, `go test ./internal/store/globaldb -count=1 -race`, and `make lint`.
- New `global_db_model_catalog.go` coverage is 85.1% (235/276 statements); existing package-wide `globaldb` coverage remains 77.6%.
- Full pre-commit `make verify` passed after the staticcheck fix; warnings only were Node `NO_COLOR`/`FORCE_COLOR`, Vite chunk-size, and macOS linker `-bind_at_load`.
- Updated workflow memory and Task 02 tracking files.
- Created local commit `80087d39 feat: add model catalog persistence`.
- Post-commit `make verify` passed with the same known non-blocking warnings.

Now:

- Final report.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `internal/modelcatalog`
- `internal/store/globaldb`
- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
- `.compozy/tasks/provider-model-catalog/memory/task_02.md`
- `.compozy/tasks/provider-model-catalog/task_02.md`
- `.compozy/tasks/provider-model-catalog/_tasks.md`
- `.compozy/tasks/provider-model-catalog/_techspec.md`
