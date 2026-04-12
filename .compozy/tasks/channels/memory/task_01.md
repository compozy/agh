# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement task 01: add the authoritative `internal/channels` domain package plus additive `globaldb` schema/helpers/tests for channel instances, secret bindings, routes, and ingest dedup records.
- Keep scope limited to domain validation, stable routing-key serialization/hash helpers, and persistence.

## Important Decisions

- Channel scope/status/types will live in `internal/channels` instead of reusing `internal/memory` scope types, to keep channel governance decoupled from memory storage concerns.
- Channel persistence will follow the existing `globaldb` pattern with dedicated helper methods and additive `CREATE TABLE IF NOT EXISTS` schema statements.
- Included the typed `DeliveryTarget` domain model now because `DeliveryEvent` depends on it, but kept resolver logic out of scope for task 03.
- Workspace-scoped `channel_instances` reference `workspaces(id)` with `ON DELETE CASCADE`, and child channel tables cascade from `channel_instances` to avoid orphan rows.

## Learnings

- `globaldb` currently has no channel schema or helpers; the clean insertion point is a new channel-focused store file plus additive schema statements in `global_db.go`.
- Existing store validation is local and explicit; exported domain types should normalize and validate their own fields without depending on transport DTOs.
- `make verify` initially failed because `web/node_modules` did not contain the already-declared `@tailwindcss/vite` dependency; `bun install` under `web/` resolved the environment and the full verification gate passed without additional code changes.

## Files / Surfaces

- `internal/channels/`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/store/globaldb/global_db_extra_test.go`
- `internal/store/globaldb/global_db_channel.go`
- `internal/store/globaldb/global_db_channels_test.go`
- `internal/store/globaldb/global_db_channels_integration_test.go`

## Errors / Corrections

- Corrected the dedup expiry tests to use valid records that become expired at lookup time, instead of invalid records where `expires_at <= received_at`.
- Installed existing web dependencies so `make verify` could load `@tailwindcss/vite` during the web formatting/build steps.

## Ready for Next Run

- Task 01 is implemented and verified.
- Verification evidence:
  - `go test ./internal/channels ./internal/store/globaldb`
  - `go test -cover ./internal/channels ./internal/store/globaldb` with `83.8%` for `internal/channels` and `80.0%` for `internal/store/globaldb`
  - `go test -tags integration ./internal/store/globaldb`
  - `make verify`
