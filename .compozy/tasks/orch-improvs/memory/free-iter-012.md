# Task Memory: free-iter-012

## Objective Snapshot

- Implement the Phase B free slice: `Add durable notification cursor primitive and global DB migration for notification_cursors.`
- Keep the slice scoped to `internal/notifications` and `internal/store/globaldb`.
- Do not implement bridge subscriptions, transport handlers, CLI, web, or site docs in this slice.

## Important Decisions

- The cursor primitive is delivery progress only. It must not become an event bus, task queue, hook dispatcher, review bus, or task authority surface.
- Cursor identity follows `(consumer_id, stream_name, subject_id)` with empty `subject_id` representing an unscoped stream.
- Cursor advancement must be monotonic, with idempotent replay allowed only for the same sequence and delivery id.

## Learnings

- `internal/notifications` should stay a narrow cursor boundary: typed cursor service and store contract only; bridge delivery, HTTP/UDS diagnostics, and SSE seeding belong in later slices.
- Full `make verify` can surface transient cross-workspace failures that do not reproduce in focused or aggregate reruns. Treat them as blockers until isolated and followed by a fresh full gate.

## Files / Surfaces

- Added: `internal/notifications/errors.go`
- Added: `internal/notifications/cursor.go`
- Added: `internal/store/globaldb/schema_notification_cursor.go`
- Added: `internal/store/globaldb/migrate_notification_cursor.go`
- Added: `internal/store/globaldb/global_db_notification_cursor.go`
- Added: `internal/store/globaldb/global_db_notification_cursor_test.go`
- Modified: `internal/store/globaldb/global_db.go`

## Errors / Corrections

- First full `make verify` failed in `sdk/typescript/src/integration.test.ts` with a 30s timeout. The focused SDK integration test and aggregate `make bun-test` both passed, so this was not reproducible after isolation.
- Second full `make verify` failed in `internal/extension` while cleaning a test temp dir. The focused test and full `internal/extension` race package run both passed, so the failure did not reproduce after isolation.
- A fresh full `make verify` was required after both investigations and passed.

## Ready for Next Run

- Completed. Recommended next backend slices: bridge task subscriptions/terminal notifier over the cursor primitive, task-service profile/review authority, bundled orchestration skills, or API/CLI/tool surfaces after domain authority lands.

## Slice Picked

Add durable notification cursor primitive and global DB migration for notification_cursors.

## Acceptance Mapping

- Advances aggregate implementation step 7: implement `internal/notifications`.
- Advances orchestration child build-order step 7: notification cursor primitive.
- Covers part of final test strategy: monotonic advance, reset, and idempotent replay cursor tests.

## Verification Evidence

- `go test ./internal/notifications -count=1` passed.
- `go test ./internal/store/globaldb -run 'TestGlobalDBNotificationCursorSchemaMigration|TestNotificationCursorSchemaStatements|TestGlobalDBNotificationCursorStore' -count=1` passed.
- `go test -race ./internal/store/globaldb -run 'TestGlobalDBNotificationCursorSchemaMigration|TestNotificationCursorSchemaStatements|TestGlobalDBNotificationCursorStore' -count=1` passed.
- `go test ./internal/store/globaldb ./internal/notifications -count=1` passed.
- `go test -race -parallel=4 ./internal/store/globaldb -count=1` passed.
- `bunx vitest run --config vitest.config.ts src/integration.test.ts --reporter verbose` passed from `sdk/typescript` after the first full-gate timeout.
- `make bun-test` passed after the first full-gate timeout.
- `go test -race -parallel=4 ./internal/extension -run '^TestHostAPIHandlerObserveEventsReturnsFilteredEventsWithSince$' -count=1 -v` passed after the temp-dir cleanup failure.
- `go test -race -parallel=4 ./internal/extension -count=1` passed after the temp-dir cleanup failure.
- Final `make verify` passed: Bun lint/typecheck/test passed, web build completed, `golangci-lint` reported 0 issues, Go race gate completed with `DONE 8149 tests in 51.714s`, and package boundaries were respected.
