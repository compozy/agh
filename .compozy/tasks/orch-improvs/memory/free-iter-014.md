# Task Memory: free-iter-014

## Objective Snapshot

- Implement the Phase B free slice: `Add bridge task subscription persistence and terminal notifier consumer over notification_cursors.`
- Keep the slice scoped to `internal/bridges` and `internal/store/globaldb`.
- Do not implement HTTP, UDS, CLI, OpenAPI/codegen, web, site docs, or review-gate verdict authority in this slice.

## Important Decisions

- Bridge subscription state stores delivery targets only; cursor progress remains in `notification_cursors`.
- The terminal notifier must replay durable `task_events.event_seq`; it must not use channel/thread state as replay authority.
- Cursor advancement happens only after `bridges/deliver` returns a validated ack for the accepted final event.

## Learnings

- `bridge_task_subscriptions` must remain target state only; the durable replay position belongs exclusively to `notification_cursors`.
- Terminal bridge delivery should carry two event identities: the bridge delivery event remains `final`, while the provider metadata envelope preserves the original task terminal event (`task.run_completed`, `task.run_failed`, etc.).
- Failed bridge delivery should record a cursor diagnostic without advancing `last_sequence`; this gives operators a replayable cursor state instead of losing the terminal event.

## Files / Surfaces

- Added/updated: `internal/bridges/task_subscription.go`
- Added/updated: `internal/bridges/task_notifier.go`
- Added/updated: `internal/bridges/task_notifier_test.go`
- Added/updated: `internal/store/globaldb/schema_bridge_task_subscription.go`
- Added/updated: `internal/store/globaldb/migrate_bridge_task_subscription.go`
- Added/updated: `internal/store/globaldb/global_db_bridge.go`
- Added/updated: `internal/store/globaldb/global_db_bridge_task_subscription_test.go`
- Updated: `internal/notifications/cursor.go`
- Updated: `internal/store/globaldb/global_db_notification_cursor.go`
- Updated: `internal/store/globaldb/global_db_notification_cursor_test.go`
- Updated: `internal/store/globaldb/global_db.go`

## Errors / Corrections

- No implementation failures required production-code rework in this slice.

## Ready for Next Run

- Completed and verified.

## Slice Picked

Add bridge task subscription persistence and terminal notifier consumer over notification_cursors.

## Acceptance Mapping

- Advances aggregate implementation step 7: implement `internal/notifications` and the bridge terminal notifier.
- Advances orchestration child build-order step 8: bridge task subscription store and terminal notifier consumer.
- Covers part of final test strategy: bridge terminal notification replay, duplicate suppression through cursor progress, and cursor advancement after confirmed delivery.

## Outcome

- Added `bridge_task_subscriptions` global DB schema and migration v20.
- Implemented `bridges.BridgeTaskSubscription` domain types, validation, cursor identity, routing identity, and delivery target projection.
- Implemented GlobalDB CRUD/list/delete for bridge task subscriptions.
- Implemented `TerminalTaskNotifier` replay over durable `task_events` using `notification_cursors`.
- Cursor advancement happens only after `DeliveryTransport.DeliverBridge` returns an ack that validates against the delivery event.
- Delivery failures record bounded `last_error` diagnostics without moving cursor sequence.

## Verification

- `go test ./internal/bridges -run 'TestBridgeTaskSubscriptionValidation|TestTerminalTaskNotifierDeliverDue' -count=1` passed.
- `go test ./internal/store/globaldb -run 'TestGlobalDBBridgeTaskSubscriptionSchemaMigration|TestBridgeTaskSubscriptionSchemaStatements|TestGlobalDBBridgeTaskSubscriptionStore|TestGlobalDBNotificationCursorStore' -count=1` passed.
- `go test ./internal/notifications -count=1` passed.
- `go test ./internal/bridges -count=1` passed.
- `go test ./internal/store/globaldb ./internal/notifications -count=1` passed.
- `go test -race ./internal/bridges -run 'TestBridgeTaskSubscriptionValidation|TestTerminalTaskNotifierDeliverDue' -count=1` passed.
- `go test -race -parallel=4 ./internal/store/globaldb -run 'TestGlobalDBBridgeTaskSubscriptionSchemaMigration|TestBridgeTaskSubscriptionSchemaStatements|TestGlobalDBBridgeTaskSubscriptionStore|TestGlobalDBNotificationCursorStore' -count=1` passed.
- `go test -race ./internal/bridges -count=1` passed.
- `go test -race -parallel=4 ./internal/store/globaldb -count=1` passed in 68.282s.
- `make lint` passed with 0 issues.
- `make verify` passed: Bun lint/typecheck/test passed, Vitest 329 files / 2088 tests passed, web build passed, `golangci-lint` reported 0 issues, Go race gate completed with `DONE 8163 tests in 175.624s`, and package boundaries passed.
