# ADR-003: Introduce Shared Durable Notification Cursors

## Status

Accepted

## Date

2026-05-05

## Context

The selected MVP includes a notifier cursor. AGH already has task events, hook observers, bridge delivery state, and SSE cursor semantics, but there is no shared durable primitive for notification delivery cursors that can be reused by bridge/thread/notifier consumers.

Storing the cursor only inside `internal/bridges` would make the design too narrow and would force other delivery consumers to duplicate cursor logic. At the same time, the cursor must remain a read-position/delivery concern, not a source of task ownership or terminal state.

## Decision

Create a new shared `internal/notifications` primitive for durable notification cursors.

`internal/notifications` is a durable cursor primitive only. It does not own task authority, hook dispatch, queue semantics, event fan-out policy, or terminal state. The first concrete MVP consumer is bridge-delivered terminal task notifications owned by `internal/bridges`.

The primitive should provide:

- A typed cursor model keyed by delivery consumer/subscription identity.
- Durable storage in a global DB side table.
- Read, advance, and reset semantics appropriate for confirmed delivery progress.
- Bounded metadata for diagnostics, such as latest delivered sequence, latest delivery time, latest error summary, and retry state if needed by the TechSpec.
- A concrete MVP integration point for bridge-delivered terminal task notifications. Any later thread or task/event notifier must arrive through a separate TechSpec and must not turn this primitive into an event bus.

The cursor table shape is part of the decision:

- Table name: `notification_cursors`.
- Primary key: `(consumer_id, stream_name, subject_id)`.
- `subject_id` is `TEXT NOT NULL DEFAULT ''` so unscoped streams do not rely on nullable composite-key behavior.
- `last_sequence` is monotonic and indexed with `(stream_name, last_sequence DESC) WHERE last_sequence > 0` for stream resume/lag reads.
- `last_delivery_id` stores the last confirmed delivery id for idempotent replay checks.
- Independent consumers must use distinct `consumer_id` values so one bridge/thread subscription cannot block another.

Cursor advancement rules:

- `Advance` must reject non-monotonic updates with a typed error.
- `Advance` may accept idempotent replay only when both `last_sequence` and `delivery_id` match the row's last confirmed `last_sequence` and `last_delivery_id`.
- `Reset` is the only path that may lower a cursor, and it requires an explicit recovery reason.
- Callers advance only after delivery is confirmed. If both delivery recording and cursor advancement are SQLite writes, they occur in the same global DB transaction.
- External delivery remains at-least-once: crash after external delivery but before cursor advance may duplicate delivery, but it must not skip events.

The first MVP consumer uses a bridge-owned subscription table, separate from cursor state:

- `bridge_task_subscriptions` defines the target.
- `notification_cursors` defines confirmed delivery progress for the subscription.
- Cursor key shape: `consumer_id = "bridge_task_subscription:<subscription_id>"`, `stream_name = "task_events"`, `subject_id = <task_id>`.

The bridge terminal task notifier:

1. Loads active `bridge_task_subscriptions`.
2. Replays durable `task_events` with `event_seq > cursor.last_sequence`.
3. Filters only `task.run_completed`, `task.run_failed`, `task.run_canceled`, `task.run_review_approved`, and `task.canceled`.
4. Reloads current task state and review rollup state before delivery.
5. Sends one final message through `bridges/deliver` directly only when the replayed event represents the accepted final terminal result. For review-gated runs, `task.run_review_approved` is the accepted-final delivery event; earlier run-level terminal events may be deferred or superseded by continuation.
6. Advances the cursor only after confirmed delivery of the accepted-final event.

The notifier distinguishes three replay decisions:

- `deliver`: the event is the accepted final terminal result and the current task terminal state agrees.
- `defer`: a run-level terminal event belongs to review/continuation work that is still pending or has been superseded. The notifier does not deliver and does not emit mismatch for that event; it may continue scanning later events and advance only after a later accepted-final event is delivered.
- `mismatch`: an event claims to be the accepted final terminal result but current task/review state disagrees. The notifier fails closed: it does not deliver, does not advance the cursor, records a bounded `last_error`, and emits `notification.terminal_state_mismatch`.

Recovery uses the explicit cursor `Reset` path or a task-service repair that makes accepted-final task event replay and current task state agree.

Hook or `EventObserver` wake-up is only a nudge; replay authority remains durable `task_events.event_seq`. The notifier must not use channel/thread state as replay authority and must not use the prompt/session `DeliveryBroker` as its primary consumer path.

The notification cursor must not:

- Assign tasks.
- Claim task runs.
- Complete or fail task runs.
- Replace SSE replay cursors.
- Replace task hooks.
- Treat channel messages as task authority.
- Define bridge delivery targets.

Notification delivery consumers should read task/event streams through existing authoritative stores and advance their cursor only after delivery is confirmed.

## Consequences

### Positive

- Prevents bridge-specific cursor logic from becoming the only delivery model.
- Provides one place to test cursor monotonicity, idempotency, reset, and failure accounting.
- Supports future notifier surfaces without adding another event bus.
- Keeps delivery progress separate from task authority.

### Negative

- Adds a new internal package and migration surface.
- Requires clear naming to avoid confusion with SSE `after_sequence`.
- Requires integration tasks to define which consumers are in MVP.

### Risks

- If cursor advancement is not transactional with delivery confirmation, consumers may drop or duplicate notifications.
- If cursor identity is too broad, independent consumers may block each other.
- If cursor APIs are too generic, they can become an accidental event-bus abstraction.
- Accepted-final terminal-event/current-state mismatches can stall one subscription until reset or repair. This is intentional fail-closed behavior because delivering a wrong final notification is worse than requiring operator recovery.
- Review-gated tasks require the notifier to understand deferred and superseded run terminal events so it does not deliver before review approval or stall forever on a rejected run.

## Rejected Alternatives

### Put notifier cursors only under `internal/bridges`

Rejected because the selected design is broader than bridges and should support notification consumers as a reusable runtime primitive.

### Reuse SSE client cursors as durable notification cursors

Rejected because SSE cursors are client replay positions, while notification cursors are daemon-side confirmed delivery state.

### Generic event bus

Rejected because AGH already uses typed task hooks and daemon composition-root observers. The notification cursor is delivery progress state, not a new event architecture.

## References

- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-dispatcher.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-dashboard.md`
- `internal/task/live_types.go`
- `internal/task/hooks.go`
- `internal/bridges/`
- `internal/api/core/sse.go`
