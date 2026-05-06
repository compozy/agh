# TC-PERF-001: SSE Resume, Query Churn, And Cursor Replay

**Priority:** P1

**Objective:** Prove task event streaming and notification cursor replay remain reliable under a
busy task without excessive UI refetch churn, missed named events, or incorrect seed precedence.

**Requirements Covered:** tasks 24-27; ADR-003, ADR-005.

## Preconditions

- Isolated QA lab with a task capable of producing multiple run, review, and notification events.
- Web app running against the isolated daemon.
- Browser automation can observe network requests and EventSource frames.

## Test Steps

1. Read task detail and record `latest_event_seq`.
   **Expected:** Snapshot exposes the durable event sequence from task event projection.

2. Open a task stream with `?after_sequence=<latest_event_seq>`.
   **Expected:** Stream starts after the snapshot sequence and does not replay already-consumed
   events.

3. Reconnect with `Last-Event-ID: 0`.
   **Expected:** Header value is honored as present and replays from zero rather than falling back
   to query parameters.

4. Reconnect with a non-zero `Last-Event-ID` and a conflicting query seed.
   **Expected:** Header takes deterministic precedence.

5. Reconnect with malformed `Last-Event-ID`.
   **Expected:** Stream fails clearly instead of silently falling back to the query seed.

6. Generate task run, review, and bridge notification events while the web Orchestration tab is
   open.
   **Expected:** Named EventSource listeners receive each canonical event type and invalidate only
   the relevant task data needed to show truthful state.

7. Force a bridge cursor replay after daemon restart.
   **Expected:** Cursor resumes from stored sequence, emits no duplicate accepted-final delivery,
   and UI diagnostics refresh once durable state changes.

## Behavioral Evidence

- Event sequence values, headers, query parameters, and received event names.
- Browser network trace or request count summary while events stream.
- Cursor sequence before and after restart.
- UI state before and after stream events.

## Disruption Probes

- Burst multiple task/review/notification events in quick succession.
- Close the browser tab while the stream is active and prove cleanup removes listeners.
- Run with a task id containing URL-sensitive characters if the API permits such ids.

