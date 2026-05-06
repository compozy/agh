# BUG-008: Session Events Could Read a Closed Recorder During Finalization

**Severity:** High  
**Priority:** P1  
**Type:** Functional  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** not applicable
- **URL:** `GET /api/sessions/{id}/events` over UDS during daemon E2E
- **Live provider/LLM:** acpmock-backed daemon E2E

## Summary

During ACP crash finalization, `GET /api/sessions/{id}/events` could receive an active session's closed recorder handle and fail with `sql: database is closed`.

## Behavioral Impact

- **Operator/User Goal:** Operators cannot reliably inspect events immediately after a crashed agent session.
- **Agent Behavior:** Runtime failure projection is persisted, but the inspection endpoint can race with session finalization.
- **Business Outcome:** Crash diagnostics and QA artifacts become flaky at the exact moment they are most needed.
- **Cross-Surface State:** Transcript could be captured while events failed through UDS.

## Reproduction

```bash
make test-e2e-runtime
```

Observed before the fix:

- `TestDaemonE2EACPmockCrashMidStreamProjectsRuntimeFailure` failed with `session: query events ... sql: database is closed` while capturing session events after a crash.

## Expected

Event queries during finalization must wait for the finalization barrier, then reopen the persisted session database from disk instead of using an active but closed recorder handle.

## Root Cause

`session.Manager.openQueryRecorder` checked the active session map before checking `m.finalizing`. During `finalizeStopped`, the session remains active while `closeSessionRecorder` closes the recorder and only then clears it. A concurrent query could return that closed handle.

## Fix

`internal/session/query.go` now waits on the finalization barrier before reusing an active recorder. `internal/session/query_test.go` adds a regression that leaves a closed recorder handle on an active finalizing session and proves queries block until finalization completes.

## Verification

- `go test ./internal/session -run '^TestManagerOpenQueryRecorderValidationAndCleanup$' -count=1`
- `go test -race ./internal/session -run '^TestManagerOpenQueryRecorderValidationAndCleanup$' -count=1`
- `go test -race -parallel=4 -count=5 -tags integration -run '^TestDaemonE2EACPmockCrashMidStreamProjectsRuntimeFailure$' ./internal/daemon`
- `make test-e2e-runtime`
- Final `make verify`

## Impact

- **Users Affected:** Operators inspecting crash diagnostics and tests capturing E2E artifacts.
- **Frequency:** Timing-dependent under crash finalization.
- **Workaround:** None.

## Related

- Test Case: TC-SCEN-001

