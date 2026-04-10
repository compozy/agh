# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build `internal/subprocess/` as the shared subprocess lifecycle layer for extension work, while refactoring ACP to reuse the shared launch/shutdown path without changing ACP's existing protocol behavior.

## Important Decisions
- Evaluated `sourcegraph/jsonrpc2` against the task requirements and local PRD analysis, then chose a custom minimal transport because `_protocol.md` requires strict one-object-per-line framing, silent notification drops, explicit 10 MiB frame rejection, and drain-state gating.
- Kept ACP on `coder/acp-go-sdk`; `internal/acp/client.go` now launches subprocesses through `subprocess.Launch(... DisableTransport: true)` and attaches ACP's own connection to the raw pipes.
- Health monitoring auto-starts after a successful `Initialize` handshake using the negotiated runtime intervals/timeouts, with a default unhealthy threshold of two failed probes.

## Learnings
- The old ACP stop path relied on `exec.CommandContext` cancellation, which would have made graceful shutdown semantics harder to preserve; the shared package now owns signal escalation directly.
- Forced shutdowns can surface as transport errors if the expected stop path is not normalized before transport teardown; `waitForExit` had to squash expected kill errors before closing pending transport calls.

## Files / Surfaces
- `internal/subprocess/process.go`
- `internal/subprocess/transport.go`
- `internal/subprocess/handshake.go`
- `internal/subprocess/health.go`
- `internal/subprocess/signals.go`
- `internal/subprocess/signals_unix.go`
- `internal/subprocess/signals_windows.go`
- `internal/subprocess/process_test.go`
- `internal/subprocess/process_integration_test.go`
- `internal/subprocess/helper_ignore_unix_test.go`
- `internal/subprocess/helper_ignore_windows_test.go`
- `internal/acp/client.go`
- `internal/acp/types.go`

## Errors / Corrections
- Initial unit coverage landed at 71.2%; added targeted validation/guard/healthy-path tests to reach 80.9%.
- First shutdown-escalation test exposed a production bug where expected `SIGKILL` during requested shutdown was reintroduced through the transport error path; fixed by normalizing requested-stop errors before transport shutdown bookkeeping.

## Ready for Next Run
- Verification evidence: `go test ./internal/subprocess ./internal/acp`, `go test ./internal/subprocess -coverprofile=/tmp/subprocess.cover.out -covermode=count` (80.9%), `go test -tags integration ./internal/subprocess`, `go test -tags integration ./internal/acp`, and `make verify` all passed after the final code changes.
