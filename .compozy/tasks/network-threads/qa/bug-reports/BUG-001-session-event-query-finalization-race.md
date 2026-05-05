# BUG-001: Session event queries can race with session recorder finalization

## Status

Fixed.

## Severity / Priority

- Severity: High
- Priority: P0

## Originating Test Cases

- `SMOKE-001`
- `TC-REG-001`

## Confirmed Failure

Command:

```bash
make test-e2e-runtime
```

Initial evidence:

- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/test-e2e-runtime.log`
- Retained E2E artifacts: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/agh-e2e-testdaemone2eacpmockpermissiondisconnectprojectsruntimefailure-2539771183`

Observed failure:

```text
CaptureSessionEvents() error = GET http://unix/api/sessions/sess-d61b503253b1f5be/events status 500: {"error":"session: query events for \"sess-d61b503253b1f5be\": store: query session events: sql: database is closed"}
```

## Root Cause

`session.Manager.openQueryRecorder` returned an active session recorder even when the session had already entered `finalizing`.
During fault-path session shutdown, finalization closes the per-session SQLite recorder before removing the active session from the manager map.
That allowed `/api/sessions/:id/events` to obtain a recorder that was concurrently closing, which surfaced as `sql: database is closed`.

The per-session SQLite query path also did not hold the recorder acceptance lock while reading, so a query could race directly with `SessionDB.Close`.

## Fix

- `internal/session/query.go`: wait for in-flight finalization before returning an active recorder, then reopen the persisted events database for stopped/finalized sessions.
- `internal/store/sessiondb/session_db.go`: guard event and hook-run queries with the same accept lock/state check used around recorder shutdown.
- `internal/session/query_test.go`: added a regression that proves finalizing sessions do not return the active recorder.
- `internal/store/sessiondb/session_db_extra_test.go`: added a closed-query assertion for the recorder lifecycle.

## Verification

Commands:

```bash
go test -race -count=1 ./internal/session
go test -race -count=1 ./internal/store/sessiondb
make test-e2e-runtime
```

Passing evidence:

- `go test -race -count=1 ./internal/session`: PASS
- `go test -race -count=1 ./internal/store/sessiondb`: PASS
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/test-e2e-runtime-after-fix.log`: PASS (`internal/daemon`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/testutil/e2e`)
