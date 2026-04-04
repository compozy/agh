# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the new `internal/cli` package for AGH v2 with Cobra commands covering daemon/session/agent/observe/whoami, using the existing `internal/udsapi` Unix-socket HTTP/SSE API as the transport.
- Deliver task-required unit and integration tests, plus task tracking/workflow-memory updates after clean verification.

## Important Decisions
- Treat `.compozy/tasks/agh-v2/_techspec.md` and ADR-007/009 as the approved design; no additional design work is needed for this task.
- Keep the transport thin: one Unix-socket HTTP client plus per-command request/response helpers instead of widening daemon/udsapi scope.
- Preserve dirty tracking files by limiting edits to task-owned memory/tracking surfaces and leaving tracking-only changes out of the auto-commit.
- Use a small local TOON-like renderer in `internal/cli/format.go` instead of importing legacy output code, keeping the v2 CLI self-contained and deterministic.
- Implement the integration harness as a real UDS-backed test daemon assembled from `session`, `observe`, `store`, and `udsapi` with a fake session driver, rather than trying to bend `internal/daemon` test hooks into task 08.

## Learnings
- `internal/cli` is currently missing entirely; `cmd/agh/main.go` still exposes only the old `start|version` shim.
- `internal/udsapi` already serves the full task 08 contract over the Unix socket, including SSE endpoints for prompt/session follow/observe follow and JSON payloads for daemon/session/agent/observe queries.
- The current UDS session list endpoint returns all known sessions because it is backed by `Manager.ListAll`; the CLI will need to apply the active-only default for `session list` client-side.
- The session stream endpoint is reliable for `wait`, but `session events --follow` is better asserted as “stream existing/new events and exit on disconnect” rather than assuming a deterministic `session_stopped` replay timing window in every integration run.
- `go test -cover -tags integration ./internal/cli` is the right coverage proof for this task because the real UDS/client command flow lives partly behind the integration-tagged harness.

## Files / Surfaces
- `cmd/agh/main.go`
- `cmd/agh/main_test.go`
- `internal/daemon/*`
- `internal/udsapi/*`
- `internal/session/*`
- `internal/observe/*`
- `internal/store/*`
- `internal/cli/*` (new)

## Errors / Corrections
- `make verify` initially failed on `errcheck` in the table/section format helpers; fixed by explicitly discarding the `fmt.Fprint*` results.
- The follow-mode integration originally asserted on `session_stopped`, but the stronger contract is stream output plus clean exit on disconnect, so the integration proof now stops the daemon to exercise the actual follow/disconnect behavior.

## Ready for Next Run
- Task 08 is complete: implementation shipped in local commit `cda1c64`, task tracking is updated on disk, and the committed state re-verified cleanly.
- Fresh verification evidence:
  - `go test -race -tags integration ./internal/cli`
  - `go test -cover -tags integration ./internal/cli` (`80.1%`)
  - `make verify`
- Tracking-only files should remain out of the commit per task instructions.
